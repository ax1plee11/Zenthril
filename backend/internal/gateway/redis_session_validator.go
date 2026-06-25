package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisSessionValidator implements SessionValidator using Redis for token
// blacklist checks and Postgres for global ban lookups.
//
// SECURITY: this is the production implementation of SessionValidator.
// NoopSessionValidator must never be used when ENV=production.
type RedisSessionValidator struct {
	redis     *redis.Client
	banLookup GlobalBanLookup
}

// GlobalBanLookup abstracts the global-ban database query so the gateway
// package stays decoupled from the auth/pgx packages.
type GlobalBanLookup interface {
	IsGloballyBanned(ctx context.Context, userID string) (bool, error)
}

// NewRedisSessionValidator creates a production SessionValidator backed by Redis
// for the token blacklist and an optional ban lookup for global ban enforcement.
//
// If banLookup is nil the validator only checks the Redis blacklist; global ban
// enforcement is disabled (appropriate for edge/gateway nodes that do not have
// direct DB access). In the primary API node banLookup must always be set.
func NewRedisSessionValidator(rdb *redis.Client, banLookup GlobalBanLookup) *RedisSessionValidator {
	return &RedisSessionValidator{redis: rdb, banLookup: banLookup}
}

func (v *RedisSessionValidator) HasGlobalBanLookup() bool {
	return v != nil && v.banLookup != nil
}

// IsTokenBlacklisted checks the Redis revocation set populated by auth.Service.Logout
// and auth.Service.LogoutAll. Key format mirrors auth/service.go: "token:blacklist:<sha256>".
func (v *RedisSessionValidator) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	key := "token:blacklist:" + hashToken(token)
	val, err := v.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("gateway blacklist check: %w", err)
	}
	return val > 0, nil
}

// IsGloballyBanned delegates to the configured GlobalBanLookup.
// If no lookup is configured it returns (false, nil) — permissive but safe
// since the ban is also enforced by the legacy HTTP middleware layer.
func (v *RedisSessionValidator) IsGloballyBanned(ctx context.Context, userID string) (bool, error) {
	if v.banLookup == nil {
		return false, nil
	}
	return v.banLookup.IsGloballyBanned(ctx, userID)
}

// hashToken mirrors auth/service.go hashToken to reuse the same Redis keys.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
