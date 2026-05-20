package grpcmiddleware

import (
	"log/slog"

	"parkir-pintar/pkg/ratelimit"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/tracing"
)

type Interceptors struct {
	jwtSecret      string
	logger         *slog.Logger
	tracer         tracing.Tracer
	redisClient    *redis.Client
	rateLimitStore *ratelimit.Store
}

func (i *Interceptors) Shutdown() {
	if i.rateLimitStore != nil {
		i.rateLimitStore.Stop()
	}
}

// Nil-safe defaults are applied: slog.Default() for a nil logger and
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
