package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Exporter type constants.
const (
	ExporterStdout = "stdout"
	ExporterNoop   = "noop"
)

// newExporter creates a SpanExporter based on the Config.Exporter value.
//
//   - "stdout"    → stdouttrace exporter for local development
//   - "otlp"      → OTLP HTTP exporter for any OTEL-compatible collector
//   - "otlp-grpc" → OTLP gRPC exporter (e.g. for Grafana Alloy)
//   - "newrelic"  → OTLP HTTP exporter pointed at New Relic's OTLP endpoint
//   - "noop"      → returns nil (caller should use NoOpTracer instead)
func newExporter(cfg *Config) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case ExporterStdout:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())

	case "otlp":
		if cfg.OTLPEndpoint == "" {
			return nil, fmt.Errorf("tracing: OTLP endpoint is required for otlp exporter")
		}
		return otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
			otlptracehttp.WithInsecure(),
		)

	case "otlp-grpc":
		if cfg.OTLPEndpoint == "" {
			return nil, fmt.Errorf("tracing: OTLP endpoint is required for otlp-grpc exporter")
		}
		conn, err := grpc.NewClient(
			cfg.OTLPEndpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("tracing: failed to create gRPC connection: %w", err)
		}
		return otlptracegrpc.New(
			context.Background(),
			otlptracegrpc.WithGRPCConn(conn),
		)

	case "newrelic":
		if cfg.NewRelic.LicenseKey == "" {
			return nil, fmt.Errorf("tracing: New Relic license key is required for newrelic exporter")
		}
		return otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpoint("otlp.nr-data.net"),
			otlptracehttp.WithHeaders(map[string]string{
				"api-key": cfg.NewRelic.LicenseKey,
			}),
		)

	case ExporterNoop, "":
		return nil, nil

	default:
		return nil, fmt.Errorf("tracing: unsupported exporter %q", cfg.Exporter)
	}
}
