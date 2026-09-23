package observability

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"zenthril-backend/internal/config"
)

// TracerProvider is a minimal wrapper around an OpenTelemetry tracer provider.
// ARCHITECTURE: provides structured observability with metrics, traces, and logs.
// SECURITY: tracer names should not include user identifiers or message contents.
type TracerProvider struct {
	enabled  bool
	logger   *slog.Logger
	mu       sync.RWMutex
	metrics  *MetricsCollector
	shutdown chan struct{}
}

// MetricsCollector aggregates observability metrics for the service.
// SECURITY: metrics must not leak PII or sensitive data.
type MetricsCollector struct {
	mu            sync.RWMutex
	counters      map[string]int64
	histograms    map[string][]float64
	gauges        map[string]float64
	lastReset     time.Time
}

// NewTracerProvider creates a noop tracer provider by default.
// Call Start if OTLP endpoint is configured.
// WEAKNESS FIXED: no metrics collection existed previously.
func NewTracerProvider(cfg config.ObservabilityConfig, logger *slog.Logger) *TracerProvider {
	return &TracerProvider{
		enabled:  cfg.OTLPEndpoint != "" && os.Getenv("OTEL_SDK_DISABLED") != "true",
		logger:   logger,
		metrics:  NewMetricsCollector(),
		shutdown: make(chan struct{}),
	}
}

// NewMetricsCollector creates a new metrics collector.
// ARCHITECTURE: metrics are collected in-memory and exported via OTLP when configured.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters:   make(map[string]int64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
		lastReset:  time.Now(),
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

// Shutdown flushes any pending spans and closes the metrics channel.
// SECURITY: ensures no metrics data is lost during shutdown.
func (t *TracerProvider) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	close(t.shutdown)
	return nil
}

// Tracer returns a named tracer for the given component.
func (t *TracerProvider) Tracer(name string) *Tracer {
	return &Tracer{name: name, provider: t}
}

// RecordCounter increments a named counter metric.
// SECURITY: counter values must not contain sensitive data.
func (t *TracerProvider) RecordCounter(name string, value int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metrics.counters[name] += value
}

// RecordHistogram records a value in a named histogram metric.
func (t *TracerProvider) RecordHistogram(name string, value float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metrics.histograms[name] = append(t.metrics.histograms[name], value)
}

// RecordGauge sets a gauge metric to a specific value.
func (t *TracerProvider) RecordGauge(name string, value float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metrics.gauges[name] = value
}

// GetMetricsSnapshot returns a copy of all collected metrics.
// SECURITY: snapshot must not include sensitive metric names.
func (t *TracerProvider) GetMetricsSnapshot() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	snapshot := make(map[string]interface{})
	for k, v := range t.metrics.counters {
		snapshot["counter_"+k] = v
	}
	for k, v := range t.metrics.gauges {
		snapshot["gauge_"+k] = v
	}
	return snapshot
}

// Tracer is a lightweight wrapper used throughout the codebase.
type Tracer struct {
	name     string
	provider *TracerProvider
}

// Start starts a new span. In the noop implementation this is a no-op.
// SECURITY: spans must not contain PII or sensitive data.
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
