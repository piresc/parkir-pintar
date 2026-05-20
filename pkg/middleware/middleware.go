// - Use keyed fields in struct literals to prevent breakages during refactors
package middleware

import (
	"log/slog"
	"sync"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/ratelimit"
	"parkir-pintar/pkg/tracing"
)

type Middleware struct {
	config *config.Config
	logger *slog.Logger
	tracer tracing.Tracer

	mu           sync.Mutex
	rateStore    *ratelimit.Store
	rateStoreCfg RateLimitConfig
}

func NewMiddleware(cfg *config.Config, logger *slog.Logger, tracer tracing.Tracer) *Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	if tracer == nil {
		tracer = tracing.NewNoOpTracer()
	}
	return &Middleware{
		config: cfg,
		logger: logger,
		tracer: tracer,
	}
}

func (m *Middleware) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rateStore != nil {
		m.rateStore.Stop()
	}
}
