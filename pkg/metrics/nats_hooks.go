package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// RecordNATSPublish increments the NATS published messages counter for the given subject.
func (m *Metrics) RecordNATSPublish(ctx context.Context, subject string) {
	m.NATSPublishedTotal.Add(ctx, 1,
		otelmetric.WithAttributes(
			attribute.String("subject", subject),
		),
	)
}

// RecordNATSConsume increments the NATS consumed messages counter and records
// the processing duration for the given subject.
func (m *Metrics) RecordNATSConsume(ctx context.Context, subject string, durationSeconds float64) {
	attrs := otelmetric.WithAttributes(
		attribute.String("subject", subject),
	)

	m.NATSConsumedTotal.Add(ctx, 1, attrs)
	m.NATSProcessingDuration.Record(ctx, durationSeconds, attrs)
}
