package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"veltrix-backend/auth"
	"veltrix-backend/config"
	"veltrix-backend/db"
	"veltrix-backend/friends"
	"veltrix-backend/guild"
	"veltrix-backend/hub"
	"veltrix-backend/message"
	"veltrix-backend/metrics"
	"veltrix-backend/moderation"
	"veltrix-backend/security"
	"veltrix-backend/spam"
)

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
	authHandler := auth.NewHandler(authSvc)

	guildSvc := guild.NewService(database, cfg.HTTPAddr)
	wsHub := hub.NewHub(guildSvc)
	go wsHub.Run()
	guildHandler := guild.NewHandler(guildSvc, wsHub)

	wsUpgrader := hub.NewUpgrader(wsAllowedOrigins(cfg))

	messageSvc := message.NewService(database, wsHub, guildSvc)
	messageHandler := message.NewHandler(messageSvc)

	spamGuard := spam.NewGuard(rdb)
	secGuard := security.NewGuard(rdb, sqlDB)

	friendsSvc := friends.NewService(database)
	friendsHandler := friends.NewHandler(friendsSvc, wsHub)

	connMonitor := moderation.NewConnectionMonitor(database)
	stateManager := moderation.NewGuildStateManager(database, cfg.AdminUserIDs)
	guildModerator := moderation.NewGuildModerator(database, connMonitor)
	moderationHandler := moderation.NewHandler(guildModerator, connMonitor, stateManager)

	guildSvc.SetStateChecker(stateManager)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(metrics.HTTPMiddleware)
	r.Use(connMonitor.ConnectionMiddleware)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed := cfg.CORSAllowedOrigins
			if len(allowed) == 0 {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				origin := r.Header.Get("Origin")
				for _, o := range allowed {
					if o == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Use(secGuard.IPRateLimit)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","app":"zenthril"}`))
	})

	r.Get("/metrics", metrics.Handler())
	r.Get("/metrics/prometheus", metrics.PrometheusHandler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.With(secGuard.BruteForceProtect).Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.Middleware)
				r.Post("/ws-ticket", authHandler.WSTicket)
				r.Get("/me", authHandler.GetMe)
			})
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
			r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query().Get("q")
				if len(q) < 2 {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("[]"))
					return
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
			r.Get("/users/{userId}/ips", moderationHandler.GetUserIPs)
			r.Post("/guilds/{guildId}/process-ban", moderationHandler.ProcessGuildBan)
			r.Post("/guilds/{guildId}/global-ban", moderationHandler.GlobalBanGuildAndMembers)
			r.Get("/guilds/{guildId}/member-ips", moderationHandler.GetGuildMembersIPs)
			r.Post("/guilds/{guildId}/lock", moderationHandler.LockGuild)
			r.Post("/guilds/{guildId}/unlock", moderationHandler.UnlockGuild)
			r.Get("/guilds/{guildId}/lock-status", moderationHandler.GetLockStatus)
			r.Get("/locked-guilds", moderationHandler.GetAllLockedGuilds)
			r.Get("/audit-log", authHandler.GetAuditLog)
		})
	})

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(wsHub, authSvc, wsUpgrader, w, r)
	})

	r.Route("/federation/v1", func(r chi.Router) {
		r.Post("/announce", notImplemented)
		r.Get("/peers", notImplemented)
	})

	log.Printf("Zenthril node starting on %s", cfg.HTTPAddr)

	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		log.Fatal(http.ListenAndServeTLS(cfg.HTTPAddr, cfg.TLSCertFile, cfg.TLSKeyFile, r))
	} else {
		log.Fatal(http.ListenAndServe(cfg.HTTPAddr, r))
	}
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"not_implemented"}`))
}
