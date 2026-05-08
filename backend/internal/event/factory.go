package event

import (
	"fmt"
	"log/slog"
	"strings"

	"zenthril-backend/internal/config"
)

func NewBusFromConfig(cfg config.EventBusConfig, logger *slog.Logger) (Bus, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "", "memory":
		return NewMemoryBus(), nil
	case "kafka", "redpanda":
		return NewKafkaBus(cfg, logger)
	case "nats", "jetstream":
		return nil, fmt.Errorf("event bus driver %q requires the nats jetstream adapter phase", cfg.Driver)
	default:
		return nil, fmt.Errorf("unsupported event bus driver %q", cfg.Driver)
	}
}
