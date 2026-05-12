package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
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

type Service struct {
	db        *pgxpool.Pool
	redis     *redis.Client
	jwtSecret string
}

func NewService(db *pgxpool.Pool, rdb *redis.Client, jwtSecret string) *Service {
	return &Service{db: db, redis: rdb, jwtSecret: jwtSecret}
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
	accessToken, err := GenerateToken(userID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := generateTypedToken(userID, "refresh", refreshTokenTTL, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	key := "refresh:" + refreshToken
	if err := s.redis.Set(ctx, key, userID, refreshTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	key := "refresh:" + refreshToken
	storedUserID, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, fmt.Errorf("check refresh token: %w", err)
	}
	if storedUserID != userID {
		return nil, ErrInvalidRefreshToken
	}

	if err := s.redis.Del(ctx, key).Err(); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	return s.issueTokenPair(ctx, userID)
}

func (s *Service) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	key := "refresh:" + refreshToken
	s.redis.Del(ctx, key)
	return nil
}

func (s *Service) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken != "" {
		if _, err := ValidateToken(accessToken, s.jwtSecret); err == nil {
			key := "token:blacklist:" + accessToken
			_ = s.redis.Set(ctx, key, "1", accessTokenTTL)
		}
	}
	if refreshToken != "" {
		_ = s.RevokeRefreshToken(ctx, refreshToken)
	}
	return nil
}

func (s *Service) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := "token:blacklist:" + token
	val, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check blacklist: %w", err)
	}
	return val > 0, nil
}

func (s *Service) ValidateTokenPublic(token string) (string, error) {
	return ValidateToken(token, s.jwtSecret)
}

const wsTicketTTL = 2 * time.Minute

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
