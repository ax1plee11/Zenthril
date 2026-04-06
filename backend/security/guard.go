package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Guard реализует защиту от DDoS и брутфорс-атак.
type Guard struct {
	redis *redis.Client
	db    *sql.DB
}

// NewGuard создаёт новый Guard.
func NewGuard(rdb *redis.Client, db *sql.DB) *Guard {
	return &Guard{redis: rdb, db: db}
}

// IPRateLimit — DDoS защита: >1000 req/сек с одного IP → блокировка на 60 сек.
func (g *Guard) IPRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		blockKey := "security:ip_block:" + ip
		counterKey := "security:ip_rps:" + ip

		// Проверяем активную блокировку
		blocked, err := g.redis.Exists(r.Context(), blockKey).Result()
		if err == nil && blocked > 0 {
			http.Error(w, `{"error":"too_many_requests","message":"IP temporarily blocked"}`, http.StatusTooManyRequests)
			return
		}

		// Инкрементируем счётчик запросов (окно 1 сек)
		pipe := g.redis.Pipeline()
		incrCmd := pipe.Incr(r.Context(), counterKey)
		pipe.Expire(r.Context(), counterKey, time.Second)
		_, _ = pipe.Exec(r.Context())

		count := incrCmd.Val()
		if count > 1000 {
			// Блокируем IP на 60 сек
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

// BruteForceProtect — защита от брутфорса на POST /auth/login.
// >10 неудачных попыток за 1 мин с одного IP → блокировка на 15 мин.
func (g *Guard) BruteForceProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		blockKey := "security:bf_block:" + ip

		// Проверяем активную блокировку
		blocked, err := g.redis.Exists(r.Context(), blockKey).Result()
		if err == nil && blocked > 0 {
			_ = g.LogSecurityEvent(r.Context(), "brute_force_blocked", ip, "", map[string]interface{}{
				"path": r.URL.Path,
			})
			http.Error(w, `{"error":"too_many_requests","message":"Too many failed login attempts"}`, http.StatusTooManyRequests)
			return
		}

		// Оборачиваем ResponseWriter для перехвата статуса ответа
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		// Если логин неудачный (401) — увеличиваем счётчик
		if rw.statusCode == http.StatusUnauthorized {
			failKey := "security:bf_fails:" + ip
			count, _ := g.redis.Incr(r.Context(), failKey).Result()
			g.redis.Expire(r.Context(), failKey, time.Minute) //nolint:errcheck

			_ = g.LogSecurityEvent(r.Context(), "auth_fail", ip, "", map[string]interface{}{
				"attempt": count,
			})

			if count > 10 {
				g.redis.Set(r.Context(), blockKey, "1", 15*time.Minute) //nolint:errcheck
				g.redis.Del(r.Context(), failKey)                        //nolint:errcheck
				_ = g.LogSecurityEvent(r.Context(), "brute_force_detected", ip, "", map[string]interface{}{
					"blocked_for": "15m",
				})
			}
		}
	})
}

// LogSecurityEvent записывает событие безопасности в таблицу security_log.
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

	_, err = g.db.ExecContext(ctx,
		`INSERT INTO security_log (event_type, ip_address, user_id, details)
		 VALUES ($1, $2, $3, $4)`,
		eventType, ipVal, userIDVal, string(detailsJSON),
	)
	if err != nil {
		return fmt.Errorf("log security event: %w", err)
	}
	return nil
}

// extractIP извлекает IP-адрес из запроса (учитывает X-Forwarded-For).
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// responseWriter оборачивает http.ResponseWriter для перехвата статус-кода.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
