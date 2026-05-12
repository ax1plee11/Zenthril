package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DBURL              string
	RedisURL           string
	JWTSecret          string
	HTTPAddr           string
	TLSCertFile        string
	TLSKeyFile         string
	NodeDomain         string
	NodePrivateKey     string
	CORSAllowedOrigins []string
	WSAllowedOrigins   []string
	AdminUserIDs       []string
	MetricsToken       string
	Environment        string
}

func Load() (*Config, error) {
	corsOrigins := splitCommaList(getEnv("CORS_ALLOWED_ORIGINS", ""))
	wsOrigins := splitCommaList(getEnv("WS_ALLOWED_ORIGINS", ""))
	adminIDs := splitCommaList(getEnv("ADMIN_USER_IDS", ""))

	cfg := &Config{
		DBURL:              getEnvWithFallback("DB_URL", "DATABASE_URL", ""),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		HTTPAddr:           getEnvWithFallback("HTTP_ADDR", "PORT", ":8080"),
		TLSCertFile:        getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:         getEnv("TLS_KEY_FILE", ""),
		NodeDomain:         getEnv("NODE_DOMAIN", "localhost"),
		NodePrivateKey:     getEnv("NODE_PRIVATE_KEY", ""),
		CORSAllowedOrigins: corsOrigins,
		WSAllowedOrigins:   wsOrigins,
		AdminUserIDs:       adminIDs,
		MetricsToken:       getEnv("METRICS_TOKEN", ""),
		Environment:        getEnv("ENVIRONMENT", "development"),
	}

	if cfg.DBURL == "" {
		return nil, fmt.Errorf("DB_URL or DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvWithFallback(primary, fallback, defaultVal string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	if v := os.Getenv(fallback); v != "" {
		if fallback == "PORT" {
			return ":" + v
		}
		return v
	}
	return defaultVal
}

func splitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
