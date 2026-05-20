package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoOpTracer_ShouldReturnNoOpTracer_WhenCalled(t *testing.T) {
	tracer := NewNoOpTracer()

	assert.NotNil(t, tracer)
	_, ok := tracer.(*NoOpTracer)
	assert.True(t, ok, "expected *NoOpTracer")
}

func TestNoOpTracer_IsEnabled_ShouldReturnFalse(t *testing.T) {
	tracer := NewNoOpTracer()

	assert.False(t, tracer.IsEnabled())
}

func TestNoOpTracer_ShouldTrace_ShouldReturnFalse_WhenAnyPath(t *testing.T) {
	tracer := NewNoOpTracer()

	assert.False(t, tracer.ShouldTrace("/api/v1/users"))
	assert.False(t, tracer.ShouldTrace("/health"))
	assert.False(t, tracer.ShouldTrace(""))
}

func TestNoOpTracer_Shutdown_ShouldReturnNil(t *testing.T) {
	tracer := NewNoOpTracer()

	err := tracer.Shutdown(context.Background())

	assert.NoError(t, err)
}

func TestNoOpTracer_StartHTTPRequest_ShouldReturnContextAndTransaction(t *testing.T) {
	tracer := NewNoOpTracer()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	ctx, txn := tracer.StartHTTPRequest(req)

	assert.NotNil(t, ctx)
	assert.NotNil(t, txn)
	_, ok := txn.(*NoOpTransaction)
	assert.True(t, ok, "expected *NoOpTransaction")
}

func TestNoOpTracer_StartExternalCall_ShouldReturnContextAndEndFunc(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	newCtx, endFn := tracer.StartExternalCall(ctx, "example.com", "GET")

	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn() // should not panic
}

func TestNoOpTracer_StartMessage_ShouldReturnContextAndEndFunc(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	newCtx, endFn := tracer.StartMessage(ctx, "orders", "publish")

	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_StartDatabase_ShouldReturnContextAndEndFunc(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	newCtx, endFn := tracer.StartDatabase(ctx, "SELECT", "users")

	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_StartSegment_ShouldReturnContextAndEndFunc(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	newCtx, endFn := tracer.StartSegment(ctx, "custom-segment")

	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_InjectContext_ShouldNotPanic(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		tracer.InjectContext(ctx, nil)
	})
}

func TestNoOpTracer_ExtractContext_ShouldReturnSameContext(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	result := tracer.ExtractContext(ctx, nil)

	assert.Equal(t, ctx, result)
}

func TestNoOpTransaction_Context_ShouldReturnStoredContext(t *testing.T) {
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("key"), "value")
	txn := &NoOpTransaction{ctx: ctx}

	assert.Equal(t, ctx, txn.Context())
}

func TestNoOpTransaction_Methods_ShouldNotPanic(t *testing.T) {
	txn := &NoOpTransaction{ctx: context.Background()}

	assert.NotPanics(t, func() {
		txn.SetName("test")
		txn.AddAttribute("key", "value")
		txn.AddError(assert.AnError)
		txn.End()
	})
}

func TestNewTracer_ShouldReturnNoOpTracer_WhenConfigIsNil(t *testing.T) {
	tracer, err := NewTracer(nil)

	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_ShouldReturnNoOpTracer_WhenDisabled(t *testing.T) {
	cfg := &Config{Enabled: false, Exporter: "stdout"}

	tracer, err := NewTracer(cfg)

	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_ShouldReturnNoOpTracer_WhenNoOpForTesting(t *testing.T) {
	cfg := &Config{Enabled: true, NoOpForTesting: true, Exporter: "stdout"}

	tracer, err := NewTracer(cfg)

	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_ShouldReturnNoOpTracer_WhenExporterIsNoop(t *testing.T) {
	cfg := &Config{Enabled: true, Exporter: "noop"}

	tracer, err := NewTracer(cfg)

	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_ShouldReturnOtelTracer_WhenStdoutExporter(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		ServiceName: "test-service",
		SampleRate:  1.0,
		Exporter:    "stdout",
	}

	tracer, err := NewTracer(cfg)

	require.NoError(t, err)
	assert.IsType(t, &otelTracer{}, tracer)
	assert.True(t, tracer.IsEnabled())

	_ = tracer.Shutdown(context.Background())
}

func TestOtelTracer_ShouldTrace_ShouldReturnTrue_WhenPathNotExcluded(t *testing.T) {
	cfg := &Config{
		Enabled:      true,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Exporter:     "stdout",
		ExcludePaths: []string{"/health", "/metrics"},
	}
	tracer, err := NewTracer(cfg)
	require.NoError(t, err)
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	assert.True(t, tracer.ShouldTrace("/api/v1/users"))
	assert.True(t, tracer.ShouldTrace("/api/orders"))
}

func TestOtelTracer_ShouldTrace_ShouldReturnFalse_WhenPathExcluded(t *testing.T) {
	cfg := &Config{
		Enabled:      true,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Exporter:     "stdout",
		ExcludePaths: []string{"/health", "/metrics"},
	}
	tracer, err := NewTracer(cfg)
	require.NoError(t, err)
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	assert.False(t, tracer.ShouldTrace("/health"))
	assert.False(t, tracer.ShouldTrace("/health/ready"))
	assert.False(t, tracer.ShouldTrace("/metrics"))
}

func TestOtelTracer_ShouldTrace_ShouldReturnTrue_WhenNoExcludePaths(t *testing.T) {
	cfg := &Config{
		Enabled:      true,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Exporter:     "stdout",
		ExcludePaths: nil,
	}
	tracer, err := NewTracer(cfg)
	require.NoError(t, err)
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	assert.True(t, tracer.ShouldTrace("/anything"))
}
