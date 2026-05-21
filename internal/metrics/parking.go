package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"

	pkgmetrics "parkir-pintar/pkg/metrics"
)

// ParkingMetrics holds domain-specific metrics for the parking system.
// It wraps the generic Metrics from pkg/metrics for access to the shared
// meter provider and adds parking-specific gauges and counters.
type ParkingMetrics struct {
	*pkgmetrics.Metrics

	ActiveParkingSessions otelmetric.Int64Gauge
	OccupiedSpots         otelmetric.Int64Gauge
	ReservationsTotal     otelmetric.Int64Counter
}

// NewParkingMetrics creates a ParkingMetrics instance on top of an existing
// generic Metrics. The meter is obtained from the provider embedded in Metrics.
func NewParkingMetrics(base *pkgmetrics.Metrics, meter otelmetric.Meter) (*ParkingMetrics, error) {
	pm := &ParkingMetrics{Metrics: base}

	var err error

	pm.ActiveParkingSessions, err = meter.Int64Gauge("parking_active_sessions",
		otelmetric.WithDescription("Number of currently active parking sessions"),
	)
	if err != nil {
		return nil, err
	}

	pm.OccupiedSpots, err = meter.Int64Gauge("parking_occupied_spots",
		otelmetric.WithDescription("Number of currently occupied parking spots"),
	)
	if err != nil {
		return nil, err
	}

	pm.ReservationsTotal, err = meter.Int64Counter("parking_reservations_total",
		otelmetric.WithDescription("Total number of parking reservations by status"),
	)
	if err != nil {
		return nil, err
	}

	return pm, nil
}

func (pm *ParkingMetrics) SetActiveParkingSessions(ctx context.Context, count int64) {
	pm.ActiveParkingSessions.Record(ctx, count)
}

func (pm *ParkingMetrics) SetOccupiedSpots(ctx context.Context, count int64) {
	pm.OccupiedSpots.Record(ctx, count)
}

func (pm *ParkingMetrics) IncReservations(ctx context.Context, status string) {
	pm.ReservationsTotal.Add(ctx, 1,
		otelmetric.WithAttributes(
			attribute.String("status", status),
		),
	)
}
