package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"zenthril-backend/internal/app"
	"zenthril-backend/middleware"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	container, err := app.New(ctx)
	if err != nil {
		log.Fatalf("boot api: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := container.Close(shutdownCtx); err != nil {
			container.Logger.Error("close container", "error", err)
		}
	}()

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	// SECURITY: defense-in-depth headers for operational and WebSocket upgrade endpoints.
	router.Use(middleware.SecurityHeaders)

	router.Get("/livez", func(w http.ResponseWriter, r *http.Request) {
		// SECURITY: public liveness is intentionally minimal and does not expose
		// dependency state, node metadata, versions, or operational internals.
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/healthz", operationalTokenAuth(container, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	router.Get("/readyz", operationalTokenAuth(container, func(w http.ResponseWriter, r *http.Request) {
		// RESILIENCE: this next-generation API entrypoint does not own live DB/Redis
		// clients yet, so readiness reports only the dependencies wired into this container.
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ready",
			"checks": map[string]any{
				"gateway":   "ok",
				"event_bus": "configured",
				"shards":    len(container.ShardManager.Shards()),
			},
		})
	}))
	router.Get("/api/v2/gateway/stats", operationalTokenAuth(container, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, container.GatewayRegistry.Stats())
	}))
	router.Handle("/ws", container.GatewayHandler)

	server := &http.Server{
		Addr:              container.Config.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		container.Logger.Info("api listening", "addr", container.Config.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		container.Logger.Info("shutdown requested")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			container.Logger.Error("api failed", "error", err)
			os.Exit(1)
		}
	}

	container.GatewayRegistry.StartDraining()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), container.Config.Gateway.DrainTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		container.Logger.Error("api shutdown failed", "error", err)
		os.Exit(1)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, `{"error":"encode_failed"}`, http.StatusInternalServerError)
	}
}

func operationalTokenAuth(container *app.Container, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// SECURITY-HARDENING: health/readiness/stats endpoints are operational surfaces in production.
		if container.Config.Environment != "production" {
			next(w, r)
			return
		}
		token := bearerToken(r)
		expected := container.Config.Security.OperationalToken
		if expected == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			container.Logger.Warn("security operational endpoint access denied", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
		return ""
	}
	return header[len(prefix):]
}
