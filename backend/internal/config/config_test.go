package config

import "testing"

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
	t.Setenv("APP_ENV", "production")

	if _, err := Load(); err == nil {
		t.Fatal("expected production JWT validation error")
	}
}
