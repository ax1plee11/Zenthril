package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements distributed fixed-window rate limiting for gateway commands.
// WEAKNESS FIXED: per-user limits are now coordinated across gateway nodes via Redis.
type RedisRateLimiter struct {
	redis  *redis.Client
	prefix string
}

func NewRedisRateLimiter(rdb *redis.Client, prefix string) *RedisRateLimiter {
	if prefix == "" {
		prefix = "zenthril"
	}
	return &RedisRateLimiter{redis: rdb, prefix: prefix}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, userID string, limit int, window time.Duration) (bool, error) {
	if l == nil || l.redis == nil {
		return false, fmt.Errorf("redis rate limiter is not configured")
	}
	if userID == "" {
		return false, fmt.Errorf("user id is required")
	}
	if limit <= 0 || window <= 0 {
		return false, fmt.Errorf("invalid rate limiter parameters")
	}

	now := time.Now().Unix()
	windowSeconds := int64(window.Seconds())
	bucket := now / windowSeconds
	key := fmt.Sprintf("%s:ratelimit:gateway:user:%s:%d", l.prefix, userID, bucket)

	pipe := l.redis.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window+5*time.Second) //nolint:errcheck
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	// SECURITY: one authenticated account cannot bypass the per-user WebSocket budget
	// by spreading traffic across multiple gateway nodes.
	return incr.Val() <= int64(limit), nil
}
