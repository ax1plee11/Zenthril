package config

import (
	"fmt"
	"os"
	"strings"
)

// Config содержит конфигурацию приложения, загружаемую из env-переменных.
type Config struct {
	// База данных
	DBURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret string

	// HTTP-сервер
	HTTPAddr string

	// TLS (опционально)
	TLSCertFile string
	TLSKeyFile  string

	// Федерация
	NodeDomain     string
	NodePrivateKey string

	// CORS: пусто = разрешить любой origin (*). Иначе список через запятую (точное совпадение).
	CORSAllowedOrigins []string
	// WebSocket Origin: пусто = как CORSAllowedOrigins; иначе свой список. Для десктопа с пустым Origin допускается подключение при не-* режиме (см. hub).
	WSAllowedOrigins []string

	// Администраторы: UUID пользователей, кому разрешены admin-эндпоинты.
	AdminUserIDs []string
}

// Load загружает конфигурацию из переменных окружения.
// Возвращает ошибку, если обязательные переменные не заданы.
func Load() (*Config, error) {
	corsOrigins := splitCommaList(getEnv("CORS_ALLOWED_ORIGINS", ""))
	wsOrigins := splitCommaList(getEnv("WS_ALLOWED_ORIGINS", ""))
	adminIDs := splitCommaList(getEnv("ADMIN_USER_IDS", ""))

	cfg := &Config{
		DBURL:              getEnv("DB_URL", ""),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		TLSCertFile:        getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:         getEnv("TLS_KEY_FILE", ""),
		NodeDomain:         getEnv("NODE_DOMAIN", "localhost"),
		NodePrivateKey:     getEnv("NODE_PRIVATE_KEY", ""),
		CORSAllowedOrigins: corsOrigins,
		WSAllowedOrigins:   wsOrigins,
		AdminUserIDs:       adminIDs,
	}

	if cfg.DBURL == "" {
		return nil, fmt.Errorf("DB_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// getEnv возвращает значение переменной окружения или defaultVal, если она не задана.
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
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
