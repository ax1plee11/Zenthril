package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"veltrix-backend/auth"
	"veltrix-backend/config"
	"veltrix-backend/db"
)

func main() {
	// Загружаем .env если есть (игнорируем ошибку — в проде переменные задаются иначе)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// Инициализируем PostgreSQL
	database, err := db.Open(cfg.DBURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer database.Close()

	// Инициализируем Redis
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url error: %v", err)
	}
	rdb := redis.NewClient(redisOpts)

	// Инициализируем AuthService и Handler
	authSvc := auth.NewService(database, rdb, cfg.JWTSecret)
	authHandler := auth.NewHandler(authSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","app":"veltrix"}`))
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
		})
		r.Route("/guilds", func(r chi.Router) {
			r.Get("/", notImplemented)
			r.Post("/", notImplemented)
		})
	})

	// Federation endpoints
	r.Route("/federation/v1", func(r chi.Router) {
		r.Post("/announce", notImplemented)
		r.Get("/peers", notImplemented)
	})

	log.Printf("Veltrix node starting on %s", cfg.HTTPAddr)

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
