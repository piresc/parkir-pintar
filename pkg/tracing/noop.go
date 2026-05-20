package tracing

import (
	"context"
	"net/http"
)

type NoOpTracer struct{}

func NewNoOpTracer() Tracer {
	return &NoOpTracer{}
}

func (n *NoOpTracer) StartHTTPRequest(r *http.Request) (context.Context, HTTPTransaction) {
	return r.Context(), &NoOpTransaction{ctx: r.Context()}
}

func (n *NoOpTracer) StartExternalCall(ctx context.Context, _, _ string) (context.Context, func()) {
	return ctx, func() {}
}

func (n *NoOpTracer) StartMessage(ctx context.Context, _, _ string) (context.Context, func()) {
	return ctx, func() {}
}

func (n *NoOpTracer) StartDatabase(ctx context.Context, _, _ string) (context.Context, func()) {
	return ctx, func() {}
}

func (n *NoOpTracer) StartSegment(ctx context.Context, _ string) (context.Context, func()) {
	return ctx, func() {}
}

func (n *NoOpTracer) InjectContext(_ context.Context, _ interface{}) {}

func (n *NoOpTracer) ExtractContext(ctx context.Context, _ interface{}) context.Context {
	return ctx
}

// IsEnabled always returns false.
func (n *NoOpTracer) IsEnabled() bool { return false }

// ShouldTrace always returns false.
func (n *NoOpTracer) ShouldTrace(_ string) bool { return false }

func (n *NoOpTracer) Shutdown(_ context.Context) error { return nil }

type NoOpTransaction struct {
	ctx context.Context
}

func (t *NoOpTransaction) Context() context.Context { return t.ctx }

func (t *NoOpTransaction) SetName(_ string) {}

func (t *NoOpTransaction) AddAttribute(_, _ string) {}

func (t *NoOpTransaction) AddError(_ error) {}

func (t *NoOpTransaction) End() {}
