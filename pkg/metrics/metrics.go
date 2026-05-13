// Package metrics provides an OpenTelemetry-based metrics abstraction with
// OTLP gRPC push exporter for the ParkirPintar platform. It exposes HTTP, gRPC,
// database, NATS messaging, and business-domain metric instruments.
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

// Metrics holds all OTel metric instruments used across the application.
type Metrics struct {
	provider *sdkmetric.MeterProvider

	// HTTP metrics
	HTTPRequestsTotal   otelmetric.Int64Counter
	HTTPRequestDuration otelmetric.Float64Histogram
	HTTPResponseSize    otelmetric.Int64Histogram

	// gRPC metrics
	GRPCRequestsTotal   otelmetric.Int64Counter
	GRPCRequestDuration otelmetric.Float64Histogram

	// Database metrics
	DBQueryDuration otelmetric.Float64Histogram

	// NATS messaging metrics
	NATSPublishedTotal     otelmetric.Int64Counter
	NATSConsumedTotal      otelmetric.Int64Counter
	NATSProcessingDuration otelmetric.Float64Histogram

	// Business metrics
	ActiveParkingSessions otelmetric.Int64Gauge
	OccupiedSpots         otelmetric.Int64Gauge
	ReservationsTotal     otelmetric.Int64Counter
}

// NewMetrics creates a new Metrics instance with an OTLP gRPC periodic reader
// that pushes metrics to the given endpoint. If otlpEndpoint is empty, a noop
// meter provider is used (metrics are recorded but not exported).
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
		// No endpoint configured — use a provider with no reader (noop export).
		provider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
		)
	}

	meter := provider.Meter(serviceName)

	m := &Metrics{
		provider: provider,
	}

	// --- HTTP metrics ---

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

	// --- gRPC metrics ---

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

	// --- Database metrics ---

	m.DBQueryDuration, err = meter.Float64Histogram("db_query_duration_seconds",
		otelmetric.WithDescription("Database query duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// --- NATS messaging metrics ---

	m.NATSPublishedTotal, err = meter.Int64Counter("nats_messages_published_total",
		otelmetric.WithDescription("Total number of NATS messages published"),
	)
	if err != nil {
		return nil, err
	}

	m.NATSConsumedTotal, err = meter.Int64Counter("nats_messages_consumed_total",
		otelmetric.WithDescription("Total number of NATS messages consumed"),
	)
	if err != nil {
		return nil, err
	}

	m.NATSProcessingDuration, err = meter.Float64Histogram("nats_processing_duration_seconds",
		otelmetric.WithDescription("NATS message processing duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// --- Business metrics ---

	m.ActiveParkingSessions, err = meter.Int64Gauge("parking_active_sessions",
		otelmetric.WithDescription("Number of currently active parking sessions"),
	)
	if err != nil {
		return nil, err
	}

	m.OccupiedSpots, err = meter.Int64Gauge("parking_occupied_spots",
		otelmetric.WithDescription("Number of currently occupied parking spots"),
	)
	if err != nil {
		return nil, err
	}

	m.ReservationsTotal, err = meter.Int64Counter("parking_reservations_total",
		otelmetric.WithDescription("Total number of parking reservations by status"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Shutdown gracefully shuts down the MeterProvider, flushing any pending metrics.
func (m *Metrics) Shutdown(ctx context.Context) error {
	if m.provider == nil {
		return nil
	}
	return m.provider.Shutdown(ctx)
}

// RecordDBQuery is a convenience method to record a database query duration.
func (m *Metrics) RecordDBQuery(ctx context.Context, operation, table string, durationSeconds float64) {
	m.DBQueryDuration.Record(ctx, durationSeconds,
		otelmetric.WithAttributes(
			attribute.String("db.operation", operation),
			attribute.String("db.sql.table", table),
		),
	)
}

// SetActiveParkingSessions sets the current number of active parking sessions.
func (m *Metrics) SetActiveParkingSessions(ctx context.Context, count int64) {
	m.ActiveParkingSessions.Record(ctx, count)
}

// SetOccupiedSpots sets the current number of occupied parking spots.
func (m *Metrics) SetOccupiedSpots(ctx context.Context, count int64) {
	m.OccupiedSpots.Record(ctx, count)
}

// IncReservations increments the reservations counter with the given status.
func (m *Metrics) IncReservations(ctx context.Context, status string) {
	m.ReservationsTotal.Add(ctx, 1,
		otelmetric.WithAttributes(
			attribute.String("status", status),
		),
	)
}
