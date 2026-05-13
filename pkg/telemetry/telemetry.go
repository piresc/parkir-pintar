// Package telemetry provides a unified initialization for all three OpenTelemetry
// signals (traces, metrics, logs) using OTLP gRPC exporters pointing to a single
// collector endpoint (e.g. Grafana Alloy).
package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds the configuration for the unified telemetry setup.
type Config struct {
	// ServiceName is the OTEL service.name resource attribute.
	ServiceName string

	// OTLPEndpoint is the gRPC address of the OTLP collector (e.g. "monitoring-alloy:4319").
	// If empty, noop providers are returned.
	OTLPEndpoint string

	// TraceSampleRate controls the TraceIDRatioBased sampler (0.0–1.0).
	TraceSampleRate float64

	// MetricInterval is the periodic reader push interval. Defaults to 15s.
	MetricInterval time.Duration
}

// Providers holds the initialized OTel SDK providers for all three signals.
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// Shutdown gracefully shuts down all providers, flushing pending data.
func (p *Providers) Shutdown(ctx context.Context) error {
	var firstErr error
	if p.LoggerProvider != nil {
		if err := p.LoggerProvider.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if p.MeterProvider != nil {
		if err := p.MeterProvider.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if p.TracerProvider != nil {
		if err := p.TracerProvider.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Init initializes all three OTel providers with OTLP gRPC exporters.
// If cfg.OTLPEndpoint is empty, noop-equivalent providers (no exporters) are returned.
func Init(ctx context.Context, cfg Config) (*Providers, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	if cfg.OTLPEndpoint == "" {
		// Return providers with no exporters (effectively noop).
		tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res))
		lp := sdklog.NewLoggerProvider(sdklog.WithResource(res))
		return &Providers{
			TracerProvider: tp,
			MeterProvider:  mp,
			LoggerProvider: lp,
		}, nil
	}

	dialOpts := grpc.WithTransportCredentials(insecure.NewCredentials())

	// --- Trace exporter ---
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithDialOption(dialOpts),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TraceSampleRate))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// --- Metric exporter ---
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlpmetricgrpc.WithDialOption(dialOpts),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	interval := cfg.MetricInterval
	if interval == 0 {
		interval = 15 * time.Second
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(interval))),
		sdkmetric.WithResource(res),
	)

	// --- Log exporter ---
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlploggrpc.WithDialOption(dialOpts),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	return &Providers{
		TracerProvider: tp,
		MeterProvider:  mp,
		LoggerProvider: lp,
	}, nil
}
