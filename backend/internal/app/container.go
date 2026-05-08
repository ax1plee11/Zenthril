package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"zenthril-backend/internal/config"
	"zenthril-backend/internal/event"
	"zenthril-backend/internal/gateway"
	"zenthril-backend/internal/repository"
)

type Container struct {
	Config          config.Config
	Logger          *slog.Logger
	EventBus        event.Bus
	ShardManager    *repository.ShardManager
	GatewayRegistry *gateway.Registry
	GatewayHandler  *gateway.Handler
}

func New(ctx context.Context) (*Container, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger := newLogger(cfg)
	shards, err := repository.NewShardManagerFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create shard manager: %w", err)
	}

	bus, err := event.NewBusFromConfig(cfg.EventBus, logger)
	if err != nil {
		return nil, fmt.Errorf("create event bus: %w", err)
	}
	registry := gateway.NewRegistry(gateway.RegistryOptions{
		NodeID:         cfg.Gateway.NodeID,
		MaxConnections: cfg.Gateway.MaxConnections,
		Logger:         logger,
	})
	handler := gateway.NewHandler(gateway.HandlerOptions{
		NodeID:         cfg.Gateway.NodeID,
		Registry:       registry,
		Bus:            bus,
		Authenticator:  gateway.NewJWTAuthenticator(cfg.Security.JWTSecret),
		Logger:         logger,
		AllowedOrigins: cfg.Gateway.AllowedOrigins,
		ReadLimitBytes: cfg.Gateway.ReadLimitBytes,
		WriteTimeout:   cfg.Gateway.WriteTimeout,
		PongWait:       cfg.Gateway.PongWait,
		PingPeriod:     cfg.Gateway.PingPeriod,
	})
	if err := handler.Start(ctx); err != nil {
		return nil, fmt.Errorf("start gateway handler: %w", err)
	}

	return &Container{
		Config:          cfg,
		Logger:          logger,
		EventBus:        bus,
		ShardManager:    shards,
		GatewayRegistry: registry,
		GatewayHandler:  handler,
	}, nil
}

func (c *Container) Close(ctx context.Context) error {
	c.GatewayRegistry.StartDraining()
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
