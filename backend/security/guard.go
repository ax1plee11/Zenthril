package security

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Guard provides IP rate limiting, brute-force protection, and security event logging.
//
// Previously Guard held a *database/sql.DB alongside the pgxpool.Pool used
// everywhere else in the service, creating two independent connection pools to
// the same Postgres instance.
//
// VULNERABILITY FIXED: Guard now uses the same *pgxpool.Pool that the rest of
// the application uses. The database/sql + lib/pq dependency has been removed.
type Guard struct {
	redis *redis.Client
	db    *pgxpool.Pool
	// trustedProxyHops is the number of reverse proxies in front of the service
	// that append the real client address to X-Forwarded-For.
	//
	// SECURITY: forwarded headers are controlled by the client unless the
	// deployment guarantees a trusted proxy rewrites them. Trusting them by
	// default would let any caller mint an arbitrary identity for per-IP rate
	// limiting and brute-force protection simply by rotating a request header.
	// The default is 0, meaning only RemoteAddr is trusted.
	trustedProxyHops int
}

func NewGuard(rdb *redis.Client, db *pgxpool.Pool) *Guard {
	return NewGuardWithTrustedProxies(rdb, db, 0)
}

// NewGuardWithTrustedProxies creates a Guard that honours X-Forwarded-For when
// the deployment sits behind trustedProxyHops reverse proxies.
//
// SECURITY: a negative value is clamped to 0 so a misconfiguration can never
// enable header trust.
func NewGuardWithTrustedProxies(rdb *redis.Client, db *pgxpool.Pool, trustedProxyHops int) *Guard {
	if trustedProxyHops < 0 {
		trustedProxyHops = 0
	}
	return &Guard{redis: rdb, db: db, trustedProxyHops: trustedProxyHops}
}

// TrustedProxyHops reports the configured number of trusted proxies.
func (g *Guard) TrustedProxyHops() int { return g.trustedProxyHops }

func (g *Guard) IPRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := g.extractIP(r)
		blockKey := "security:ip_block:" + ip
		counterKey := "security:ip_rps:" + ip

		blocked, err := g.redis.Exists(r.Context(), blockKey).Result()
		if err == nil && blocked > 0 {
			http.Error(w, `{"error":"too_many_requests","message":"IP temporarily blocked"}`, http.StatusTooManyRequests)
			return
		}

		pipe := g.redis.Pipeline()
		incrCmd := pipe.Incr(r.Context(), counterKey)
		pipe.Expire(r.Context(), counterKey, time.Second)
		_, _ = pipe.Exec(r.Context())

		count := incrCmd.Val()
		if count > 1000 {
			g.redis.Set(r.Context(), blockKey, "1", 60*time.Second) //nolint:errcheck
			_ = g.LogSecurityEvent(r.Context(), "ip_blocked", ip, "", map[string]interface{}{
				"requests_per_second": count,
			})
			http.Error(w, `{"error":"too_many_requests","message":"IP temporarily blocked"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (g *Guard) BruteForceProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := g.extractIP(r)
		blockKey := "security:bf_block:" + ip

		blocked, err := g.redis.Exists(r.Context(), blockKey).Result()
		if err == nil && blocked > 0 {
			_ = g.LogSecurityEvent(r.Context(), "brute_force_blocked", ip, "", map[string]interface{}{
				"path": r.URL.Path,
			})
			http.Error(w, `{"error":"too_many_requests","message":"Too many failed login attempts"}`, http.StatusTooManyRequests)
			return
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		if rw.statusCode == http.StatusUnauthorized {
			failKey := "security:bf_fails:" + ip
			count, _ := g.redis.Incr(r.Context(), failKey).Result()
			g.redis.Expire(r.Context(), failKey, time.Minute) //nolint:errcheck

			_ = g.LogSecurityEvent(r.Context(), "auth_fail", ip, "", map[string]interface{}{
				"attempt": count,
			})

			if count > 10 {
				g.redis.Set(r.Context(), blockKey, "1", 15*time.Minute) //nolint:errcheck
				g.redis.Del(r.Context(), failKey)                       //nolint:errcheck
				_ = g.LogSecurityEvent(r.Context(), "brute_force_detected", ip, "", map[string]interface{}{
					"blocked_for": "15m",
				})
			}
		}
	})
}

// rateLimitAllows reports whether a request within the given budget may proceed.
// SECURITY: the boundary is inclusive, so `limit` is the exact number of
// permitted requests per window before rejection starts.
func rateLimitAllows(count, limit int64) bool {
	return count <= limit
}

// authRateLimitKey namespaces the counter per scope and client IP so limits for
// different endpoints cannot be exhausted by each other.
func authRateLimitKey(scope, ip string) string {
	return "security:rl:" + scope + ":" + ip
}

// AuthRateLimit applies a fixed-window request budget per client IP.
//
// SECURITY: BruteForceProtect only counts 401 responses, so endpoints that
// succeed on the happy path (registration, refresh rotation, logout) had no
// effective limit and could be flooded. Each scope gets an independent budget.
func (g *Guard) AuthRateLimit(scope string, limit int, window time.Duration) func(http.Handler) http.Handler {
	if limit <= 0 || window <= 0 {
		// SECURITY: fail at construction time rather than silently creating a
		// limiter that allows unlimited traffic.
		panic("security: AuthRateLimit requires a positive limit and window")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := g.extractIP(r)
			key := authRateLimitKey(scope, ip)

			count, err := g.redis.Incr(r.Context(), key).Result()
			if err != nil {
				// SECURITY: fail closed. Authentication already depends on Redis
				// for refresh rotation and WebSocket tickets, so this adds no new
				// availability risk, whereas failing open would remove the control.
				slog.Warn("auth rate limit unavailable", "scope", scope, "error", err)
				http.Error(w, `{"error":"service_unavailable","message":"Rate limiter unavailable"}`, http.StatusServiceUnavailable)
				return
			}
			if count == 1 {
				g.redis.Expire(r.Context(), key, window) //nolint:errcheck
			}
			if !rateLimitAllows(count, int64(limit)) {
				_ = g.LogSecurityEvent(r.Context(), "rate_limited", ip, "", map[string]interface{}{
					"scope":  scope,
					"limit":  limit,
					"window": window.String(),
					"count":  count,
				})
				http.Error(w, `{"error":"too_many_requests","message":"Too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (g *Guard) LogSecurityEvent(ctx context.Context, eventType, ip, userID string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	var userIDVal interface{}
	if userID != "" {
		userIDVal = userID
	}

	var ipVal interface{}
	if ip != "" {
		ipVal = ip
	}

	_, err = g.db.Exec(ctx,
		`INSERT INTO security_log (event_type, ip_address, user_id, details)
		 VALUES ($1, $2, $3, $4)`,
		eventType, ipVal, userIDVal, string(detailsJSON),
	)
	if err != nil {
		return fmt.Errorf("log security event: %w", err)
	}
	return nil
}

// resolveForwardedClientIP returns the client address observed by the first
// trusted proxy, or ok=false when the header must not be trusted.
//
// SECURITY: with N trusted proxies the client controls the leading entries of
// X-Forwarded-For and each proxy appends exactly one entry, so the address the
// first proxy observed sits at index len(parts)-N. That index is valid when the
// chain is at least N entries long: a longer chain means the client supplied a
// prefix that is correctly skipped, and an exactly-N-long chain means the client
// supplied nothing at all. A shorter chain means a proxy did not append, so no
// entry can be attributed and the header is rejected rather than guessed.
//
// SECURITY: this makes the result depend on the declared hop count being
// accurate. If TRUSTED_PROXY_HOPS overstates the real proxy depth, a client can
// position its own value in the trusted region. The setting is therefore an
// explicit operator declaration, not something inferred at runtime.
func resolveForwardedClientIP(xff string, trustedProxyHops int) (string, bool) {
	parts := strings.Split(xff, ",")
	index := len(parts) - trustedProxyHops
	if index < 0 || index >= len(parts) {
		return "", false
	}
	ip := strings.TrimSpace(parts[index])
	if ip == "" || net.ParseIP(ip) == nil {
		return "", false
	}
	return ip, true
}

// clientIPFromRemoteAddr extracts the host portion of a host:port remote address.
func clientIPFromRemoteAddr(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return strings.TrimSpace(remoteAddr)
	}
	return host
}

// extractIP resolves the address used as a rate-limit and brute-force key.
//
// SECURITY: X-Forwarded-For and X-Real-IP are attacker controlled whenever the
// service is reachable without a trusted proxy rewriting them. With the default
// configuration of zero trusted proxies only RemoteAddr is used, so a client
// cannot bypass a per-IP limit by rotating a header. This value gates access
// control: it keys IPRateLimit, BruteForceProtect and AuthRateLimit.
func (g *Guard) extractIP(r *http.Request) string {
	if g.trustedProxyHops > 0 {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if ip, ok := resolveForwardedClientIP(xff, g.trustedProxyHops); ok {
				return ip
			}
		}
		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" && net.ParseIP(xri) != nil {
			return xri
		}
	}
	return clientIPFromRemoteAddr(r.RemoteAddr)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
