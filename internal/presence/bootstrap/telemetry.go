package bootstrap

import (
	"context"
	"log/slog"
	"os"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/metrics"
	"parkir-pintar/pkg/telemetry"
	"parkir-pintar/pkg/tracing"
)

type Telemetry struct {
	Logger    *slog.Logger
	Tracer    tracing.Tracer
	Metrics   *metrics.Metrics
	Providers *telemetry.Providers
}

// initTelemetry initializes unified telemetry (traces, metrics, logs via OTLP).
func initTelemetry(cfg *config.Config) (*Telemetry, error) {
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)

	providers, telErr := telemetry.Init(context.Background(), telemetry.Config{
		ServiceName:     "parkir-pintar-presence",
		OTLPEndpoint:    otlpEndpoint,
		TraceSampleRate: cfg.Tracing.SampleRate,
	})
	if telErr != nil {
		slog.Warn("telemetry init failed, continuing with noop", slog.Any("error", telErr))
	}

	// Initialize logger (with OTLP log export if available).
	var log *slog.Logger
	if providers != nil && providers.LoggerProvider != nil {
		log = logger.NewLoggerWithProvider(cfg.Logger, providers.LoggerProvider)
	} else {
		log = logger.NewLogger(cfg.Logger)
	}

	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled: cfg.Tracing.Enabled, ServiceName: "parkir-pintar-presence",
		SampleRate: cfg.Tracing.SampleRate, Exporter: cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	metricsInst, err := metrics.NewMetrics("parkir-pintar-presence", otlpEndpoint)
	if err != nil {
		return nil, err
	}

	return &Telemetry{
		Logger:    log,
		Tracer:    tracer,
		Metrics:   metricsInst,
		Providers: providers,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
