package main

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"strings"
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

func wsAllowedOrigins(cfg *config.Config) []string {
	if len(cfg.WSAllowedOrigins) > 0 {
		return cfg.WSAllowedOrigins
	}
	return cfg.CORSAllowedOrigins
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

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
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

	authSvc := auth.NewService(database, rdb, cfg.JWTSecret)
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
	r.Use(metrics.HTTPMiddleware)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := cfg.CORSAllowedOrigins

			w.Header().Set("Vary", "Origin")

			// SECURITY: production must use an explicit allowlist to prevent hostile browser origins.
			if len(allowed) == 0 {
				if cfg.Environment == "production" {
					if origin != "" {
						slog.Warn("security cors origin rejected", "reason", "missing_origin_config", "origin", origin, "path", r.URL.Path)
						w.WriteHeader(http.StatusForbidden)
						return
					}
				} else {
					if origin != "" {
						w.Header().Set("Access-Control-Allow-Origin", origin)
					} else {
						w.Header().Set("Access-Control-Allow-Origin", "*")
					}
				}
			} else {
				originAllowed := false
				for _, o := range allowed {
					if o == origin {
						originAllowed = true
						break
					}
				}
				if originAllowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				} else if origin != "" {
					slog.Warn("security cors origin rejected", "reason", "origin_not_allowed", "origin", origin, "path", r.URL.Path)
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Use(secGuard.IPRateLimit)
	r.Use(requestBodyLimit(maxRequestBodyBytes))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","app":"zenthril"}`))
	})

	r.Get("/metrics", metricsAuth(cfg, metrics.Handler()))
	r.Get("/metrics/prometheus", metricsAuth(cfg, metrics.PrometheusHandler()))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.With(secGuard.BruteForceProtect).Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Post("/refresh", authHandler.Refresh)
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
		r.Post("/announce", federationHandler.Announce)
		r.Get("/peers", federationHandler.Peers)
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

	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		log.Fatal(server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}

func metricsAuth(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// SECURITY: metrics expose operational internals and require a dedicated bearer token outside development.
		if cfg.MetricsToken == "" && cfg.Environment == "development" {
			next(w, r)
			return
		}
		token := bearerToken(r)
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.MetricsToken)) != 1 {
			slog.Warn("security metrics access denied", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Valid metrics token required"}`))
			return
		}
		next(w, r)
	}
}

func bearerToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
}
