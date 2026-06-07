package hub

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type UserMessageLimiter interface {
	Allow(ctx context.Context, userID string, limit int, window time.Duration) (bool, error)
}

type RedisFixedWindowLimiter struct {
	redis  *redis.Client
	prefix string
}

func NewRedisFixedWindowLimiter(rdb *redis.Client, prefix string) *RedisFixedWindowLimiter {
	if prefix == "" {
		prefix = "zenthril"
	}
	return &RedisFixedWindowLimiter{redis: rdb, prefix: prefix}
}

func (l *RedisFixedWindowLimiter) Allow(ctx context.Context, userID string, limit int, window time.Duration) (bool, error) {
	if l == nil || l.redis == nil {
		return false, fmt.Errorf("redis limiter is not configured")
	}
	if userID == "" {
		return false, fmt.Errorf("user id is required")
	}
	if limit <= 0 || window <= 0 {
		return false, fmt.Errorf("invalid limiter parameters")
	}

	now := time.Now().Unix()
	windowSeconds := int64(window.Seconds())
	bucket := now / windowSeconds
	key := fmt.Sprintf("%s:ratelimit:ws:user:%s:%d", l.prefix, userID, bucket)

	pipe := l.redis.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window+5*time.Second) //nolint:errcheck
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	// SECURITY-HARDENING: one authenticated account cannot bypass the per-user
	// WebSocket message budget by spreading traffic across multiple API nodes.
	return incr.Val() <= int64(limit), nil
}
