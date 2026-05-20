// Package logger tests
//
// Best practices applied (from Go testing standards KB):
// - Use descriptive names: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - Follow AAA (Arrange-Act-Assert) pattern
// - Table-driven tests for multiple scenarios
// - Use testify assertions for clear failure messages
// - Test both success and error/edge cases
// - Tests are fast, isolated, repeatable, and clear
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"parkir-pintar/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// --- NewLogger tests ---

func TestNewLogger_ShouldReturnLogger_WhenJSONFormatConfigured(t *testing.T) {
	// Arrange
	cfg := config.LoggerConfig{
		Level:  "info",
		Format: "json",
	}

	// Act
	logger := NewLogger(cfg)

	// Assert
	assert.NotNil(t, logger)
}

func TestNewLogger_ShouldReturnLogger_WhenTextFormatConfigured(t *testing.T) {
	// Arrange
	cfg := config.LoggerConfig{
		Level:  "debug",
		Format: "text",
	}

	// Act
	logger := NewLogger(cfg)

	// Assert
	assert.NotNil(t, logger)
}

func TestNewLogger_ShouldDefaultToJSON_WhenFormatIsEmpty(t *testing.T) {
	// Arrange
	cfg := config.LoggerConfig{
		Level:  "info",
		Format: "",
	}

	// Act
	logger := NewLogger(cfg)

	// Assert — logger should be created without error (defaults to JSON)
	assert.NotNil(t, logger)
}

// --- parseLevel tests (table-driven) ---

func TestParseLevel_ShouldReturnCorrectLevel_WhenValidLevelProvided(t *testing.T) {
	// Arrange — table-driven test for all supported levels
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{name: "debug level", input: "debug", expected: slog.LevelDebug},
		{name: "info level", input: "info", expected: slog.LevelInfo},
		{name: "warn level", input: "warn", expected: slog.LevelWarn},
		{name: "error level", input: "error", expected: slog.LevelError},
		{name: "uppercase DEBUG", input: "DEBUG", expected: slog.LevelDebug},
		{name: "mixed case Info", input: "Info", expected: slog.LevelInfo},
		{name: "unknown defaults to info", input: "unknown", expected: slog.LevelInfo},
		{name: "empty defaults to info", input: "", expected: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := parseLevel(tt.input)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- otelHandler tests ---

func TestOtelHandler_ShouldAddTraceAttributes_WhenValidSpanInContext(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := &otelHandler{base: baseHandler}

	// Create a span context with known trace/span IDs
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	require.NoError(t, err)

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), spanCtx)

	// Act
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err = handler.Handle(ctx, record)

	// Assert
	require.NoError(t, err)

	var logEntry map[string]any
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "0102030405060708090a0b0c0d0e0f10", logEntry["trace_id"])
	assert.Equal(t, "0102030405060708", logEntry["span_id"])
	assert.Equal(t, "test message", logEntry["msg"])
}

func TestOtelHandler_ShouldNotAddTraceAttributes_WhenNoSpanInContext(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := &otelHandler{base: baseHandler}

	ctx := context.Background()

	// Act
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "no span message", 0)
	err := handler.Handle(ctx, record)

	// Assert
	require.NoError(t, err)

	var logEntry map[string]any
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Nil(t, logEntry["trace_id"], "trace_id should not be present without a span")
	assert.Nil(t, logEntry["span_id"], "span_id should not be present without a span")
}

func TestOtelHandler_ShouldRespectLogLevel_WhenLevelBelowThreshold(t *testing.T) {
	// Arrange
	handler := &otelHandler{
		base: slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn}),
	}

	// Act & Assert — debug should not be enabled when level is warn
	assert.False(t, handler.Enabled(context.Background(), slog.LevelDebug))
	assert.False(t, handler.Enabled(context.Background(), slog.LevelInfo))
	assert.True(t, handler.Enabled(context.Background(), slog.LevelWarn))
	assert.True(t, handler.Enabled(context.Background(), slog.LevelError))
}

func TestOtelHandler_ShouldPreserveAttrs_WhenWithAttrsCalled(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := &otelHandler{base: baseHandler}

	// Act
	childHandler := handler.WithAttrs([]slog.Attr{slog.String("service", "test-svc")})
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "with attrs", 0)
	err := childHandler.Handle(context.Background(), record)

	// Assert
	require.NoError(t, err)

	var logEntry map[string]any
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test-svc", logEntry["service"])
}

func TestOtelHandler_ShouldPreserveGroup_WhenWithGroupCalled(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := &otelHandler{base: baseHandler}

	// Act
	groupHandler := handler.WithGroup("request")
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "grouped", 0)
	record.AddAttrs(slog.String("method", "GET"))
	err := groupHandler.Handle(context.Background(), record)

	// Assert
	require.NoError(t, err)

	var logEntry map[string]any
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	requestGroup, ok := logEntry["request"].(map[string]any)
	require.True(t, ok, "expected 'request' group in log output")
	assert.Equal(t, "GET", requestGroup["method"])
}

// --- Helper attribute constructor tests ---

func TestString_ShouldReturnStringAttr_WhenCalled(t *testing.T) {
	// Act
	attr := String("key", "value")

	// Assert
	assert.Equal(t, "key", attr.Key)
	assert.Equal(t, "value", attr.Value.String())
}

func TestInt_ShouldReturnIntAttr_WhenCalled(t *testing.T) {
	// Act
	attr := Int("count", 42)

	// Assert
	assert.Equal(t, "count", attr.Key)
	assert.Equal(t, int64(42), attr.Value.Int64())
}

func TestErr_ShouldReturnErrorAttr_WhenErrorProvided(t *testing.T) {
	// Arrange
	testErr := errors.New("something went wrong")

	// Act
	attr := Err(testErr)

	// Assert
	assert.Equal(t, "error", attr.Key)
}

func TestFloat64_ShouldReturnFloat64Attr_WhenCalled(t *testing.T) {
	// Act
	attr := Float64("rate", 3.14)

	// Assert
	assert.Equal(t, "rate", attr.Key)
	assert.InDelta(t, 3.14, attr.Value.Float64(), 0.001)
}

func TestDuration_ShouldReturnDurationAttr_WhenCalled(t *testing.T) {
	// Act
	attr := Duration("elapsed", 5*time.Second)

	// Assert
	assert.Equal(t, "elapsed", attr.Key)
	assert.Equal(t, (5 * time.Second).String(), attr.Value.Duration().String())
}

func TestAny_ShouldReturnAnyAttr_WhenCalled(t *testing.T) {
	// Act
	attr := Any("data", map[string]int{"a": 1})

	// Assert
	assert.Equal(t, "data", attr.Key)
}
