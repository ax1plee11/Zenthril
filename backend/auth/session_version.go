package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
)

func sessionVersionKey(userID string) string { return "auth:session-version:" + userID }

// SetSessionRevocationHook closes local realtime connections when sessions change.
func (s *Service) SetSessionRevocationHook(hook func(string)) { s.onSessionsRevoked = hook }

func (s *Service) sessionVersion(ctx context.Context, userID string) (string, error) {
	version, err := s.redis.Get(ctx, sessionVersionKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return version, err
}

func (s *Service) validateSessionVersion(ctx context.Context, c *claims) error {
	version, err := s.sessionVersion(ctx, c.UserID)
	if err != nil {
		return fmt.Errorf("read session version: %w", err)
	}
	if version != c.SessionVersion {
		return ErrInvalidRefreshToken
	}
	return nil
}
