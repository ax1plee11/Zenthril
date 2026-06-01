package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/fx"

	"zenthril-backend/internal/config"
	"zenthril-backend/internal/cqrs"
	"zenthril-backend/internal/event"
	"zenthril-backend/internal/gateway"
	"zenthril-backend/internal/repository"
)

type Container struct {
	Config          config.Config
	Logger          *slog.Logger
	CommandBus      *cqrs.CommandBus
	QueryBus        *cqrs.QueryBus
	EventStore      cqrs.EventStore
	EventBus        event.Bus
	ShardManager    *repository.ShardManager
	GatewayRegistry *gateway.Registry
	GatewayHandler  *gateway.Handler

	fxApp *fx.App
}

func New(ctx context.Context) (*Container, error) {
	var container *Container
	app := fx.New(
		fx.NopLogger,
		Module,
		fx.Populate(&container),
	)

	if err := app.Start(ctx); err != nil {
		return nil, fmt.Errorf("start fx app: %w", err)
	}
	container.fxApp = app
	return container, nil
}

func (c *Container) Close(ctx context.Context) error {
	c.GatewayRegistry.StartDraining()
	if c.fxApp != nil {
		if err := c.fxApp.Stop(ctx); err != nil {
			return fmt.Errorf("stop fx app: %w", err)
		}
	}
	if c.EventBus != nil {
		return c.EventBus.Close(ctx)
	}
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	level := slog.LevelInfo
	if cfg.Environment == "development" {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
		With("service", cfg.ServiceName, "node_id", cfg.Gateway.NodeID)
}
