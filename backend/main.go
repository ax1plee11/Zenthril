package main

import (
	"context"
	"crypto/subtle"
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
	"zenthril-backend/user"
)

const maxRequestBodyBytes int64 = 1 << 20
const readinessTimeout = 2 * time.Second

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

func blockDebugEndpoints(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Environment == "production" && strings.Contains(r.URL.Path, "/debug/") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requestBodyLimit(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// validPreflightRequest ensures the preflight only asks for methods and headers
// the API intentionally exposes.
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

type readinessDependency struct {
	Name string
	Ping func(context.Context) error
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readinessHandler(environment string, deps ...readinessDependency) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		checks := make(map[string]string, len(deps))
		ready := true
		for _, dep := range deps {
			if dep.Name == "" || dep.Ping == nil {
				continue
			}
			if err := dep.Ping(ctx); err != nil {
				ready = false
				checks[dep.Name] = "down"
				slog.Warn("readiness dependency failed", "dependency", dep.Name, "error", err)
				continue
			}
			checks[dep.Name] = "ok"
		}

		if !ready {
			metrics.Global().IncrementReadinessFailures()
			// SECURITY-HARDENING: production readiness avoids leaking internal dependency details to unauthenticated probes.
			if environment == "production" {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
				return
			}
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"checks": checks,
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ready",
			"checks": checks,
		})
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

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url error: %v", err)
	}
	rdb := redis.NewClient(redisOpts)

	authSvc := auth.NewService(database, rdb, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	authHandler := auth.NewHandler(authSvc, cfg.SecureCookies)
	deviceSvc := device.NewService(database)
	deviceHandler := device.NewHandler(deviceSvc)

	guildSvc := guild.NewService(database, cfg.HTTPAddr)
	guildSvc.SetSuperAdmins(cfg.AdminUserIDs)
	wsHub := hub.NewHubWithUserMessageLimiter(guildSvc, hub.NewRedisFixedWindowLimiter(rdb, "zenthril"))
	go wsHub.Run()
	guildHandler := guild.NewHandler(guildSvc, wsHub)

	wsUpgrader := hub.NewUpgrader(wsAllowedOrigins(cfg), cfg.Environment)

	messageSvc := message.NewService(database, wsHub, guildSvc)
	messageHandler := message.NewHandler(messageSvc)

	spamGuard := spam.NewGuard(rdb)
	// VULNERABILITY FIXED: Guard now uses the same pgxpool.Pool as the rest of
	// the application. The second database/sql + lib/pq pool has been removed.
	secGuard := security.NewGuard(rdb, database)

	friendsSvc := friends.NewService(database)
	friendsHandler := friends.NewHandler(friendsSvc, wsHub)

	userSvc := user.NewService(database)

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
	r.Use(securityheaders.CORS(*cfg))

	r.Use(secGuard.IPRateLimit)
	r.Use(requestBodyLimit(maxRequestBodyBytes))

	ready := readinessHandler(cfg.Environment,
		readinessDependency{
			Name: "postgres",
			Ping: func(ctx context.Context) error {
				return database.Ping(ctx)
			},
		},
		readinessDependency{
			Name: "redis",
			Ping: func(ctx context.Context) error {
				return rdb.Ping(ctx).Err()
			},
		},
	)
	r.Get("/health", operationalTokenAuth(cfg, livenessHandler))
	r.Get("/healthz", operationalTokenAuth(cfg, livenessHandler))
	r.Get("/ready", operationalTokenAuth(cfg, ready))
	r.Get("/readyz", operationalTokenAuth(cfg, ready))
	r.Get("/livez", livenessHandler)

	r.Get("/metrics", metricsAuth(cfg, metrics.Handler()))
	r.Get("/metrics/prometheus", metricsAuth(cfg, metrics.PrometheusHandler()))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.With(secGuard.BruteForceProtect).Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Post("/refresh", authHandler.Refresh)
			// TOTP/MFA routes are registered but return 501 Not Implemented.
			// They are explicitly hidden in production to avoid false signals of
			// MFA support. Set TOTP_ENABLED=true to expose them in development/staging.
			if cfg.Environment != "production" {
				r.Post("/mfa/totp/start", authHandler.TOTPStart)
				r.Post("/mfa/totp/confirm", authHandler.TOTPConfirm)
				r.Post("/mfa/totp/disable", authHandler.TOTPDisable)
			}
			r.Group(func(r chi.Router) {
				r.Use(authSvc.Middleware)
				r.Post("/ws-ticket", authHandler.WSTicket)
				r.Post("/logout-all", authHandler.LogoutAll)
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
				r.Get("/roles", guildHandler.ListRoles)
				r.Post("/roles", guildHandler.CreateRole)
				r.Patch("/roles/{roleId}", guildHandler.UpdateRole)
				r.Delete("/roles/{roleId}", guildHandler.DeleteRole)
				r.Get("/members", guildHandler.GetGuildMembers)
				r.Route("/members/{userId}", func(r chi.Router) {
					r.Delete("/", guildHandler.RemoveMember)
					r.Patch("/role", guildHandler.AssignRole)
					r.Put("/roles/{roleId}", guildHandler.AddMemberRole)
					r.Delete("/roles/{roleId}", guildHandler.RemoveMemberRole)
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
			r.Get("/{channelId}/e2ee-recipients", messageHandler.ListRecipientDevices)
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
				results, err := userSvc.Search(r.Context(), q)
				if err != nil {
					// Return empty array for validation errors (too short, invalid chars).
					// Log actual DB errors as warnings.
					if err == user.ErrQueryTooShort || err == user.ErrQueryTooLong || err == user.ErrQueryInvalid {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte("[]"))
						return
					}
					slog.Warn("user search error", "error", err)
					http.Error(w, `{"error":"search_failed"}`, http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, results)
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

	// Static SPA — serve pre-built frontend if STATIC_DIR is set.
	// This allows a single ngrok / reverse-proxy URL to serve both the API and
	// the React app without a separate web server.
	if cfg.StaticDir != "" {
		fs := http.FileServer(http.Dir(cfg.StaticDir))
		// All unmatched paths fall through to index.html for client-side routing.
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := cfg.StaticDir + r.URL.Path
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, cfg.StaticDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, cfg.StaticDir+"/index.html")
		})
		slog.Info("serving static frontend", "dir", cfg.StaticDir)
	}

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

	// ARCHITECTURE: legacy entrypoint now drains HTTP requests and WebSocket
	// connections on SIGTERM like cmd/api.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	wsHub.Drain()
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

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, `{"error":"encode_failed"}`, http.StatusInternalServerError)
	}
}
