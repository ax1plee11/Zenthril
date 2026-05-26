package config

import (
	"testing"
)

const productionJWTSecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestSplitCommaList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"a", []string{"a"}},
		{"a,b", []string{"a", "b"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{",,,", nil},
	}
	for _, tt := range tests {
		got := splitCommaList(tt.in)
		if len(got) != len(tt.want) {
			t.Fatalf("splitCommaList(%q): len %d, want %d (%v vs %v)", tt.in, len(got), len(tt.want), got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("splitCommaList(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestLoad_RequiresDBAndJWT(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DB_URL and validation fails")
	}
}

func TestLoad_OK(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", productionJWTSecret)
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.com, https://b.com")
	t.Setenv("ADMIN_USER_IDS", " id-1 , id-2 ")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != "https://a.com" {
		t.Fatalf("CORSAllowedOrigins: %#v", cfg.CORSAllowedOrigins)
	}
	if len(cfg.AdminUserIDs) != 2 || cfg.AdminUserIDs[0] != "id-1" {
		t.Fatalf("AdminUserIDs: %#v", cfg.AdminUserIDs)
	}
}

func TestLoad_ProductionRequiresExplicitSecurityConfig(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", productionJWTSecret)

	_, err := Load()
	if err == nil {
		t.Fatal("expected production security validation error")
	}
}

func TestLoad_ProductionRejectsWildcardOrigins(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", productionJWTSecret)
	t.Setenv("METRICS_TOKEN", "metrics-token-minimum-32-chars!!!!")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("expected wildcard CORS validation error")
	}
}

func TestLoadRejectsWildcardOriginsOutsideProduction(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", productionJWTSecret)
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	_, err := Load()
	if err == nil {
		t.Fatal("expected wildcard origin validation error")
	}
}

func TestLoad_ProductionOK(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", productionJWTSecret)
	t.Setenv("METRICS_TOKEN", "metrics-token-minimum-32-chars!!!!")
	t.Setenv("OPERATIONAL_TOKEN", "operational-token-minimum-32-chars")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
}

func TestLoad_ProductionRejectsPlaceholderSecrets(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", "change-me-use-openssl-rand-hex-64")
	t.Setenv("METRICS_TOKEN", "metrics-token-minimum-32-chars!!!!")
	t.Setenv("OPERATIONAL_TOKEN", "operational-token-minimum-32-chars")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")

	if _, err := Load(); err == nil {
		t.Fatal("expected placeholder JWT validation error")
	}
}

func TestLoad_ProductionRejectsTestSecrets(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars!!")
	t.Setenv("METRICS_TOKEN", "metrics-token-minimum-32-chars!!!!")
	t.Setenv("OPERATIONAL_TOKEN", "operational-token-minimum-32-chars")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")

	if _, err := Load(); err == nil {
		t.Fatal("expected test JWT validation error")
	}
}

func TestLoad_ProductionFederationRequiresTokenWhenEnabled(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars!!")
	t.Setenv("METRICS_TOKEN", "metrics-token-minimum-32-chars!!!!")
	t.Setenv("OPERATIONAL_TOKEN", "operational-token-minimum-32-chars")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("FEDERATION_ENABLED", "true")

	if _, err := Load(); err == nil {
		t.Fatal("expected missing federation token validation error")
	}
}

func TestLoadRejectsOriginWithPath(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars!!")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com/path")

	if _, err := Load(); err == nil {
		t.Fatal("expected origin with path to be rejected")
	}
}

func TestSecurityWarnings(t *testing.T) {
	cfg := &Config{
		Environment: "development",
		JWTSecret:   "change-me-use-openssl-rand-hex-64",
	}

	warnings := cfg.SecurityWarnings()
	if len(warnings) == 0 {
		t.Fatal("expected development security warnings")
	}
}
