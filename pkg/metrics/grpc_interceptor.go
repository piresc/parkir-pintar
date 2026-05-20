package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func (m *Metrics) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start).Seconds()
		grpcCode := status.Code(err).String()

		attrs := otelmetric.WithAttributes(
			attribute.String("method", info.FullMethod),
			attribute.String("grpc_code", grpcCode),
		)

		m.GRPCRequestsTotal.Add(ctx, 1, attrs)
		m.GRPCRequestDuration.Record(ctx, duration, attrs)

		return resp, err
	}
}
