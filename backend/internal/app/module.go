package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/redis/go-redis/v9"
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
//
// SECURITY: cmd/api is EXPERIMENTAL. It does not wire the full business-logic
// stack (messages, guilds, friends). Do NOT deploy it as a replacement for the
// legacy main.go until all services and session validation are ported.
// In production this module enforces:
//   - A real Redis-backed SessionValidator (NoopSessionValidator is rejected).
//   - A durable EventStore (InMemoryEventStore is rejected).
var Module = fx.Options(
	fx.Provide(
		loadConfig,
		newLogger,
		newCommandBus,
		newQueryBus,
		newEventStore,
		newShardManager,
		newEventBus,
		newRedisClient,
		newSessionValidator,
		newGatewayRegistry,
		newGatewayAuthenticator,
		newGatewayHandler,
		newContainer,
	),
	fx.Invoke(
		enforceProductionGuards,
		startGatewayHandler,
	),
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

func newEventStore(cfg config.Config) (cqrs.EventStore, error) {
	// SECURITY-HARDENING: InMemoryEventStore loses all data on restart.
	// Production must use a durable adapter (Postgres, Scylla).
	// enforceProductionGuards also checks this; this is a defence-in-depth guard.
	if strings.EqualFold(cfg.Environment, "production") {
		return nil, errors.New(
			"InMemoryEventStore is not allowed in production; " +
				"implement a durable EventStore and wire it here before deploying",
		)
	}
	// EVENT-SOURCING: memory store is alpha-only. Durable adapters should be
	// added before moving event-sourced aggregates to production traffic.
	return cqrs.NewInMemoryEventStore(), nil
}

// newRedisClient connects to Redis using the URL from config.
// The client is used by the session validator and optional rate limiter.
func newRedisClient(cfg config.Config) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return redis.NewClient(opts), nil
}

// newSessionValidator builds the production SessionValidator backed by Redis.
// SECURITY: NoopSessionValidator is forbidden in production — the production
// guard in enforceProductionGuards will catch any attempt to bypass this.
func newSessionValidator(cfg config.Config, rdb *redis.Client) gateway.SessionValidator {
	// In non-production environments we still return the real validator so that
	// development/staging runs are as close to production as possible.
	// Developers who truly need to skip session checks must set ENV=test explicitly.
	if strings.EqualFold(cfg.Environment, "test") {
		return gateway.NoopSessionValidator{}
	}
	// banLookup is nil intentionally: the next-gen gateway container does not
	// yet own a direct DB connection. The legacy HTTP middleware enforces global
	// bans on the HTTP layer. Once a Postgres pool is wired into this module
	// the ban lookup should be replaced with a real implementation.
	return gateway.NewRedisSessionValidator(rdb, nil)
}

func newGatewayRegistry(cfg config.Config, logger *slog.Logger) *gateway.Registry {
	return gateway.NewRegistry(gateway.RegistryOptions{
		NodeID:         cfg.Gateway.NodeID,
		MaxConnections: cfg.Gateway.MaxConnections,
		Logger:         logger,
	})
}

func newGatewayAuthenticator(cfg config.Config, validator gateway.SessionValidator) gateway.Authenticator {
	return gateway.NewJWTAuthenticator(cfg.Security.JWTSecret, validator)
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

// enforceProductionGuards prevents cmd/api from starting in production with
// known-unsafe configurations. This is a defence-in-depth check — individual
// providers also guard themselves, but a single enforcer makes the policy explicit.
//
// SECURITY: add new guards here whenever a new component has a known unsafe default.
func enforceProductionGuards(cfg config.Config, validator gateway.SessionValidator, store cqrs.EventStore, logger *slog.Logger) error {
	if !strings.EqualFold(cfg.Environment, "production") {
		if _, isNoop := validator.(gateway.NoopSessionValidator); isNoop {
			logger.Warn("security cmd/api is running with NoopSessionValidator; token blacklist and global bans are NOT enforced — do not deploy this to production")
		}
		return nil
	}

	// --- Production guards ---

	// Guard 1: NoopSessionValidator must never reach production.
	if _, isNoop := validator.(gateway.NoopSessionValidator); isNoop {
		return errors.New(
			"SECURITY: cmd/api cannot start in production with NoopSessionValidator; " +
				"wire RedisSessionValidator in internal/app/module.go",
		)
	}
	if cfg.Gateway.Role == "primary" && !hasGlobalBanLookup(validator) {
		return errors.New(
			"SECURITY: primary production gateway requires a global ban lookup; " +
				"use GATEWAY_ROLE=edge only for edge-only gateways that do not make ban decisions",
		)
	}

	// Guard 2: InMemoryEventStore must never reach production.
	if _, isMem := store.(*cqrs.InMemoryEventStore); isMem {
		return errors.New(
			"SECURITY: cmd/api cannot start in production with InMemoryEventStore; " +
				"implement a durable EventStore before deploying",
		)
	}

	// Guard 3: warn about experimental status — cmd/api does not yet route
	// guild, message, or friend traffic, so it must not be the sole entrypoint.
	logger.Warn(
		"EXPERIMENTAL: cmd/api is running in production mode; " +
			"this server does not yet implement the full business-logic API " +
			"(guilds, messages, friends). Ensure the legacy main.go is also running.",
	)

	return nil
}

type globalBanLookupAware interface {
	HasGlobalBanLookup() bool
}

func hasGlobalBanLookup(validator gateway.SessionValidator) bool {
	aware, ok := validator.(globalBanLookupAware)
	return ok && aware.HasGlobalBanLookup()
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
		OnStop: func(_ context.Context) error {
			// ARCHITECTURE: fx lifecycle must tear down bus subscriptions on shutdown.
			logger.Info("gateway handler stopped")
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
