package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment   string
	ServiceName   string
	HTTPAddr      string
	PublicBaseURL string

	Gateway       GatewayConfig
	Postgres      PostgresConfig
	Redis         RedisConfig
	EventBus      EventBusConfig
	Scylla        ScyllaConfig
	ObjectStore   ObjectStoreConfig
	Security      SecurityConfig
	Sharding      ShardingConfig
	Observability ObservabilityConfig
}

type GatewayConfig struct {
	NodeID         string
	PublicAddr     string
	AllowedOrigins []string
	MaxConnections int
	ReadLimitBytes int64
	WriteTimeout   time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	DrainTimeout   time.Duration
}

type PostgresConfig struct {
	Shards []PostgresShardConfig
}

type PostgresShardConfig struct {
	ID     string
	DSN    string
	Weight int
}

type RedisConfig struct {
	URL         string
	ClusterURLs []string
}

type EventBusConfig struct {
	Driver        string
	KafkaBrokers  []string
	NATSURL       string
	ConsumerGroup string
}

type ScyllaConfig struct {
	Hosts    []string
	Keyspace string
}

type ObjectStoreConfig struct {
	Endpoint  string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

type SecurityConfig struct {
	JWTSecret        string
	OperationalToken string
	VaultAddr        string
}

type ShardingConfig struct {
	VirtualNodes int
}

type ObservabilityConfig struct {
	OTLPEndpoint string
}

func Load() (Config, error) {
	httpAddr := envWithFallback("HTTP_ADDR", "PORT", ":8080")
	nodeID := env("NODE_ID", defaultNodeID())
	dbURL := envWithFallback("DB_URL", "DATABASE_URL", "")

	cfg := Config{
		Environment:   env("APP_ENV", "development"),
		ServiceName:   env("SERVICE_NAME", "zenthril-api"),
		HTTPAddr:      httpAddr,
		PublicBaseURL: env("PUBLIC_BASE_URL", "http://localhost:8080"),
		Gateway: GatewayConfig{
			NodeID:         nodeID,
			PublicAddr:     env("GATEWAY_PUBLIC_ADDR", httpAddr),
			AllowedOrigins: firstNonEmptyList(splitCommaList(os.Getenv("WS_ALLOWED_ORIGINS")), splitCommaList(os.Getenv("CORS_ALLOWED_ORIGINS"))),
			MaxConnections: envInt("GATEWAY_MAX_CONNECTIONS", 50000),
			ReadLimitBytes: int64(envInt("GATEWAY_READ_LIMIT_BYTES", 1<<20)),
			WriteTimeout:   envDuration("GATEWAY_WRITE_TIMEOUT", 10*time.Second),
			PongWait:       envDuration("GATEWAY_PONG_WAIT", 60*time.Second),
			PingPeriod:     envDuration("GATEWAY_PING_PERIOD", 45*time.Second),
			DrainTimeout:   envDuration("GATEWAY_DRAIN_TIMEOUT", 30*time.Second),
		},
		Postgres: PostgresConfig{
			Shards: parsePostgresShards(os.Getenv("POSTGRES_SHARDS"), dbURL),
		},
		Redis: RedisConfig{
			URL:         env("REDIS_URL", "redis://localhost:6379"),
			ClusterURLs: splitCommaList(os.Getenv("REDIS_CLUSTER_URLS")),
		},
		EventBus: EventBusConfig{
			Driver:        env("EVENT_BUS_DRIVER", "memory"),
			KafkaBrokers:  splitCommaList(os.Getenv("KAFKA_BROKERS")),
			NATSURL:       env("NATS_URL", ""),
			ConsumerGroup: env("EVENT_CONSUMER_GROUP", "zenthril-api"),
		},
		Scylla: ScyllaConfig{
			Hosts:    splitCommaList(os.Getenv("SCYLLA_HOSTS")),
			Keyspace: env("SCYLLA_KEYSPACE", "zenthril_messages"),
		},
		ObjectStore: ObjectStoreConfig{
			Endpoint:  env("S3_ENDPOINT", ""),
			Bucket:    env("S3_BUCKET", "zenthril-media"),
			Region:    env("S3_REGION", "auto"),
			AccessKey: env("S3_ACCESS_KEY_ID", ""),
			SecretKey: env("S3_SECRET_ACCESS_KEY", ""),
			UseSSL:    envBool("S3_USE_SSL", true),
		},
		Security: SecurityConfig{
			JWTSecret:        env("JWT_SECRET", ""),
			OperationalToken: firstNonEmpty(env("OPERATIONAL_TOKEN", ""), env("METRICS_TOKEN", "")),
			VaultAddr:        env("VAULT_ADDR", ""),
		},
		Sharding: ShardingConfig{
			VirtualNodes: envInt("SHARD_VIRTUAL_NODES", 128),
		},
		Observability: ObservabilityConfig{
			OTLPEndpoint: env("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		},
	}

	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if c.HTTPAddr == "" {
		return errors.New("HTTP_ADDR is required")
	}
	if c.Gateway.NodeID == "" {
		return errors.New("NODE_ID is required")
	}
	if c.Gateway.MaxConnections <= 0 {
		return errors.New("GATEWAY_MAX_CONNECTIONS must be positive")
	}
	if len(c.Postgres.Shards) == 0 {
		return errors.New("DB_URL/DATABASE_URL or POSTGRES_SHARDS is required")
	}
	for _, shard := range c.Postgres.Shards {
		if shard.ID == "" || shard.DSN == "" {
			return fmt.Errorf("postgres shard must have id and dsn: %#v", shard)
		}
		if _, err := url.Parse(shard.DSN); err != nil {
			return fmt.Errorf("parse postgres shard %q dsn: %w", shard.ID, err)
		}
	}
	if strings.EqualFold(c.Environment, "production") && len(c.Security.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 bytes in production")
	}
	if strings.EqualFold(c.Environment, "production") {
		if isPlaceholderSecret(c.Security.JWTSecret) {
			return errors.New("JWT_SECRET must be replaced in production")
		}
		if len(c.Security.OperationalToken) < 32 {
			return errors.New("OPERATIONAL_TOKEN or METRICS_TOKEN must be at least 32 bytes in production")
		}
		if isPlaceholderSecret(c.Security.OperationalToken) {
			return errors.New("OPERATIONAL_TOKEN or METRICS_TOKEN must be replaced in production")
		}
		// SECURITY: production websocket gateways must fail closed without exact allowed origins.
		if len(c.Gateway.AllowedOrigins) == 0 {
			return errors.New("WS_ALLOWED_ORIGINS or CORS_ALLOWED_ORIGINS is required in production")
		}
	}
	if hasWildcard(c.Gateway.AllowedOrigins) {
		return errors.New("gateway allowed origins cannot contain wildcard")
	}
	if err := validateExactOrigins("gateway allowed origins", c.Gateway.AllowedOrigins); err != nil {
		return err
	}
	if c.EventBus.Driver == "kafka" && len(c.EventBus.KafkaBrokers) == 0 {
		return errors.New("KAFKA_BROKERS is required when EVENT_BUS_DRIVER=kafka")
	}
	if c.EventBus.Driver == "nats" && c.EventBus.NATSURL == "" {
		return errors.New("NATS_URL is required when EVENT_BUS_DRIVER=nats")
	}
	return nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envWithFallback(primary, fallback, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(primary)); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv(fallback)); value != "" {
		if fallback == "PORT" && !strings.HasPrefix(value, ":") {
			return ":" + value
		}
		return value
	}
	return defaultValue
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCommaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func firstNonEmptyList(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func firstNonEmpty(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

func hasWildcard(values []string) bool {
	for _, value := range values {
		if value == "*" {
			return true
		}
	}
	return false
}

func validateExactOrigins(name string, origins []string) error {
	for _, origin := range origins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("%s must contain exact origins only, got %q", name, origin)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("%s origin must use http or https, got %q", name, origin)
		}
	}
	return nil
}

func parsePostgresShards(raw string, fallbackDSN string) []PostgresShardConfig {
	entries := splitCommaList(raw)
	if len(entries) == 0 {
		if strings.TrimSpace(fallbackDSN) == "" {
			return nil
		}
		return []PostgresShardConfig{{ID: "primary", DSN: fallbackDSN, Weight: 100}}
	}

	shards := make([]PostgresShardConfig, 0, len(entries))
	for _, entry := range entries {
		id, dsn, ok := strings.Cut(entry, "=")
		if !ok {
			id = fmt.Sprintf("shard-%d", len(shards))
			dsn = entry
		}
		id = strings.TrimSpace(id)
		dsn = strings.TrimSpace(dsn)
		if id == "" || dsn == "" {
			continue
		}
		shards = append(shards, PostgresShardConfig{ID: id, DSN: dsn, Weight: 100})
	}
	return shards
}

func defaultNodeID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

func isPlaceholderSecret(value string) bool {
	normalized := strings.ToLower(value)
	for _, placeholder := range []string{"change-me", "changeme", "replace-me", "example-password"} {
		if strings.Contains(normalized, placeholder) {
			return true
		}
	}
	return false
}
