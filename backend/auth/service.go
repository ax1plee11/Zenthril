package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
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
	db              *pgxpool.Pool
	redis           *redis.Client
	jwtSecret       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
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
		db:              db,
		redis:           rdb,
		jwtSecret:       jwtSecret,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
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

func (s *Service) Login(ctx context.Context, username, password string) (*models.User, *TokenPair, error) {
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

	pair, err := s.issueTokenPair(ctx, user.ID.String())
	if err != nil {
		return nil, nil, err
	}

	return &user, pair, nil
}

func (s *Service) issueTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
	// SECURITY-HARDENING: access tokens are short-lived; refresh tokens are single-use and server-tracked.
	accessToken, err := GenerateAccessToken(userID, s.jwtSecret, s.accessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := GenerateRefreshToken(userID, s.jwtSecret, s.refreshTokenTTL)
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
	_ = s.redis.Del(ctx, refreshTokenKey(tokenID)).Err()
	_ = s.redis.SRem(ctx, refreshUserSetKey(userID), tokenID).Err()
	return nil
}

func (s *Service) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken != "" {
		if _, err := ValidateToken(accessToken, s.jwtSecret); err == nil {
			// SECURITY-HARDENING: blacklist valid access tokens until natural expiry on logout.
			key := "token:blacklist:" + hashToken(accessToken)
			_ = s.redis.Set(ctx, key, "1", s.accessTokenTTL)
		}
	}
	if refreshToken != "" {
		_ = s.RevokeRefreshToken(ctx, refreshToken)
	}
	return nil
}

func (s *Service) LogoutAll(ctx context.Context, userID, accessToken string) error {
	if accessToken != "" {
		if tokenUserID, err := ValidateToken(accessToken, s.jwtSecret); err == nil && tokenUserID == userID {
			// SECURITY-HARDENING: logout-all immediately rejects the current
			// access token and removes every tracked refresh token for the user.
			key := "token:blacklist:" + hashToken(accessToken)
			_ = s.redis.Set(ctx, key, "1", s.accessTokenTTL)
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
	return val > 0, nil
}

func (s *Service) wasRefreshTokenUsed(ctx context.Context, tokenID string) bool {
	exists, err := s.redis.Exists(ctx, usedRefreshTokenKey(tokenID)).Result()
	return err == nil && exists > 0
}

func (s *Service) revokeUserRefreshTokens(ctx context.Context, userID string) error {
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
	if err := s.redis.Set(ctx, key, userID, wsTicketTTL).Err(); err != nil {
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
	return nil
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
	return userID, nil
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
