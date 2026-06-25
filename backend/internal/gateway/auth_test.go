package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type stubSessionValidator struct {
	blacklisted     bool
	banned          bool
	blacklistChecks int
	globalBanChecks int
}

func (s *stubSessionValidator) IsTokenBlacklisted(context.Context, string) (bool, error) {
	s.blacklistChecks++
	return s.blacklisted, nil
}

func (s *stubSessionValidator) IsGloballyBanned(context.Context, string) (bool, error) {
	s.globalBanChecks++
	return s.banned, nil
}

func TestJWTAuthenticatorRejectsBlacklistedToken(t *testing.T) {
	t.Parallel()
	validator := &stubSessionValidator{blacklisted: true}
	auth := NewJWTAuthenticator("test-secret-that-is-long-enough", validator)

	_, err := auth.AuthenticateWebSocket(context.Background(), signedGatewayToken(t, "test-secret-that-is-long-enough", "user-1"))
	if err == nil {
		t.Fatal("expected blacklisted token rejection")
	}
	if validator.blacklistChecks != 1 {
		t.Fatalf("blacklist checks = %d, want 1", validator.blacklistChecks)
	}
}

func TestJWTAuthenticatorRejectsBannedUser(t *testing.T) {
	t.Parallel()
	validator := &stubSessionValidator{banned: true}
	auth := NewJWTAuthenticator("secret", validator)

	_, err := auth.AuthenticateWebSocket(context.Background(), signedGatewayToken(t, "secret", "user-1"))
	if err == nil {
		t.Fatal("expected banned user rejection")
	}
	if validator.globalBanChecks != 1 {
		t.Fatalf("global ban checks = %d, want 1", validator.globalBanChecks)
	}
}

func TestJWTAuthenticatorSkipsValidatorForInvalidToken(t *testing.T) {
	t.Parallel()
	validator := &stubSessionValidator{}
	auth := NewJWTAuthenticator("secret", validator)

	_, err := auth.AuthenticateWebSocket(context.Background(), "not-a-jwt")
	if err == nil {
		t.Fatal("expected invalid token rejection")
	}
	if validator.blacklistChecks != 0 || validator.globalBanChecks != 0 {
		t.Fatalf("validator was called for invalid token: blacklist=%d banned=%d", validator.blacklistChecks, validator.globalBanChecks)
	}
}

func TestJWTAuthenticatorRejectsOversizedTokenBeforeValidator(t *testing.T) {
	t.Parallel()
	validator := &stubSessionValidator{}
	auth := NewJWTAuthenticator("secret", validator)

	_, err := auth.AuthenticateWebSocket(context.Background(), string(make([]byte, maxWebSocketBearerTokenBytes+1)))
	if err == nil {
		t.Fatal("expected oversized token rejection")
	}
	if validator.blacklistChecks != 0 || validator.globalBanChecks != 0 {
		t.Fatalf("validator was called for oversized token: blacklist=%d banned=%d", validator.blacklistChecks, validator.globalBanChecks)
	}
}

func signedGatewayToken(t *testing.T, secret, userID string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		UserID:   userID,
		DeviceID: "device-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "session-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
