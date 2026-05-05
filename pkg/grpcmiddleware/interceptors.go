package grpcmiddleware

import (
	"log/slog"

	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/tracing"
)

// Interceptors is the single entry point for all gRPC interceptors. It holds
// shared dependencies that individual interceptor methods need.
type Interceptors struct {
	jwtSecret   string
	logger      *slog.Logger
	tracer      tracing.Tracer
	redisClient *redis.RedisClient
}

// NewInterceptors creates an Interceptors instance with the given dependencies.
// Nil-safe defaults are applied: slog.Default() for a nil logger and
// tracing.NewNoOpTracer() for a nil tracer. The redisClient may be nil if
// Redis-backed interceptors (idempotency) are not used.
func NewInterceptors(jwtSecret string, logger *slog.Logger, tracer tracing.Tracer, redisClient *redis.RedisClient) *Interceptors {
	if logger == nil {
		logger = slog.Default()
	}
	if tracer == nil {
		tracer = tracing.NewNoOpTracer()
	}
	return &Interceptors{
		jwtSecret:   jwtSecret,
		logger:      logger,
		tracer:      tracer,
		redisClient: redisClient,
	}
}
