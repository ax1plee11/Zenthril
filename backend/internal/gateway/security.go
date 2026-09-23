package gateway

import (
	"context"
	"sync"
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
// SECURITY: prevents connection flooding and IP-based abuse.
type ConnectionGuard struct {
	maxConnectionsPerIP    int
	maxConnectionsPerUser  int
	antiFlood              *AntiFloodConfig
	mu                     sync.RWMutex
	reconnectTimestamps    map[string][]time.Time
	ipConnectionTimestamps map[string][]time.Time
}

// NewConnectionGuard creates a connection guard with anti-flooding.
// SECURITY: anti-flood config prevents reconnect flooding attacks.
// WEAKNESS FIXED: no reconnect throttling or IP-based flood protection existed.
func NewConnectionGuard(maxPerIP, maxPerUser int) *ConnectionGuard {
	if maxPerIP <= 0 {
		maxPerIP = 20
	}
	if maxPerUser <= 0 {
		maxPerUser = 5
	}
	return &ConnectionGuard{
		maxConnectionsPerIP:    maxPerIP,
		maxConnectionsPerUser:  maxPerUser,
		antiFlood:              DefaultAntiFloodConfig(),
		reconnectTimestamps:    make(map[string][]time.Time),
		ipConnectionTimestamps: make(map[string][]time.Time),
	}
}

func (g *ConnectionGuard) AllowIP(ip string, currentIPConnections int) bool {
	if g == nil {
		return true
	}
	if currentIPConnections >= g.maxConnectionsPerIP {
		return false
	}
	// SECURITY: check IP connection rate to prevent rapid connection cycling.
	// WEAKNESS FIXED: no IP-based connection rate limiting existed.
	return g.allowIPConnectionRate(ip)
}

func (g *ConnectionGuard) AllowUser(userID string, currentUserConnections int) bool {
	if g == nil {
		return true
	}
	return currentUserConnections < g.maxConnectionsPerUser
}

// CheckReconnectThrottle validates whether a reconnect attempt is allowed.
// SECURITY: prevents reconnect flooding attacks.
// WEAKNESS FIXED: no reconnect throttling existed.
func (g *ConnectionGuard) CheckReconnectThrottle(userID string) (bool, time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	timestamps := g.reconnectTimestamps[userID]

	// Remove timestamps older than 1 minute.
	cutoff := now.Add(-time.Minute)
	filtered := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= g.antiFlood.MaxReconnectAttempts {
		// Calculate backoff based on number of recent reconnects.
		backoff := g.antiFlood.ReconnectBackoff * time.Duration(len(filtered)-g.antiFlood.MaxReconnectAttempts+1)
		return false, backoff
	}

	filtered = append(filtered, now)
	g.reconnectTimestamps[userID] = filtered
	return true, 0
}

// RecordDisconnect records a disconnect event for reconnect tracking.
func (g *ConnectionGuard) RecordDisconnect(userID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.reconnectTimestamps[userID] = append(g.reconnectTimestamps[userID], time.Now())
}

// allowIPConnectionRate checks if the IP is connecting too rapidly.
// SECURITY: prevents IP-based connection flooding.
func (g *ConnectionGuard) allowIPConnectionRate(ip string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)
	timestamps := g.ipConnectionTimestamps[ip]

	filtered := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	// Allow max 10 new connections per minute per IP.
	if len(filtered) >= 10 {
		return false
	}

	filtered = append(filtered, now)
	g.ipConnectionTimestamps[ip] = filtered
	return true
}

// GetConnectionStats returns connection guard statistics for monitoring.
func (g *ConnectionGuard) GetConnectionStats() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return map[string]int{
		"tracked_users":    len(g.reconnectTimestamps),
		"tracked_ips":      len(g.ipConnectionTimestamps),
		"anti_flood_active": 1,
	}
}
