// Package tracing provides an OpenTelemetry-native tracing abstraction with
// pluggable exporters (stdout, OTLP, New Relic) and a no-op fallback for
// testing and disabled tracing scenarios.
package tracing

import (
	"context"
	"net/http"
)

// Tracer wraps OTEL TracerProvider and provides convenience methods
// for starting spans in common scenarios (HTTP, DB, messaging, etc.).
type Tracer interface {
	// StartHTTPRequest begins a new HTTP transaction span from an incoming request.
	StartHTTPRequest(r *http.Request) (context.Context, HTTPTransaction)

	// StartExternalCall begins a span for an outbound HTTP/RPC call.
	StartExternalCall(ctx context.Context, host, method string) (context.Context, func())

	// StartMessage begins a span for a messaging operation (publish/consume).
	StartMessage(ctx context.Context, topic, operation string) (context.Context, func())

	// StartDatabase begins a span for a database operation.
	StartDatabase(ctx context.Context, operation, table string) (context.Context, func())

	// StartSegment begins a generic named span.
	StartSegment(ctx context.Context, name string) (context.Context, func())

	// InjectContext propagates trace context into an outbound carrier.
	InjectContext(ctx context.Context, carrier interface{})

	// ExtractContext extracts trace context from an inbound carrier.
	ExtractContext(ctx context.Context, carrier interface{}) context.Context

	// IsEnabled reports whether tracing is active.
	IsEnabled() bool

	// ShouldTrace reports whether the given path should be traced.
	ShouldTrace(path string) bool

	// Shutdown flushes pending spans and releases resources.
	Shutdown(ctx context.Context) error
}

// HTTPTransaction represents an in-flight HTTP span with helpers for
// setting attributes and recording errors.
type HTTPTransaction interface {
	// Context returns the span-enriched context.
	Context() context.Context

	// SetName overrides the span/transaction name.
	SetName(name string)

	// AddAttribute records a key-value attribute on the span.
	AddAttribute(key, value string)

	// AddError records an error on the span.
	AddError(err error)

	// End completes the span.
	End()
}

// Config holds tracing configuration.
type Config struct {
	// Enabled toggles tracing on/off globally.
	Enabled bool

	// ServiceName is the OTEL service.name resource attribute.
	ServiceName string

	// SampleRate controls the TraceIDRatioBased sampler (0.0–1.0).
	SampleRate float64

	// ExcludePaths lists URL paths that should not be traced (e.g. health checks).
	ExcludePaths []string

	// Exporter selects the span exporter: "stdout", "otlp", "newrelic", "noop".
	Exporter string

	// OTLPEndpoint is the OTLP collector address (for "otlp" exporter).
	OTLPEndpoint string

	// NewRelic holds New Relic–specific exporter settings.
	NewRelic NewRelicExporterConfig

	// NoOpForTesting forces a NoOpTracer regardless of other settings.
	NoOpForTesting bool
}

// NewRelicExporterConfig holds configuration for the New Relic OTLP exporter.
type NewRelicExporterConfig struct {
	LicenseKey string
	Enabled    bool
}

// NewTracer creates a Tracer implementation based on the provided Config.
// When tracing is disabled or NoOpForTesting is set, a NoOpTracer is returned.
// Otherwise, the exporter field selects the appropriate OTEL exporter.
func NewTracer(cfg *Config) (Tracer, error) {
	if cfg == nil {
		return NewNoOpTracer(), nil
	}

	if !cfg.Enabled || cfg.NoOpForTesting || cfg.Exporter == "noop" {
		return NewNoOpTracer(), nil
	}

	return initOTELTracer(cfg)
}
