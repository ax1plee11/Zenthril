package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func TestPasswordRejectsUnsafeStoredParameters(t *testing.T) {
	for _, encoded := range []string{
		"$argon2id$v=19$m=65536,t=3,p=0$c29tZXNhbHRzb21lc2FsdA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY",
		"$argon2id$v=19$m=4294967295,t=3,p=4$c29tZXNhbHRzb21lc2FsdA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY",
		"$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHRzb21lc2FsdA$",
		"$other$v=19$m=65536,t=3,p=4$c29tZXNhbHRzb21lc2FsdA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY",
	} {
		if ok, err := VerifyPassword("password", encoded); ok || err == nil {
			t.Errorf("accepted unsafe hash: %s", encoded)
		}
	}
}

// Run against a dedicated disposable database, never a real application DB.
func TestAuditMFALoginAndReplayIntegration(t *testing.T) {
	url := os.Getenv("AUDIT_TEST_DB_URL")
	if url == "" {
		t.Skip("AUDIT_TEST_DB_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("AUDIT_TEST_REDIS_ADDR")})
	defer rdb.Close()
	_, err = db.Exec(ctx, `CREATE TABLE IF NOT EXISTS users(id UUID PRIMARY KEY, username TEXT UNIQUE, password_hash TEXT, public_key TEXT, created_at TIMESTAMPTZ DEFAULT NOW()); CREATE TABLE IF NOT EXISTS user_mfa_totp(user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, secret_enc TEXT NOT NULL, enabled BOOLEAN DEFAULT FALSE, created_at TIMESTAMPTZ DEFAULT NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	username := "audit-" + id
	hash, err := HashPassword("Audit-Strong-Password-2026!")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(ctx, `INSERT INTO users(id,username,password_hash,public_key) VALUES($1,$2,$3,'')`, id, username, hash)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	svc := NewService(db, rdb, "audit-local-only-secret-at-least-32-chars")
	// Accounts without MFA still work.
	if _, _, err := svc.Login(ctx, username, "Audit-Strong-Password-2026!"); err != nil {
		t.Fatal(err)
	}
	_, pair1, err := svc.Login(ctx, username, "Audit-Strong-Password-2026!")
	if err != nil {
		t.Fatal(err)
	}
	_, pair2, err := svc.Login(ctx, username, "Audit-Strong-Password-2026!")
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := svc.IssueWSTicket(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.LogoutAll(ctx, id, pair1.AccessToken); err != nil {
		t.Fatal(err)
	}
	for _, pair := range []*TokenPair{pair1, pair2} {
		if revoked, err := svc.IsTokenBlacklisted(ctx, pair.AccessToken); err != nil || !revoked {
			t.Fatalf("logout-all left access token usable: %v", err)
		}
		if _, err := svc.RefreshTokens(ctx, pair.RefreshToken); err == nil {
			t.Fatal("logout-all left refresh token usable")
		}
	}
	if _, err := svc.ConsumeWSTicket(ctx, ticket); err == nil {
		t.Fatal("logout-all left pending ticket usable")
	}
	_, newPair, err := svc.Login(ctx, username, "Audit-Strong-Password-2026!")
	if err != nil {
		t.Fatal(err)
	}
	if revoked, err := svc.IsTokenBlacklisted(ctx, newPair.AccessToken); err != nil || revoked {
		t.Fatalf("new session rejected after logout-all: %v", err)
	}
	secret, _, err := svc.SetupTOTP(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _ := base32Decode(secret)
	code, _ := computeTOTPCode(decoded, uint64(time.Now().Unix()/totpTimeStep))
	if err := svc.VerifyTOTPSetup(ctx, id, code); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Login(ctx, username, "Audit-Strong-Password-2026!"); !errors.Is(err, ErrMFARequired) {
		t.Fatalf("password-only login bypassed MFA: %v", err)
	}
	if _, _, err := svc.Login(ctx, username, "Audit-Strong-Password-2026!", "abcdef"); !errors.Is(err, ErrInvalidTOTPCode) {
		t.Fatalf("invalid code accepted: %v", err)
	}
	if _, _, err := svc.Login(ctx, username, "Audit-Strong-Password-2026!", code); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Login(ctx, username, "Audit-Strong-Password-2026!", code); !errors.Is(err, ErrInvalidTOTPCode) {
		t.Fatalf("replayed code accepted: %v", err)
	}
	if err := svc.DisableTOTP(ctx, id, ""); !errors.Is(err, ErrMFARequired) {
		t.Fatalf("disabled MFA without code: %v", err)
	}
	if err := svc.DisableTOTP(ctx, id, code); !errors.Is(err, ErrInvalidTOTPCode) {
		t.Fatalf("disabled MFA with replay: %v", err)
	}
	if _, _, err := svc.SetupTOTP(ctx, id); !errors.Is(err, ErrTOTPAlreadyEnabled) {
		t.Fatalf("overwrote active MFA: %v", err)
	}
	var storedSecret string
	if err := db.QueryRow(ctx, `SELECT secret_enc FROM user_mfa_totp WHERE user_id=$1`, id).Scan(&storedSecret); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(storedSecret, totpStoragePrefix) || strings.Contains(storedSecret, secret) {
		t.Fatal("MFA secret stored in plaintext")
	}
	// Simulate an old plaintext row: login must migrate it without breaking MFA.
	if _, err := db.Exec(ctx, `UPDATE user_mfa_totp SET secret_enc=$1 WHERE user_id=$2`, secret, id); err != nil {
		t.Fatal(err)
	}
	nextCode, _ := computeTOTPCode(decoded, uint64(time.Now().Unix()/totpTimeStep)+1)
	if err := svc.VerifyTOTP(ctx, id, nextCode); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `SELECT secret_enc FROM user_mfa_totp WHERE user_id=$1`, id).Scan(&storedSecret); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(storedSecret, totpStoragePrefix) {
		t.Fatal("legacy plaintext secret not migrated")
	}
	if _, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS password_reset_tokens(user_id UUID REFERENCES users(id) ON DELETE CASCADE,token_hash TEXT UNIQUE, expires_at TIMESTAMPTZ,used_at TIMESTAMPTZ)`); err != nil {
		t.Fatal(err)
	}
	resetToken, err := svc.RequestPasswordReset(ctx, username)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- svc.ConfirmPasswordReset(ctx, resetToken, "New-Audit-Password-2026!")
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("reset token must succeed exactly once; successes=%d", success)
	}
	remaining, err := rdb.SCard(ctx, refreshUserSetKey(id)).Result()
	if err != nil || remaining != 0 {
		t.Fatalf("password reset did not revoke refresh sessions: %v", err)
	}
}
