// Package middleware provides a unified set of Gin middleware for the
// parkir-pintar. All middleware is accessed via the Middleware struct,
// which holds shared dependencies (config, logger, tracer).
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Use keyed fields in struct literals to prevent breakages during refactors
// - Keep interfaces small and define them where they're used
// - Use context.Context as first parameter for consistency
package middleware

import (
	"log/slog"
	"sync"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/tracing"
)

// Middleware is the single entry point for all HTTP middleware. It holds
// shared dependencies that individual middleware handlers need.
type Middleware struct {
	config *config.Config
	logger *slog.Logger
	tracer tracing.Tracer

	mu           sync.Mutex
	rateStore    *rateLimitStore
	rateStoreCfg RateLimitConfig
}

// NewMiddleware creates a Middleware instance with the given dependencies.
// All middleware methods are accessed through this struct.
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
