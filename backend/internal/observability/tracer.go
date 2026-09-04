package observability

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"zenthril-backend/internal/config"
)

// TracerProvider is a minimal wrapper around an OpenTelemetry tracer provider.
// E2EE: this module does not handle message contents; it only emits span metadata
// such as operation names and dependency names.
type TracerProvider struct {
	enabled bool
	logger  *slog.Logger
	mu      sync.RWMutex
}

// NewTracerProvider creates a noop tracer provider by default.
// Call Start if OTLP endpoint is configured.
func NewTracerProvider(cfg config.ObservabilityConfig, logger *slog.Logger) *TracerProvider {
	return &TracerProvider{
		enabled: cfg.OTLPEndpoint != "" && os.Getenv("OTEL_SDK_DISABLED") != "true",
		logger:  logger,
	}
}

// Start initializes the tracer provider when an OTLP endpoint is configured.
// WEAKNESS FIXED: without an endpoint, the provider remains noop and does not
// export spans, avoiding panics or missing-dependency failures in alpha builds.
func (t *TracerProvider) Start(ctx context.Context) error {
	if !t.enabled {
		t.logger.Info("observability tracer disabled; no OTLP endpoint configured")
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	// ARCHITECTURE: real OTLP exporter wiring belongs here once the project
	// adopts go.opentelemetry.io/otel as a direct dependency.
	t.logger.Info("observability tracer enabled", "endpoint", "<configured>")
	return nil
}

// Shutdown flushes any pending spans.
func (t *TracerProvider) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return nil
}

// Tracer returns a named tracer for the given component.
// SECURITY: tracer names should not include user identifiers or message contents.
func (t *TracerProvider) Tracer(name string) *Tracer {
	return &Tracer{name: name, provider: t}
}

// Tracer is a lightweight wrapper used throughout the codebase.
type Tracer struct {
	name     string
	provider *TracerProvider
}

// Start starts a new span. In the noop implementation this is a no-op.
func (tr *Tracer) Start(ctx context.Context, spanName string, opts ...SpanOption) (context.Context, Span) {
	if !tr.provider.enabled {
		return ctx, &noopSpan{name: spanName}
	}
	// ARCHITECTURE: real span creation belongs here once OTel is wired.
	return ctx, &noopSpan{name: spanName}
}

// Span represents a tracing span.
type Span interface {
	End()
	SetAttributes(...any)
}

// SpanOption configures span behavior.
type SpanOption func(*spanConfig)

type spanConfig struct {
	attributes map[string]any
}

func WithAttributes(attributes map[string]any) SpanOption {
	return func(c *spanConfig) {
		c.attributes = attributes
	}
}

type noopSpan struct {
	name string
}

func (s *noopSpan) End() {}

func (s *noopSpan) SetAttributes(_ ...any) {}
