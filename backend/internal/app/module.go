package app

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/fx"

	"zenthril-backend/internal/config"
	"zenthril-backend/internal/cqrs"
	"zenthril-backend/internal/event"
	"zenthril-backend/internal/gateway"
	"zenthril-backend/internal/repository"
)

// Module defines the next-generation API dependency graph.
//
// ARCHITECTURE: keep this module small while the legacy backend/main.go is
// still present. New services should enter through fx providers here first.
var Module = fx.Options(
	fx.Provide(
		loadConfig,
		newLogger,
		newCommandBus,
		newQueryBus,
		newEventStore,
		newShardManager,
		newEventBus,
		newGatewayRegistry,
		newGatewayAuthenticator,
		newGatewayHandler,
		newContainer,
	),
	fx.Invoke(startGatewayHandler),
)

func loadConfig() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func newShardManager(cfg config.Config) (*repository.ShardManager, error) {
	shards, err := repository.NewShardManagerFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create shard manager: %w", err)
	}
	return shards, nil
}

func newEventBus(cfg config.Config, logger *slog.Logger) (event.Bus, error) {
	bus, err := event.NewBusFromConfig(cfg.EventBus, logger)
	if err != nil {
		return nil, fmt.Errorf("create event bus: %w", err)
	}
	return bus, nil
}

func newCommandBus() *cqrs.CommandBus {
	// CQRS: write-side modules register commands here through fx providers.
	return cqrs.NewCommandBus()
}

func newQueryBus() *cqrs.QueryBus {
	// CQRS: read-side modules register projection/query handlers here.
	return cqrs.NewQueryBus()
}

func newEventStore() cqrs.EventStore {
	// EVENT-SOURCING: memory store is alpha-only. Durable adapters should be
	// added before moving event-sourced aggregates to production traffic.
	return cqrs.NewInMemoryEventStore()
}

func newGatewayRegistry(cfg config.Config, logger *slog.Logger) *gateway.Registry {
	return gateway.NewRegistry(gateway.RegistryOptions{
		NodeID:         cfg.Gateway.NodeID,
		MaxConnections: cfg.Gateway.MaxConnections,
		Logger:         logger,
	})
}

func newGatewayAuthenticator(cfg config.Config) gateway.Authenticator {
	return gateway.NewJWTAuthenticator(cfg.Security.JWTSecret)
}

func newGatewayHandler(
	cfg config.Config,
	registry *gateway.Registry,
	bus event.Bus,
	authenticator gateway.Authenticator,
	logger *slog.Logger,
) *gateway.Handler {
	return gateway.NewHandler(gateway.HandlerOptions{
		NodeID:         cfg.Gateway.NodeID,
		Registry:       registry,
		Bus:            bus,
		Authenticator:  authenticator,
		Logger:         logger,
		AllowedOrigins: cfg.Gateway.AllowedOrigins,
		ReadLimitBytes: cfg.Gateway.ReadLimitBytes,
		WriteTimeout:   cfg.Gateway.WriteTimeout,
		PongWait:       cfg.Gateway.PongWait,
		PingPeriod:     cfg.Gateway.PingPeriod,
	})
}

func startGatewayHandler(lc fx.Lifecycle, handler *gateway.Handler, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := handler.Start(ctx); err != nil {
				return fmt.Errorf("start gateway handler: %w", err)
			}
			logger.Info("gateway handler started")
			return nil
		},
	})
}

func newContainer(
	cfg config.Config,
	logger *slog.Logger,
	commandBus *cqrs.CommandBus,
	queryBus *cqrs.QueryBus,
	eventStore cqrs.EventStore,
	bus event.Bus,
	shards *repository.ShardManager,
	registry *gateway.Registry,
	handler *gateway.Handler,
) *Container {
	return &Container{
		Config:          cfg,
		Logger:          logger,
		CommandBus:      commandBus,
		QueryBus:        queryBus,
		EventStore:      eventStore,
		EventBus:        bus,
		ShardManager:    shards,
		GatewayRegistry: registry,
		GatewayHandler:  handler,
	}
}
