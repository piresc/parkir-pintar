package tracing

import (
	"context"
	"net/http"
)

type Tracer interface {
	StartHTTPRequest(r *http.Request) (context.Context, HTTPTransaction)

	StartExternalCall(ctx context.Context, host, method string) (context.Context, func())

	StartMessage(ctx context.Context, topic, operation string) (context.Context, func())

	StartDatabase(ctx context.Context, operation, table string) (context.Context, func())

	StartSegment(ctx context.Context, name string) (context.Context, func())

	InjectContext(ctx context.Context, carrier interface{})

	ExtractContext(ctx context.Context, carrier interface{}) context.Context

	IsEnabled() bool

	ShouldTrace(path string) bool

	Shutdown(ctx context.Context) error
}

type HTTPTransaction interface {
	Context() context.Context

	SetName(name string)

	AddAttribute(key, value string)

	AddError(err error)

	End()
}

type Config struct {
	Enabled bool

	ServiceName string

	SampleRate float64

	ExcludePaths []string

	Exporter string

	OTLPEndpoint string

	NewRelic NewRelicExporterConfig

	NoOpForTesting bool
}

type NewRelicExporterConfig struct {
	LicenseKey string
	Enabled    bool
}

// Otherwise, the exporter field selects the appropriate OTEL exporter.
func NewTracer(cfg *Config) (Tracer, error) {
	if cfg == nil {
		return NewNoOpTracer(), nil
	}

	if !cfg.Enabled || cfg.NoOpForTesting || cfg.Exporter == ExporterNoop {
		return NewNoOpTracer(), nil
	}

	return initOTELTracer(cfg)
}
