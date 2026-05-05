// Best practices applied (from bella-sdlc coding standards knowledge base):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - Table-driven tests for multiple scenarios
// - Tests are fast, isolated, repeatable, clear, and comprehensive
// - Uses testify/assert for assertions
// - Covers both happy paths and error cases

package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NoOpTracer tests
// ---------------------------------------------------------------------------

func TestNewNoOpTracer_ShouldReturnNoOpTracer_WhenCalled(t *testing.T) {
	// Arrange & Act
	tracer := NewNoOpTracer()

	// Assert
	assert.NotNil(t, tracer)
	_, ok := tracer.(*NoOpTracer)
	assert.True(t, ok, "expected *NoOpTracer")
}

func TestNoOpTracer_IsEnabled_ShouldReturnFalse(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()

	// Act & Assert
	assert.False(t, tracer.IsEnabled())
}

func TestNoOpTracer_ShouldTrace_ShouldReturnFalse_WhenAnyPath(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()

	// Act & Assert
	assert.False(t, tracer.ShouldTrace("/api/v1/users"))
	assert.False(t, tracer.ShouldTrace("/health"))
	assert.False(t, tracer.ShouldTrace(""))
}

func TestNoOpTracer_Shutdown_ShouldReturnNil(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()

	// Act
	err := tracer.Shutdown(context.Background())

	// Assert
	assert.NoError(t, err)
}

func TestNoOpTracer_StartHTTPRequest_ShouldReturnContextAndTransaction(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Act
	ctx, txn := tracer.StartHTTPRequest(req)

	// Assert
	assert.NotNil(t, ctx)
	assert.NotNil(t, txn)
	_, ok := txn.(*NoOpTransaction)
	assert.True(t, ok, "expected *NoOpTransaction")
}

func TestNoOpTracer_StartExternalCall_ShouldReturnContextAndEndFunc(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Act
	newCtx, endFn := tracer.StartExternalCall(ctx, "example.com", "GET")

	// Assert
	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn() // should not panic
}

func TestNoOpTracer_StartMessage_ShouldReturnContextAndEndFunc(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Act
	newCtx, endFn := tracer.StartMessage(ctx, "orders", "publish")

	// Assert
	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_StartDatabase_ShouldReturnContextAndEndFunc(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Act
	newCtx, endFn := tracer.StartDatabase(ctx, "SELECT", "users")

	// Assert
	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_StartSegment_ShouldReturnContextAndEndFunc(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Act
	newCtx, endFn := tracer.StartSegment(ctx, "custom-segment")

	// Assert
	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_InjectContext_ShouldNotPanic(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Act & Assert — should not panic with any carrier type
	assert.NotPanics(t, func() {
		tracer.InjectContext(ctx, nil)
	})
}

func TestNoOpTracer_ExtractContext_ShouldReturnSameContext(t *testing.T) {
	// Arrange
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Act
	result := tracer.ExtractContext(ctx, nil)

	// Assert
	assert.Equal(t, ctx, result)
}

// ---------------------------------------------------------------------------
// NoOpTransaction tests
// ---------------------------------------------------------------------------

func TestNoOpTransaction_Context_ShouldReturnStoredContext(t *testing.T) {
	// Arrange
	ctx := context.WithValue(context.Background(), "key", "value")
	txn := &NoOpTransaction{ctx: ctx}

	// Act & Assert
	assert.Equal(t, ctx, txn.Context())
}

func TestNoOpTransaction_Methods_ShouldNotPanic(t *testing.T) {
	// Arrange
	txn := &NoOpTransaction{ctx: context.Background()}

	// Act & Assert — all methods are no-ops and should not panic
	assert.NotPanics(t, func() {
		txn.SetName("test")
		txn.AddAttribute("key", "value")
		txn.AddError(assert.AnError)
		txn.End()
	})
}

// ---------------------------------------------------------------------------
// NewTracer factory tests
// ---------------------------------------------------------------------------

func TestNewTracer_ShouldReturnNoOpTracer_WhenConfigIsNil(t *testing.T) {
	// Arrange & Act
	tracer, err := NewTracer(nil)

	// Assert
	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_ShouldReturnNoOpTracer_WhenDisabled(t *testing.T) {
	// Arrange
	cfg := &Config{Enabled: false, Exporter: "stdout"}

	// Act
	tracer, err := NewTracer(cfg)

	// Assert
	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_ShouldReturnNoOpTracer_WhenNoOpForTesting(t *testing.T) {
	// Arrange
	cfg := &Config{Enabled: true, NoOpForTesting: true, Exporter: "stdout"}

	// Act
	tracer, err := NewTracer(cfg)

	// Assert
	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_ShouldReturnNoOpTracer_WhenExporterIsNoop(t *testing.T) {
	// Arrange
	cfg := &Config{Enabled: true, Exporter: "noop"}

	// Act
	tracer, err := NewTracer(cfg)

	// Assert
	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_ShouldReturnOtelTracer_WhenStdoutExporter(t *testing.T) {
	// Arrange
	cfg := &Config{
		Enabled:     true,
		ServiceName: "test-service",
		SampleRate:  1.0,
		Exporter:    "stdout",
	}

	// Act
	tracer, err := NewTracer(cfg)

	// Assert
	require.NoError(t, err)
	assert.IsType(t, &otelTracer{}, tracer)
	assert.True(t, tracer.IsEnabled())

	// Cleanup
	_ = tracer.Shutdown(context.Background())
}

// ---------------------------------------------------------------------------
// ShouldTrace tests (otelTracer)
// ---------------------------------------------------------------------------

func TestOtelTracer_ShouldTrace_ShouldReturnTrue_WhenPathNotExcluded(t *testing.T) {
	// Arrange
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

	// Act & Assert
	assert.True(t, tracer.ShouldTrace("/api/v1/users"))
	assert.True(t, tracer.ShouldTrace("/api/orders"))
}

func TestOtelTracer_ShouldTrace_ShouldReturnFalse_WhenPathExcluded(t *testing.T) {
	// Arrange
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

	// Act & Assert
	assert.False(t, tracer.ShouldTrace("/health"))
	assert.False(t, tracer.ShouldTrace("/health/ready"))
	assert.False(t, tracer.ShouldTrace("/metrics"))
}

func TestOtelTracer_ShouldTrace_ShouldReturnTrue_WhenNoExcludePaths(t *testing.T) {
	// Arrange
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

	// Act & Assert
	assert.True(t, tracer.ShouldTrace("/anything"))
}
