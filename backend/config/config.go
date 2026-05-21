package config

import (
	"fmt"
	"net/url"
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
		Environment:        getEnvWithFallback("ENVIRONMENT", "APP_ENV", "development"),
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
	if cfg.Environment == "production" {
		if isPlaceholderSecret(cfg.JWTSecret) {
			return nil, fmt.Errorf("JWT_SECRET must be replaced in production")
		}
		if isPlaceholderSecret(cfg.DBURL) {
			return nil, fmt.Errorf("DB_URL must not contain placeholder secrets in production")
		}
		if cfg.MetricsToken == "" {
			return nil, fmt.Errorf("METRICS_TOKEN is required in production")
		}
		if len(cfg.MetricsToken) < 32 {
			return nil, fmt.Errorf("METRICS_TOKEN must be at least 32 characters in production")
		}
		if isPlaceholderSecret(cfg.MetricsToken) {
			return nil, fmt.Errorf("METRICS_TOKEN must be replaced in production")
		}
		if len(cfg.CORSAllowedOrigins) == 0 {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS is required in production")
		}
		if hasWildcard(cfg.CORSAllowedOrigins) {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain wildcard in production")
		}
		if len(cfg.WSAllowedOrigins) == 0 {
			return nil, fmt.Errorf("WS_ALLOWED_ORIGINS is required in production")
		}
		if hasWildcard(cfg.WSAllowedOrigins) {
			return nil, fmt.Errorf("WS_ALLOWED_ORIGINS cannot contain wildcard in production")
		}
	}
	if err := validateOrigins("CORS_ALLOWED_ORIGINS", cfg.CORSAllowedOrigins); err != nil {
		return nil, err
	}
	if err := validateOrigins("WS_ALLOWED_ORIGINS", cfg.WSAllowedOrigins); err != nil {
		return nil, err
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

func hasWildcard(values []string) bool {
	for _, value := range values {
		if value == "*" {
			return true
		}
	}
	return false
}

func validateOrigins(name string, origins []string) error {
	for _, origin := range origins {
		if origin == "*" {
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("%s must contain exact origins only, got %q", name, origin)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("%s origin must use http or https, got %q", name, origin)
		}
	}
	return nil
}

func isPlaceholderSecret(value string) bool {
	normalized := strings.ToLower(value)
	placeholders := []string{
		"change-me",
		"changeme",
		"replace-me",
		"example-password",
	}
	for _, placeholder := range placeholders {
		if strings.Contains(normalized, placeholder) {
			return true
		}
	}
	return false
}
