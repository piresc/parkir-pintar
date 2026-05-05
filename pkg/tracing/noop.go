package tracing

import (
	"context"
	"net/http"
)

// NoOpTracer satisfies the Tracer interface with no-op behavior.
// Used when tracing is disabled or during testing.
type NoOpTracer struct{}

// NewNoOpTracer returns a Tracer that does nothing.
func NewNoOpTracer() Tracer {
	return &NoOpTracer{}
}

// StartHTTPRequest returns the request context and a no-op transaction.
func (n *NoOpTracer) StartHTTPRequest(r *http.Request) (context.Context, HTTPTransaction) {
	return r.Context(), &NoOpTransaction{ctx: r.Context()}
}

// StartExternalCall returns the context and a no-op end function.
func (n *NoOpTracer) StartExternalCall(ctx context.Context, _, _ string) (context.Context, func()) {
	return ctx, func() {}
}

// StartMessage returns the context and a no-op end function.
func (n *NoOpTracer) StartMessage(ctx context.Context, _, _ string) (context.Context, func()) {
	return ctx, func() {}
}

// StartDatabase returns the context and a no-op end function.
func (n *NoOpTracer) StartDatabase(ctx context.Context, _, _ string) (context.Context, func()) {
	return ctx, func() {}
}

// StartSegment returns the context and a no-op end function.
func (n *NoOpTracer) StartSegment(ctx context.Context, _ string) (context.Context, func()) {
	return ctx, func() {}
}

// InjectContext is a no-op.
func (n *NoOpTracer) InjectContext(_ context.Context, _ interface{}) {}

// ExtractContext returns the context unchanged.
func (n *NoOpTracer) ExtractContext(ctx context.Context, _ interface{}) context.Context {
	return ctx
}

// IsEnabled always returns false.
func (n *NoOpTracer) IsEnabled() bool { return false }

// ShouldTrace always returns false.
func (n *NoOpTracer) ShouldTrace(_ string) bool { return false }

// Shutdown is a no-op and returns nil.
func (n *NoOpTracer) Shutdown(_ context.Context) error { return nil }

// NoOpTransaction satisfies the HTTPTransaction interface with no-op behavior.
type NoOpTransaction struct {
	ctx context.Context
}

// Context returns the stored context.
func (t *NoOpTransaction) Context() context.Context { return t.ctx }

// SetName is a no-op.
func (t *NoOpTransaction) SetName(_ string) {}

// AddAttribute is a no-op.
func (t *NoOpTransaction) AddAttribute(_, _ string) {}

// AddError is a no-op.
func (t *NoOpTransaction) AddError(_ error) {}

// End is a no-op.
func (t *NoOpTransaction) End() {}
