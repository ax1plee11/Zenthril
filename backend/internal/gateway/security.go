package gateway

import (
	"context"
	"time"
)

// ChannelAccessChecker authorizes channel subscriptions before registry mutation.
// ARCHITECTURE: mirrors hub.ChannelAccessChecker to keep ACL semantics consistent.
type ChannelAccessChecker interface {
	UserHasChannelAccess(ctx context.Context, userID, channelID string) (bool, error)
}

// SessionValidator enforces session lifecycle checks beyond JWT signature validation.
type SessionValidator interface {
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	IsGloballyBanned(ctx context.Context, userID string) (bool, error)
}

// DistributedRateLimiter coordinates per-user command budgets across gateway nodes.
type DistributedRateLimiter interface {
	Allow(ctx context.Context, userID string, limit int, window time.Duration) (bool, error)
}

// ConnectionGuard limits WebSocket upgrade abuse by IP and per-user connection count.
type ConnectionGuard struct {
	maxConnectionsPerIP   int
	maxConnectionsPerUser int
}

func NewConnectionGuard(maxPerIP, maxPerUser int) *ConnectionGuard {
	if maxPerIP <= 0 {
		maxPerIP = 20
	}
	if maxPerUser <= 0 {
		maxPerUser = 5
	}
	return &ConnectionGuard{
		maxConnectionsPerIP:   maxPerIP,
		maxConnectionsPerUser: maxPerUser,
	}
}

func (g *ConnectionGuard) AllowIP(ip string, currentIPConnections int) bool {
	if g == nil {
		return true
	}
	return currentIPConnections < g.maxConnectionsPerIP
}

func (g *ConnectionGuard) AllowUser(userID string, currentUserConnections int) bool {
	if g == nil {
		return true
	}
	return currentUserConnections < g.maxConnectionsPerUser
}
