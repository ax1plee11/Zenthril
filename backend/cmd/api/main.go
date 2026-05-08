package main

import (
	"context"
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

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ready",
			"shards": len(container.ShardManager.Shards()),
		})
	})
	router.Get("/api/v2/gateway/stats", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, container.GatewayRegistry.Stats())
	})
	router.Handle("/ws", container.GatewayHandler)

	server := &http.Server{
		Addr:              container.Config.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
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
