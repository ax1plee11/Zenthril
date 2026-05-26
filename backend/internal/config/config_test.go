package config

import "testing"

const productionJWTSecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestLoadUsesSingleDBURLAsPrimaryShard(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/zenthril")
	t.Setenv("JWT_SECRET", "dev-secret")
	t.Setenv("APP_ENV", "development")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Postgres.Shards) != 1 {
		t.Fatalf("len(shards) = %d, want 1", len(cfg.Postgres.Shards))
	}
	if cfg.Postgres.Shards[0].ID != "primary" {
		t.Fatalf("primary shard id = %q", cfg.Postgres.Shards[0].ID)
	}
}

func TestLoadParsesPostgresShards(t *testing.T) {
	t.Setenv("POSTGRES_SHARDS", "a=postgres://a,b=postgres://b")
	t.Setenv("JWT_SECRET", "dev-secret")
	t.Setenv("APP_ENV", "development")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Postgres.Shards) != 2 {
		t.Fatalf("len(shards) = %d, want 2", len(cfg.Postgres.Shards))
	}
	if cfg.Postgres.Shards[1].ID != "b" {
		t.Fatalf("second shard id = %q", cfg.Postgres.Shards[1].ID)
	}
}

func TestProductionRequiresStrongJWTSecret(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/zenthril")
	t.Setenv("JWT_SECRET", "short")
	t.Setenv("METRICS_TOKEN", "metrics-token-minimum-32-chars!!!!")
	t.Setenv("APP_ENV", "production")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")

	if _, err := Load(); err == nil {
		t.Fatal("expected production JWT validation error")
	}
}

func TestProductionRequiresGatewayOrigins(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/zenthril")
	t.Setenv("JWT_SECRET", productionJWTSecret)
	t.Setenv("METRICS_TOKEN", "metrics-token-minimum-32-chars!!!!")
	t.Setenv("APP_ENV", "production")

	if _, err := Load(); err == nil {
		t.Fatal("expected production origin validation error")
	}
}

func TestProductionRequiresOperationalToken(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/zenthril")
	t.Setenv("JWT_SECRET", productionJWTSecret)
	t.Setenv("APP_ENV", "production")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")

	if _, err := Load(); err == nil {
		t.Fatal("expected production operational token validation error")
	}
}

func TestProductionRejectsMemoryEventBus(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/zenthril")
	t.Setenv("JWT_SECRET", productionJWTSecret)
	t.Setenv("OPERATIONAL_TOKEN", "operational-token-minimum-32-chars")
	t.Setenv("APP_ENV", "production")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("EVENT_BUS_DRIVER", "memory")

	if _, err := Load(); err == nil {
		t.Fatal("expected production memory event bus validation error")
	}
}

func TestRejectsGatewayOriginWithPath(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/zenthril")
	t.Setenv("JWT_SECRET", "dev-secret")
	t.Setenv("APP_ENV", "development")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com/path")

	if _, err := Load(); err == nil {
		t.Fatal("expected origin with path validation error")
	}
}

func TestRejectsGatewayWildcardOutsideProduction(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/zenthril")
	t.Setenv("JWT_SECRET", "dev-secret")
	t.Setenv("APP_ENV", "development")
	t.Setenv("WS_ALLOWED_ORIGINS", "*")

	if _, err := Load(); err == nil {
		t.Fatal("expected gateway wildcard validation error")
	}
}
