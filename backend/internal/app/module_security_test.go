package app

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"zenthril-backend/internal/config"
	"zenthril-backend/internal/cqrs"
	"zenthril-backend/internal/event"
	"zenthril-backend/internal/gateway"
)

// stubLogger returns a no-op slog.Logger for test isolation.
func stubLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestProductionGuardRejectsNoopSessionValidator ensures cmd/api cannot start
// in production with the insecure default validator.
//
// This test is the regression guard for the NoopSessionValidator security issue:
// if module.go is accidentally reverted to passing NoopSessionValidator{} into
// NewJWTAuthenticator, this test will catch it before a PR is merged.
func TestProductionGuardRejectsNoopSessionValidator(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Environment: "production"}
	store := cqrs.NewInMemoryEventStore() // we expect this guard to fire first or both

	// Noop validator must be rejected in production.
	err := enforceProductionGuards(cfg, gateway.NoopSessionValidator{}, store, stubLogger())
	if err == nil {
		t.Fatal("expected error when NoopSessionValidator is used in production, got nil")
	}
	if !strings.Contains(err.Error(), "NoopSessionValidator") {
		t.Errorf("error should mention NoopSessionValidator, got: %v", err)
	}
}

// TestProductionGuardRejectsInMemoryEventStore ensures the ephemeral event store
// cannot reach production.
func TestProductionGuardRejectsInMemoryEventStore(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Environment: "production", Gateway: config.GatewayConfig{Role: "edge"}}
	store := cqrs.NewInMemoryEventStore()
	realValidator := gateway.NewRedisSessionValidator(nil, nil) // nil redis OK for guard test

	err := enforceProductionGuards(cfg, realValidator, store, stubLogger())
	if err == nil {
		t.Fatal("expected error when InMemoryEventStore is used in production, got nil")
	}
	if !strings.Contains(err.Error(), "InMemoryEventStore") {
		t.Errorf("error should mention InMemoryEventStore, got: %v", err)
	}
}

// TestProductionGuardPassesWithRealValidator verifies that a properly wired
// production configuration passes the guards (aside from EventStore which still
// fails — this tests partial acceptance).
func TestProductionGuardWarnsButPassesWithRealValidatorAndRealStore(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Environment: "production", Gateway: config.GatewayConfig{Role: "primary"}}
	realValidator := &stubSessionValidator{hasBanLookup: true}

	// Use a stub event store that satisfies the interface but isn't InMemoryEventStore.
	stubStore := &stubEventStore{}

	err := enforceProductionGuards(cfg, realValidator, stubStore, stubLogger())
	if err != nil {
		t.Fatalf("expected no error with real validator and real store, got: %v", err)
	}
}

func TestProductionGuardRejectsPrimaryGatewayWithoutBanLookup(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Environment: "production", Gateway: config.GatewayConfig{Role: "primary"}}
	err := enforceProductionGuards(cfg, gateway.NewRedisSessionValidator(nil, nil), &stubEventStore{}, stubLogger())
	if err == nil {
		t.Fatal("expected primary gateway without ban lookup to fail")
	}
	if !strings.Contains(err.Error(), "global ban lookup") {
		t.Errorf("error should mention global ban lookup, got: %v", err)
	}
}

func TestProductionGuardAllowsEdgeGatewayWithoutBanLookup(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Environment: "production", Gateway: config.GatewayConfig{Role: "edge"}}
	err := enforceProductionGuards(cfg, gateway.NewRedisSessionValidator(nil, nil), &stubEventStore{}, stubLogger())
	if err != nil {
		t.Fatalf("expected edge gateway without ban lookup to pass, got: %v", err)
	}
}

// TestNonProductionGuardAllowsNoopWithWarning verifies that development/staging
// environments are allowed to run with NoopSessionValidator (with a warning).
func TestNonProductionGuardAllowsNoopWithWarning(t *testing.T) {
	t.Parallel()

	for _, env := range []string{"development", "staging", "test", ""} {
		cfg := config.Config{Environment: env}
		err := enforceProductionGuards(cfg, gateway.NoopSessionValidator{}, cqrs.NewInMemoryEventStore(), stubLogger())
		if err != nil {
			t.Errorf("env=%q: expected no error in non-production, got: %v", env, err)
		}
	}
}

// TestNewEventStoreRejectsProductionEnvironment verifies the provider-level guard.
func TestNewEventStoreRejectsProductionEnvironment(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Environment: "production"}
	_, err := newEventStore(cfg)
	if err == nil {
		t.Fatal("expected newEventStore to return error in production")
	}
}

// TestNewEventStoreAllowsNonProduction verifies the memory store is accepted outside production.
func TestNewEventStoreAllowsNonProduction(t *testing.T) {
	t.Parallel()

	for _, env := range []string{"development", "staging", "test", ""} {
		cfg := config.Config{Environment: env}
		store, err := newEventStore(cfg)
		if err != nil {
			t.Errorf("env=%q: expected no error, got: %v", env, err)
		}
		if store == nil {
			t.Errorf("env=%q: expected non-nil store", env)
		}
	}
}

// TestNewSessionValidatorReturnsNoopOnlyForTestEnv verifies that only the
// explicit "test" environment gets a NoopSessionValidator.
func TestNewSessionValidatorReturnsNoopOnlyForTestEnv(t *testing.T) {
	t.Parallel()

	for _, env := range []string{"development", "staging", ""} {
		cfg := config.Config{
			Environment: env,
			Redis:       config.RedisConfig{URL: "redis://localhost:6379"},
		}
		// newSessionValidator needs a redis.Client — pass nil to test the type branch only.
		// We call the function with a real client only in integration tests.
		// Here we verify the logic branch by checking env="test" separately.
		validator := newSessionValidator(cfg, nil) // nil redis client — not dialled
		if _, isNoop := validator.(gateway.NoopSessionValidator); isNoop {
			t.Errorf("env=%q: got NoopSessionValidator but expected RedisSessionValidator", env)
		}
	}

	// Only "test" environment should return NoopSessionValidator.
	testCfg := config.Config{Environment: "test"}
	testValidator := newSessionValidator(testCfg, nil)
	if _, isNoop := testValidator.(gateway.NoopSessionValidator); !isNoop {
		t.Error("env=test: expected NoopSessionValidator")
	}
}

// stubEventStore is a minimal EventStore that satisfies the interface without
// being an InMemoryEventStore. Used to test the production guard path.
type stubEventStore struct{}

func (s *stubEventStore) Append(_ context.Context, _ string, _ int64, _ []event.Event) ([]cqrs.StoredEvent, error) {
	return nil, nil
}

func (s *stubEventStore) Load(_ context.Context, _ string) ([]cqrs.StoredEvent, error) {
	return nil, nil
}

type stubSessionValidator struct {
	hasBanLookup bool
}

func (s *stubSessionValidator) IsTokenBlacklisted(context.Context, string) (bool, error) {
	return false, nil
}

func (s *stubSessionValidator) IsGloballyBanned(context.Context, string) (bool, error) {
	return false, nil
}

func (s *stubSessionValidator) HasGlobalBanLookup() bool {
	return s.hasBanLookup
}
