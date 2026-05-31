package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"zenthril-backend/auth"
	"zenthril-backend/config"
	"zenthril-backend/db"
	"zenthril-backend/device"
	"zenthril-backend/federation"
	"zenthril-backend/friends"
	"zenthril-backend/guild"
	"zenthril-backend/hub"
	"zenthril-backend/message"
	"zenthril-backend/metrics"
	securityheaders "zenthril-backend/middleware"
	"zenthril-backend/security"
	"zenthril-backend/spam"
)

const maxRequestBodyBytes int64 = 1 << 20

var allowedCORSMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodOptions: {},
}

var allowedCORSHeaders = map[string]struct{}{
	"authorization": {},
	"content-type":  {},
}

func wsAllowedOrigins(cfg *config.Config) []string {
	return cfg.WSAllowedOrigins
}

func isAdmin(cfg *config.Config, userID string) bool {
	if userID == "" {
		return false
	}
	for _, id := range cfg.AdminUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func adminOnly(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.UserIDFromContext(r.Context())
			if !ok || !isAdmin(cfg, userID) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden","message":"Admin access required"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requestBodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// SECURITY: all mutating endpoints get a hard body cap before JSON decoding.
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func blockDebugEndpoints(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Environment == "production" && (r.URL.Path == "/debug" || strings.HasPrefix(r.URL.Path, "/debug/")) {
				// SECURITY-HARDENING: debug/pprof endpoints must never be exposed in production.
				// VULNERABILITY FIXED: accidental profiler/debug route registration cannot leak runtime internals.
				slog.Warn("security debug endpoint blocked", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"not_found","message":"Not found"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func corsMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(cfg.CORSAllowedOrigins))
	for _, origin := range cfg.CORSAllowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))

			w.Header().Add("Vary", "Origin")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")

			if origin != "" {
				if _, ok := allowed[origin]; !ok {
					// SECURITY-HARDENING: reject unknown browser origins instead of reflecting them.
					// VULNERABILITY FIXED: the API no longer mirrors arbitrary Origin values into CORS responses.
					slog.Warn("security cors origin rejected", "reason", "origin_not_allowed", "origin", origin, "path", r.URL.Path)
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				if origin == "" {
					slog.Warn("security cors preflight rejected", "reason", "empty_origin", "path", r.URL.Path)
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if !validPreflightRequest(r) {
					// SECURITY-HARDENING: preflight must request only methods and headers the API intentionally exposes.
					// VULNERABILITY FIXED: browsers cannot pre-authorize unsafe custom headers or methods.
					slog.Warn("security cors preflight rejected", "reason", "method_or_headers_not_allowed", "origin", origin, "path", r.URL.Path)
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func validPreflightRequest(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Header.Get("Access-Control-Request-Method")))
	if method == "" {
		return false
	}
	if _, ok := allowedCORSMethods[method]; !ok {
		return false
	}
	for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		header = strings.ToLower(strings.TrimSpace(header))
		if header == "" {
			continue
		}
		if _, ok := allowedCORSHeaders[header]; !ok {
			return false
		}
	}
	return true
}

func operationalTokenAuth(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// SECURITY-HARDENING: operational endpoints expose runtime state and require a bearer token in production.
		if cfg.Environment != "production" {
			next(w, r)
			return
		}
		if cfg.OperationalToken == "" || subtle.ConstantTimeCompare([]byte(bearerToken(r)), []byte(cfg.OperationalToken)) != 1 {
			slog.Warn("security operational endpoint access denied", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Valid operational token required"}`))
			return
		}
		next(w, r)
	}
}

func federationAuth(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// SECURITY-HARDENING: federation is alpha and fail-closed unless explicitly enabled.
		if !cfg.FederationEnabled {
			slog.Warn("security federation endpoint rejected", "reason", "disabled", "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = w.Write([]byte(`{"error":"federation_disabled","message":"Federation is not enabled on this node"}`))
			return
		}
		if cfg.FederationToken == "" || subtle.ConstantTimeCompare([]byte(bearerToken(r)), []byte(cfg.FederationToken)) != 1 {
			slog.Warn("security federation endpoint rejected", "reason", "invalid_token", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Valid federation token required"}`))
			return
		}
		next(w, r)
	}
}

func main() {
	_ = godotenv.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	for _, warning := range cfg.SecurityWarnings() {
		slog.Warn("security configuration warning", "warning", warning)
	}

	database, err := db.Open(cfg.DBURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer database.Close()

	sqlDB, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("sql db error: %v", err)
	}
	defer sqlDB.Close()

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url error: %v", err)
	}
	rdb := redis.NewClient(redisOpts)

	authSvc := auth.NewService(database, rdb, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	authHandler := auth.NewHandler(authSvc, cfg.Environment == "production")
	deviceSvc := device.NewService(database)
	deviceHandler := device.NewHandler(deviceSvc)

	guildSvc := guild.NewService(database, cfg.HTTPAddr)
	wsHub := hub.NewHub(guildSvc)
	go wsHub.Run()
	guildHandler := guild.NewHandler(guildSvc, wsHub)

	wsUpgrader := hub.NewUpgrader(wsAllowedOrigins(cfg), cfg.Environment)

	messageSvc := message.NewService(database, wsHub, guildSvc)
	messageHandler := message.NewHandler(messageSvc)

	spamGuard := spam.NewGuard(rdb)
	secGuard := security.NewGuard(rdb, sqlDB)

	friendsSvc := friends.NewService(database)
	friendsHandler := friends.NewHandler(friendsSvc, wsHub)

	federationSvc := federation.NewService(database)
	federationHandler := federation.NewHandler(federationSvc)

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(securityheaders.SecurityHeaders)
	r.Use(blockDebugEndpoints(cfg))
	r.Use(metrics.HTTPMiddleware)
	r.Use(corsMiddleware(cfg))

	r.Use(secGuard.IPRateLimit)
	r.Use(requestBodyLimit(maxRequestBodyBytes))

	r.Get("/health", operationalTokenAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	r.Get("/metrics", metricsAuth(cfg, metrics.Handler()))
	r.Get("/metrics/prometheus", metricsAuth(cfg, metrics.PrometheusHandler()))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.With(secGuard.BruteForceProtect).Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/mfa/totp/start", authHandler.TOTPStart)
			r.Post("/mfa/totp/confirm", authHandler.TOTPConfirm)
			r.Post("/mfa/totp/disable", authHandler.TOTPDisable)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.Middleware)
				r.Post("/ws-ticket", authHandler.WSTicket)
				r.Get("/me", authHandler.GetMe)
			})
		})
		r.Route("/devices", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Get("/", deviceHandler.ListOwn)
			r.Post("/register", deviceHandler.Register)
			r.Delete("/{deviceId}", deviceHandler.RevokeOwn)
		})
		r.Route("/key-bundles", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Post("/claim", deviceHandler.ClaimKeyBundle)
		})
		r.Route("/guilds", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Get("/", guildHandler.GetUserGuilds)
			r.Post("/", guildHandler.CreateGuild)
			r.Route("/{guildId}", func(r chi.Router) {
				r.Post("/invites", guildHandler.CreateInvite)
				r.Post("/roles", guildHandler.CreateRole)
				r.Get("/members", guildHandler.GetGuildMembers)
				r.Route("/members/{userId}", func(r chi.Router) {
					r.Delete("/", guildHandler.RemoveMember)
					r.Patch("/role", guildHandler.AssignRole)
					r.Post("/mute", guildHandler.MuteMember)
					r.Post("/ban", guildHandler.BanMember)
				})
				r.Post("/channels", guildHandler.CreateChannel)
				r.Get("/channels", guildHandler.GetGuildChannels)
			})
		})
		r.Route("/invites", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Post("/{code}/join", guildHandler.JoinByInvite)
		})
		r.Route("/channels", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Route("/{channelId}/messages", func(r chi.Router) {
				r.With(spamGuard.Middleware).Post("/", messageHandler.SendMessage)
				r.Get("/", messageHandler.GetHistory)
			})
		})
		r.Route("/messages", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Patch("/{messageId}", messageHandler.EditMessage)
			r.Delete("/{messageId}", messageHandler.DeleteMessage)
		})
		r.Route("/users", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Get("/{userId}/devices", deviceHandler.ListUser)
			r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query().Get("q")
				if len(q) < 2 {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("[]"))
					return
				}
				// Security: Validate query parameter to prevent SQL injection
				// Only allow alphanumeric, underscore, and hyphen
				for _, r := range q {
					if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
						w.Header().Set("Content-Type", "application/json")
						w.Write([]byte("[]"))
						return
					}
				}
				rows, err := database.Query(r.Context(),
					`SELECT id, username FROM users WHERE username ILIKE $1 LIMIT 20`,
					"%"+q+"%",
				)
				if err != nil {
					http.Error(w, `{"error":"search_failed"}`, 500)
					return
				}
				defer rows.Close()
				type result struct {
					ID       string `json:"id"`
					Username string `json:"username"`
				}
				var results []result
				for rows.Next() {
					var res result
					if err := rows.Scan(&res.ID, &res.Username); err == nil {
						results = append(results, res)
					}
				}
				if results == nil {
					results = []result{}
				}
				out, err := json.Marshal(results)
				if err != nil {
					http.Error(w, `{"error":"search_failed"}`, 500)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(out)
			})
		})
		r.Route("/friends", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Get("/", friendsHandler.List)
			r.Post("/request", friendsHandler.SendRequest)
			r.Post("/{userId}/accept", friendsHandler.Accept)
			r.Delete("/{userId}", friendsHandler.Decline)
		})
		r.Route("/admin", func(r chi.Router) {
			r.Use(authSvc.Middleware)
			r.Use(adminOnly(cfg))
			r.Post("/users/{userId}/ban", authHandler.GlobalBan)
			r.Delete("/users/{userId}/ban", authHandler.GlobalUnban)
		})
	})

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(wsHub, authSvc, wsUpgrader, w, r)
	})

	r.Route("/federation/v1", func(r chi.Router) {
		r.Post("/announce", federationAuth(cfg, federationHandler.Announce))
		r.Get("/peers", federationAuth(cfg, federationHandler.Peers))
		r.Post("/inbox", federationAuth(cfg, federationHandler.Inbox))
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("Zenthril node starting on %s", cfg.HTTPAddr)

	errCh := make(chan error, 1)
	go func() {
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			errCh <- server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
			return
		}
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown requested")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}

	// ARCHITECTURE: legacy entrypoint now drains HTTP requests on SIGTERM like cmd/api.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
}

func metricsAuth(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// SECURITY-HARDENING: metrics expose operational internals and require a dedicated bearer token outside development.
		// VULNERABILITY FIXED: Prometheus metrics are no longer anonymously readable in production.
		if cfg.MetricsToken == "" && cfg.Environment == "development" {
			next(w, r)
			return
		}
		token := bearerToken(r)
		if token != "" && cfg.MetricsToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(cfg.MetricsToken)) == 1 {
			next(w, r)
			return
		}
		if token != "" {
			if userID, err := auth.ValidateToken(token, cfg.JWTSecret); err == nil && isAdmin(cfg, userID) {
				// SECURITY-HARDENING: human access to metrics is limited to configured admins.
				// VULNERABILITY FIXED: regular authenticated users cannot read operational telemetry.
				next(w, r)
				return
			}
		}
		if token == "" {
			slog.Warn("security metrics access denied", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Valid metrics token required"}`))
			return
		}
		slog.Warn("security metrics access denied", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Valid metrics token required"}`))
	}
}

func bearerToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
}
