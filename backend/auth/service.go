package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"zenthril-backend/models"
)

var ErrUsernameTaken = errors.New("username_taken")
var ErrInvalidCredentials = errors.New("invalid_credentials")
var ErrInvalidRefreshToken = errors.New("invalid_refresh_token")

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type refreshTokenRecord struct {
	UserID    string    `json:"user_id"`
	TokenID   string    `json:"token_id"`
	TokenHash string    `json:"token_hash"`
	IssuedAt  time.Time `json:"issued_at"`
}

type Service struct {
	onSessionsRevoked   func(string)
	mfaEncryptionSecret string
	db                  *pgxpool.Pool
	redis               *redis.Client
	jwtSecret           string
	accessTokenTTL      time.Duration
	refreshTokenTTL     time.Duration
}

func NewService(db *pgxpool.Pool, rdb *redis.Client, jwtSecret string, ttl ...time.Duration) *Service {
	accessTTL := DefaultAccessTokenTTL
	refreshTTL := DefaultRefreshTokenTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		accessTTL = ttl[0]
	}
	if len(ttl) > 1 && ttl[1] > 0 {
		refreshTTL = ttl[1]
	}
	return &Service{
		mfaEncryptionSecret: jwtSecret,
		db:                  db,
		redis:               rdb,
		jwtSecret:           jwtSecret,
		accessTokenTTL:      accessTTL,
		refreshTokenTTL:     refreshTTL,
	}
}

func (s *Service) Register(ctx context.Context, username, password, publicKey string) (*models.User, *TokenPair, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	var user models.User
	err = s.db.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, public_key)
		 VALUES ($1, $2, $3)
		 RETURNING id, username, public_key, created_at`,
		username, hash, publicKey,
	).Scan(&user.ID, &user.Username, &user.PublicKey, &user.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique") {
			return nil, nil, ErrUsernameTaken
		}
		return nil, nil, fmt.Errorf("insert user: %w", err)
	}

	pair, err := s.issueTokenPair(ctx, user.ID.String())
	if err != nil {
		return nil, nil, err
	}

	return &user, pair, nil
}

func (s *Service) Login(ctx context.Context, username, password string, totpCodes ...string) (*models.User, *TokenPair, error) {
	var user models.User
	var passwordHash string

	err := s.db.QueryRow(ctx,
		`SELECT id, username, password_hash, public_key, created_at
		 FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &passwordHash, &user.PublicKey, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("query user: %w", err)
	}

	ok, err := VerifyPassword(password, passwordHash)
	if err != nil {
		return nil, nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, nil, ErrInvalidCredentials
	}

	code := ""
	if len(totpCodes) > 0 {
		code = totpCodes[0]
	}
	if err := s.VerifyTOTP(ctx, user.ID.String(), code); err != nil {
		return nil, nil, err
	}
	pair, err := s.issueTokenPair(ctx, user.ID.String())
	if err != nil {
		return nil, nil, err
	}
	return &user, pair, nil
}

func (s *Service) issueTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
	// SECURITY-HARDENING: access tokens are short-lived; refresh tokens are single-use and server-tracked.
	version, err := s.sessionVersion(ctx, userID)
	if err != nil {
		return nil, err
	}
	accessToken, err := generateTypedToken(userID, "access", s.accessTokenTTL, s.jwtSecret, version)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := generateTypedToken(userID, "refresh", s.refreshTokenTTL, s.jwtSecret, version)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	_, tokenID, err := ValidateRefreshTokenWithID(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token id: %w", err)
	}
	record := refreshTokenRecord{
		UserID:    userID,
		TokenID:   tokenID,
		TokenHash: hashToken(refreshToken),
		IssuedAt:  time.Now().UTC(),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("encode refresh token record: %w", err)
	}

	key := refreshTokenKey(tokenID)
	if err := s.redis.Set(ctx, key, data, s.refreshTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	if err := s.redis.SAdd(ctx, refreshUserSetKey(userID), tokenID).Err(); err != nil {
		_ = s.redis.Del(ctx, key).Err()
		return nil, fmt.Errorf("index refresh token: %w", err)
	}
	_ = s.redis.Expire(ctx, refreshUserSetKey(userID), s.refreshTokenTTL).Err()

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
	}, nil
}

func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, tokenID, err := ValidateRefreshTokenWithID(refreshToken, s.jwtSecret)
	if err != nil {
		slog.Warn("security refresh token rejected", "reason", "invalid_jwt")
		return nil, ErrInvalidRefreshToken
	}

	c, err := parseTypedToken(refreshToken, s.jwtSecret, "refresh")
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	if err := s.validateSessionVersion(ctx, c); err != nil {
		return nil, err
	}
	key := refreshTokenKey(tokenID)
	// SECURITY-HARDENING: GETDEL makes refresh token rotation atomic. Only one request can
	// consume a refresh token; concurrent replays see a missing/used token.
	// VULNERABILITY FIXED: stolen refresh tokens cannot be reused after the first successful rotation.
	raw, err := s.redis.GetDel(ctx, key).Bytes()
	if err == redis.Nil {
		if s.wasRefreshTokenUsed(ctx, tokenID) {
			// SECURITY-HARDENING: refresh-token replay indicates theft; revoke the user's active refresh set.
			slog.Warn("security refresh token replay detected", "user_id", userID, "token_id", tokenID)
			_ = s.revokeUserRefreshTokens(ctx, userID)
		}
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, fmt.Errorf("check refresh token: %w", err)
	}
	var record refreshTokenRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		_ = s.redis.Del(ctx, key).Err()
		_ = s.redis.SRem(ctx, refreshUserSetKey(userID), tokenID).Err()
		return nil, fmt.Errorf("decode refresh token record: %w", err)
	}
	if record.UserID != userID || record.TokenID != tokenID || !equalTokenHash(record.TokenHash, hashToken(refreshToken)) {
		slog.Warn("security refresh token record mismatch", "user_id", userID, "token_id", tokenID)
		_ = s.revokeUserRefreshTokens(ctx, userID)
		return nil, ErrInvalidRefreshToken
	}

	_ = s.redis.SRem(ctx, refreshUserSetKey(userID), tokenID).Err()
	// SECURITY-HARDENING: remember used refresh IDs to detect replay after rotation.
	if err := s.redis.Set(ctx, usedRefreshTokenKey(tokenID), userID, s.refreshTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("mark refresh token used: %w", err)
	}

	return s.issueTokenPair(ctx, userID)
}

func (s *Service) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	userID, tokenID, err := ValidateRefreshTokenWithID(refreshToken, s.jwtSecret)
	if err != nil {
		return nil
	}
	if err := s.redis.Del(ctx, refreshTokenKey(tokenID)).Err(); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	_ = s.redis.SRem(ctx, refreshUserSetKey(userID), tokenID).Err()
	return nil
}

func (s *Service) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken != "" {
		if _, err := ValidateToken(accessToken, s.jwtSecret); err == nil {
			// SECURITY-HARDENING: blacklist valid access tokens until natural expiry on logout.
			key := "token:blacklist:" + hashToken(accessToken)
			if err := s.redis.Set(ctx, key, "1", s.accessTokenTTL).Err(); err != nil {
				return fmt.Errorf("revoke access token: %w", err)
			}
		}
	}
	if refreshToken != "" {
		if err := s.RevokeRefreshToken(ctx, refreshToken); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) LogoutAll(ctx context.Context, userID, accessToken string) error {
	if accessToken != "" {
		if tokenUserID, err := ValidateToken(accessToken, s.jwtSecret); err == nil && tokenUserID == userID {
			// SECURITY-HARDENING: logout-all immediately rejects the current
			// access token and removes every tracked refresh token for the user.
			key := "token:blacklist:" + hashToken(accessToken)
			if err := s.redis.Set(ctx, key, "1", s.accessTokenTTL).Err(); err != nil {
				return fmt.Errorf("revoke access token: %w", err)
			}
		}
	}
	if userID != "" {
		return s.revokeUserRefreshTokens(ctx, userID)
	}
	return nil
}

func (s *Service) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := "token:blacklist:" + hashToken(token)
	val, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check blacklist: %w", err)
	}
	if val > 0 {
		return true, nil
	}
	c, err := parseTypedToken(token, s.jwtSecret, "access")
	if err != nil {
		return true, nil
	}
	version, err := s.sessionVersion(ctx, c.UserID)
	if err != nil {
		return false, err
	}
	return version != c.SessionVersion, nil
}

func (s *Service) wasRefreshTokenUsed(ctx context.Context, tokenID string) bool {
	exists, err := s.redis.Exists(ctx, usedRefreshTokenKey(tokenID)).Result()
	return err == nil && exists > 0
}

func (s *Service) revokeUserRefreshTokens(ctx context.Context, userID string) error {
	versionBytes := make([]byte, 32)
	if _, err := rand.Read(versionBytes); err != nil {
		return err
	}
	if err := s.redis.Set(ctx, sessionVersionKey(userID), hex.EncodeToString(versionBytes), s.refreshTokenTTL+s.accessTokenTTL).Err(); err != nil {
		return err
	}
	if s.onSessionsRevoked != nil {
		s.onSessionsRevoked(userID)
	}
	setKey := refreshUserSetKey(userID)
	tokenIDs, err := s.redis.SMembers(ctx, setKey).Result()
	if err != nil {
		return fmt.Errorf("list user refresh tokens: %w", err)
	}
	if len(tokenIDs) > 0 {
		keys := make([]string, 0, len(tokenIDs)+1)
		for _, tokenID := range tokenIDs {
			keys = append(keys, refreshTokenKey(tokenID))
		}
		keys = append(keys, setKey)
		if err := s.redis.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("revoke user refresh tokens: %w", err)
		}
	}
	return nil
}

func refreshTokenKey(tokenID string) string {
	return "refresh:jti:" + tokenID
}

func usedRefreshTokenKey(tokenID string) string {
	return "refresh:used:" + tokenID
}

func refreshUserSetKey(userID string) string {
	return "refresh:user:" + userID
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func equalTokenHash(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *Service) ValidateTokenPublic(token string) (string, error) {
	return ValidateToken(token, s.jwtSecret)
}

func (s *Service) RefreshTokenTTL() time.Duration {
	return s.refreshTokenTTL
}

const wsTicketTTL = 2 * time.Minute
const wsTicketRateLimitPerMinute = 30

func (s *Service) CheckWSTicketRateLimit(ctx context.Context, userID string) error {
	if s.redis == nil || userID == "" {
		return nil
	}
	key := fmt.Sprintf("ws:ticket:rate:%s:%d", userID, time.Now().Unix()/60)
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("ws ticket rate limit: %w", err)
	}
	if count == 1 {
		_ = s.redis.Expire(ctx, key, time.Minute+5*time.Second).Err()
	}
	if count > wsTicketRateLimitPerMinute {
		return fmt.Errorf("ws ticket rate limit exceeded")
	}
	return nil
}

func (s *Service) IssueWSTicket(ctx context.Context, userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random ticket: %w", err)
	}
	ticket := base64.RawURLEncoding.EncodeToString(b)
	key := "ws:ticket:" + ticket
	version, err := s.sessionVersion(ctx, userID)
	if err != nil {
		return "", err
	}
	record, _ := json.Marshal(struct {
		UserID  string `json:"user_id"`
		Version string `json:"version"`
	}{userID, version})
	if err := s.redis.Set(ctx, key, record, wsTicketTTL).Err(); err != nil {
		return "", fmt.Errorf("store ws ticket: %w", err)
	}
	return ticket, nil
}

func (s *Service) IsGloballyBanned(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM global_bans
			WHERE user_id = $1
			AND (expires_at IS NULL OR expires_at > NOW())
		)`,
		userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check global ban: %w", err)
	}
	return exists, nil
}

func (s *Service) GetGlobalBanReason(ctx context.Context, userID string) (string, error) {
	var reason string
	err := s.db.QueryRow(ctx,
		`SELECT reason FROM global_bans
		 WHERE user_id = $1
		 AND (expires_at IS NULL OR expires_at > NOW())`,
		userID,
	).Scan(&reason)
	if err != nil {
		return "", err
	}
	return reason, nil
}

func (s *Service) GlobalBan(ctx context.Context, userID, bannedBy, reason string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO global_bans (user_id, banned_by, reason)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET reason = $3, expires_at = NULL, created_at = NOW()`,
		userID, bannedBy, reason,
	)
	if err != nil {
		return fmt.Errorf("global ban: %w", err)
	}
	return s.revokeUserRefreshTokens(ctx, userID)
}

func (s *Service) GlobalUnban(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM global_bans WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("global unban: %w", err)
	}
	return nil
}

func (s *Service) ConsumeWSTicket(ctx context.Context, ticket string) (string, error) {
	if ticket == "" {
		return "", errors.New("missing ticket")
	}
	key := "ws:ticket:" + ticket
	// SECURITY-HARDENING: websocket tickets are one-time credentials consumed atomically.
	// VULNERABILITY FIXED: a captured ticket cannot be replayed across parallel socket upgrades.
	userID, err := s.redis.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", errors.New("invalid or expired ticket")
	}
	if err != nil {
		return "", fmt.Errorf("consume ws ticket: %w", err)
	}
	var record struct {
		UserID  string `json:"user_id"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(userID), &record); err != nil {
		record.UserID = userID
	}
	version, err := s.sessionVersion(ctx, record.UserID)
	if err != nil {
		return "", err
	}
	if version != record.Version || record.UserID == "" {
		return "", errors.New("revoked ticket")
	}
	return record.UserID, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(ctx,
		`SELECT id, username, public_key, created_at
		 FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Username, &user.PublicKey, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &user, nil
}

// SetupTOTP generates a TOTP secret for the user and stores it as pending
// (not yet enabled). Returns the secret and provisioning URI.
func (s *Service) SetupTOTP(ctx context.Context, userID string) (secret, uri string, err error) {
	var enabled bool
	err = s.db.QueryRow(ctx,
		`SELECT enabled FROM user_mfa_totp WHERE user_id = $1`,
		userID,
	).Scan(&enabled)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("query totp: %w", err)
	}
	if err == nil && enabled {
		return "", "", ErrTOTPAlreadyEnabled
	}

	secret, err = generateTOTPSecret()
	if err != nil {
		return "", "", fmt.Errorf("generate totp secret: %w", err)
	}
	uri = totpURI(userID, secret)

	encryptedSecret, err := s.encryptTOTPSecret(userID, secret)
	if err != nil {
		return "", "", err
	}
	result, err := s.db.Exec(ctx,
		`INSERT INTO user_mfa_totp (user_id, secret_enc, enabled)
		 VALUES ($1, $2, FALSE)
		 ON CONFLICT (user_id) DO UPDATE SET secret_enc = $2, enabled = FALSE, created_at = NOW() WHERE user_mfa_totp.enabled = FALSE`,
		userID, encryptedSecret,
	)
	if err != nil {
		return "", "", fmt.Errorf("store totp secret: %w", err)
	}
	if result.RowsAffected() != 1 {
		return "", "", ErrTOTPAlreadyEnabled
	}
	return secret, uri, nil
}

// VerifyTOTPSetup verifies a TOTP code and enables MFA for the user.
func (s *Service) VerifyTOTPSetup(ctx context.Context, userID, code string) error {
	var secretHash string
	var enabled bool
	err := s.db.QueryRow(ctx,
		`SELECT secret_enc, enabled FROM user_mfa_totp WHERE user_id = $1`,
		userID,
	).Scan(&secretHash, &enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTOTPNotEnabled
		}
		return fmt.Errorf("query totp secret: %w", err)
	}
	if enabled {
		return ErrTOTPAlreadyEnabled
	}

	plaintext, err := s.decryptTOTPSecret(userID, secretHash)
	if err != nil {
		return err
	}
	secretBytes, err := base32Decode(plaintext)
	if err != nil {
		return fmt.Errorf("decode totp secret: %w", err)
	}

	if err := verifyTOTPCode(secretBytes, code); err != nil {
		return ErrInvalidTOTPCode
	}

	result, err := s.db.Exec(ctx,
		`UPDATE user_mfa_totp SET enabled = TRUE WHERE user_id = $1 AND secret_enc = $2 AND enabled = FALSE`,
		userID, secretHash,
	)
	if err != nil {
		return fmt.Errorf("enable totp: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrInvalidTOTPCode
	}
	return nil
}

// VerifyTOTP verifies a TOTP code during login for MFA-enforced users.
func (s *Service) VerifyTOTP(ctx context.Context, userID, code string) error {
	var secretHash string
	var enabled bool
	err := s.db.QueryRow(ctx,
		`SELECT secret_enc, enabled FROM user_mfa_totp WHERE user_id = $1`,
		userID,
	).Scan(&secretHash, &enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // MFA not configured, skip
		}
		return fmt.Errorf("query totp secret: %w", err)
	}
	if !enabled {
		return nil // MFA not enabled, skip
	}

	plaintext, err := s.decryptTOTPSecret(userID, secretHash)
	if err != nil {
		return err
	}
	secretBytes, err := base32Decode(plaintext)
	if err != nil {
		return fmt.Errorf("decode totp secret: %w", err)
	}

	if code == "" {
		return ErrMFARequired
	}
	if err := verifyTOTPCode(secretBytes, code); err != nil {
		return err
	}
	// Reject reuse across login and MFA changes, including concurrent requests.
	accepted, err := s.redis.SetNX(ctx, "mfa:used:"+userID+":"+hashToken(plaintext+":"+code), "1", 2*time.Minute).Result()
	if err != nil {
		return fmt.Errorf("record totp use: %w", err)
	}
	if !accepted {
		return ErrInvalidTOTPCode
	}
	if !strings.HasPrefix(secretHash, totpStoragePrefix) {
		encrypted, err := s.encryptTOTPSecret(userID, plaintext)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(ctx, `UPDATE user_mfa_totp SET secret_enc=$1 WHERE user_id=$2 AND secret_enc=$3`, encrypted, userID, secretHash); err != nil {
			return err
		}
	}
	return nil
}

// DisableTOTP disables MFA for the user.
func (s *Service) DisableTOTP(ctx context.Context, userID, code string) error {
	if err := s.VerifyTOTP(ctx, userID, code); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx,
		`UPDATE user_mfa_totp SET enabled = FALSE WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("disable totp: %w", err)
	}
	return nil
}

// IsMFAEnabled returns true if the user has TOTP MFA enabled.
func (s *Service) IsMFAEnabled(ctx context.Context, userID string) (bool, error) {
	var enabled bool
	err := s.db.QueryRow(ctx,
		`SELECT enabled FROM user_mfa_totp WHERE user_id = $1`,
		userID,
	).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query totp status: %w", err)
	}
	return enabled, nil
}

// RequestPasswordReset generates a password reset token for the user.
// The token is NOT returned to the caller; it must be delivered out-of-band
// (e.g., email, SMS) to prevent token interception via HTTP response.
// Returns an empty string (not an error) when the user does not exist, to
// prevent user enumeration.
func (s *Service) RequestPasswordReset(ctx context.Context, username string) (string, error) {
	var userID string
	err := s.db.QueryRow(ctx,
		`SELECT id FROM users WHERE username = $1`,
		username,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return a non-error to prevent user enumeration
			return "", nil
		}
		return "", fmt.Errorf("lookup user: %w", err)
	}

	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}
	token := base32.StdEncoding.EncodeToString(tokenBytes)

	// Hash the token before storing
	tokenSum := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenSum[:])

	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = s.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}

	return token, nil
}

// ConfirmPasswordReset validates a reset token and updates the user's password.
func (s *Service) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	tokenSum := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenSum[:])

	var userID string
	var expiresAt time.Time
	err := s.db.QueryRow(ctx,
		`SELECT user_id, expires_at FROM password_reset_tokens
		 WHERE token_hash = $1 AND used_at IS NULL`,
		tokenHash,
	).Scan(&userID, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("invalid or expired reset token")
		}
		return fmt.Errorf("lookup reset token: %w", err)
	}

	if time.Now().After(expiresAt) {
		return errors.New("reset token has expired")
	}

	// Hash the new password
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update password and mark token as used in a single transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Consume exactly once inside the same transaction as the password change.
	err = tx.QueryRow(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW()
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		 RETURNING user_id`, tokenHash).Scan(&userID)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}
	if err := s.revokeUserRefreshTokens(ctx, userID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		passwordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
