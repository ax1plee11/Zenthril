package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"veltrix-backend/models"
)

var ErrUsernameTaken = errors.New("username_taken")
var ErrInvalidCredentials = errors.New("invalid_credentials")

type Service struct {
	db        *pgxpool.Pool
	redis     *redis.Client
	jwtSecret string
}

func NewService(db *pgxpool.Pool, rdb *redis.Client, jwtSecret string) *Service {
	return &Service{db: db, redis: rdb, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, username, password, publicKey string) (*models.User, string, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
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
			return nil, "", ErrUsernameTaken
		}
		return nil, "", fmt.Errorf("insert user: %w", err)
	}

	token, err := GenerateToken(user.ID.String(), s.jwtSecret)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	return &user, token, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*models.User, string, error) {
	var user models.User
	var passwordHash string

	err := s.db.QueryRow(ctx,
		`SELECT id, username, password_hash, public_key, created_at
		 FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &passwordHash, &user.PublicKey, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", fmt.Errorf("query user: %w", err)
	}

	ok, err := VerifyPassword(password, passwordHash)
	if err != nil {
		return nil, "", fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, "", ErrInvalidCredentials
	}

	token, err := GenerateToken(user.ID.String(), s.jwtSecret)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	return &user, token, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := ValidateToken(token, s.jwtSecret)
	if err != nil {
		return nil
	}
	key := "token:blacklist:" + token
	if err := s.redis.Set(ctx, key, "1", tokenTTL).Err(); err != nil {
		return fmt.Errorf("blacklist token: %w", err)
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

// ValidateTokenPublic проверяет JWT-токен и возвращает userID (публичный метод для использования вне пакета).
func (s *Service) ValidateTokenPublic(token string) (string, error) {
	return ValidateToken(token, s.jwtSecret)
}
