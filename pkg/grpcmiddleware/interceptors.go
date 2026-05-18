package grpcmiddleware

import (
	"log/slog"

	"parkir-pintar/pkg/ratelimit"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/tracing"
)

// Interceptors is the single entry point for all gRPC interceptors. It holds
// shared dependencies that individual interceptor methods need.
type Interceptors struct {
	jwtSecret      string
	logger         *slog.Logger
	tracer         tracing.Tracer
	redisClient    *redis.Client
	rateLimitStore *ratelimit.Store
}

// Shutdown releases resources held by the interceptors, including stopping
// the background cleanup goroutine of the rate limit store.
func (i *Interceptors) Shutdown() {
	if i.rateLimitStore != nil {
		i.rateLimitStore.Stop()
	}
}

// NewInterceptors creates an Interceptors instance with the given dependencies.
// Nil-safe defaults are applied: slog.Default() for a nil logger and
// tracing.NewNoOpTracer() for a nil tracer. The redisClient may be nil if
// Redis-backed interceptors (idempotency) are not used.
func NewInterceptors(jwtSecret string, logger *slog.Logger, tracer tracing.Tracer, redisClient *redis.Client) *Interceptors {
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
