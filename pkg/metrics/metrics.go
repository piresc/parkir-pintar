package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Metrics struct {
	provider *sdkmetric.MeterProvider
	meter    otelmetric.Meter

	HTTPRequestsTotal   otelmetric.Int64Counter
	HTTPRequestDuration otelmetric.Float64Histogram
	HTTPResponseSize    otelmetric.Int64Histogram

	GRPCRequestsTotal   otelmetric.Int64Counter
	GRPCRequestDuration otelmetric.Float64Histogram

	DBQueryDuration otelmetric.Float64Histogram
}

func NewMetrics(serviceName string, otlpEndpoint string) (*Metrics, error) {
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	var provider *sdkmetric.MeterProvider

	if otlpEndpoint != "" {
		exporter, exporterErr := otlpmetricgrpc.New(
			context.Background(),
			otlpmetricgrpc.WithEndpoint(otlpEndpoint),
			otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			otlpmetricgrpc.WithInsecure(),
		)
		if exporterErr != nil {
			return nil, exporterErr
		}

		provider = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(15*time.Second))),
			sdkmetric.WithResource(res),
		)
	} else {
		provider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
		)
	}

	meter := provider.Meter(serviceName)

	m := &Metrics{
		provider: provider,
		meter:    meter,
	}

	m.HTTPRequestsTotal, err = meter.Int64Counter("http_requests_total",
		otelmetric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	m.HTTPRequestDuration, err = meter.Float64Histogram("http_request_duration_seconds",
		otelmetric.WithDescription("HTTP request duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.HTTPResponseSize, err = meter.Int64Histogram("http_response_size_bytes",
		otelmetric.WithDescription("HTTP response size in bytes"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	m.GRPCRequestsTotal, err = meter.Int64Counter("grpc_requests_total",
		otelmetric.WithDescription("Total number of gRPC requests"),
	)
	if err != nil {
		return nil, err
	}

	m.GRPCRequestDuration, err = meter.Float64Histogram("grpc_request_duration_seconds",
		otelmetric.WithDescription("gRPC request duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.DBQueryDuration, err = meter.Float64Histogram("db_query_duration_seconds",
		otelmetric.WithDescription("Database query duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Meter returns the OpenTelemetry meter for creating additional instruments.
// Domain-specific packages (e.g. internal/metrics) use this to register their
// own counters and gauges on the same meter provider.
func (m *Metrics) Meter() otelmetric.Meter {
	return m.meter
}

func (m *Metrics) Shutdown(ctx context.Context) error {
	if m.provider == nil {
		return nil
	}
	return m.provider.Shutdown(ctx)
}

func (m *Metrics) RecordDBQuery(ctx context.Context, operation, table string, durationSeconds float64) {
	m.DBQueryDuration.Record(ctx, durationSeconds,
		otelmetric.WithAttributes(
			attribute.String("db.operation", operation),
			attribute.String("db.sql.table", table),
		),
	)
}
