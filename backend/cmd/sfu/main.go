package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zenthril-backend/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	container, err := app.New(ctx)
	if err != nil {
		log.Fatalf("boot sfu: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := container.Close(shutdownCtx); err != nil {
			container.Logger.Error("close container", "error", err)
		}
	}()

	container.Logger.Info("sfu control plane started")
	<-ctx.Done()
	container.Logger.Info("sfu control plane stopped")
}
