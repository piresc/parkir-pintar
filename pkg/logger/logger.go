// Package logger provides structured slog-based logging with native OTEL
// trace/span context correlation and optional OTLP log export via the
// official OTel slog bridge.
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Use context.Context as first parameter for consistency
// - Return errors as the last value from functions
// - Never ignore errors
// - Use keyed fields in struct literals
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"parkir-pintar/pkg/config"

	"go.opentelemetry.io/otel/trace"

	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// NewLogger creates a configured *slog.Logger based on the provided LoggerConfig.
// It supports JSON (default) and text output formats, configurable log level,
// source file/line info, and automatic OTEL trace_id/span_id extraction from context.
// If a LoggerProvider is supplied, logs are also exported via OTLP.
func NewLogger(cfg config.LoggerConfig) *slog.Logger {
	return NewLoggerWithProvider(cfg, nil)
}

// NewLoggerWithProvider creates a configured *slog.Logger that writes to stdout
// and optionally exports logs via the given OTel LoggerProvider.
func NewLoggerWithProvider(cfg config.LoggerConfig, lp *sdklog.LoggerProvider) *slog.Logger {
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	var baseHandler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "text":
		baseHandler = slog.NewTextHandler(os.Stdout, opts)
	default:
		// JSON is the default format for structured logging in production.
		baseHandler = slog.NewJSONHandler(os.Stdout, opts)
	}

	// Wrap with OTel trace context injection.
	handler := &otelHandler{base: baseHandler}

	if lp != nil {
		// Create an OTel slog bridge that sends logs via OTLP.
		otelBridge := otelslog.NewHandler("parkir-pintar",
			otelslog.WithLoggerProvider(lp),
		)
		// Fan-out handler: write to both stdout (with trace context) and OTLP.
		fanout := &fanoutHandler{handlers: []slog.Handler{handler, otelBridge}}
		return slog.New(fanout)
	}

	return slog.New(handler)
}

// fanoutHandler sends log records to multiple slog.Handlers.
type fanoutHandler struct {
	handlers []slog.Handler
}

// Enabled reports whether any handler handles records at the given level.
func (f *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle sends the record to all handlers.
func (f *fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range f.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs returns a new fanout handler with the given attributes pre-applied.
func (f *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: handlers}
}

// WithGroup returns a new fanout handler with the given group name.
func (f *fanoutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &fanoutHandler{handlers: handlers}
}

// Log level string constants.
const (
	levelDebug = "debug"
	levelInfo  = "info"
)

// parseLevel converts a string log level to slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case levelDebug:
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case levelInfo:
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

// otelHandler is a custom slog.Handler that wraps a base handler and
// automatically extracts OTEL trace_id and span_id from context via
// trace.SpanFromContext. When the span context is valid, these attributes
// are added to every log record transparently.
type otelHandler struct {
	base slog.Handler
}

// Enabled reports whether the handler handles records at the given level.
func (h *otelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle adds OTEL trace_id and span_id attributes when a valid span is
// present in the context, then delegates to the base handler.
func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			r.AddAttrs(
				slog.String("trace_id", span.SpanContext().TraceID().String()),
				slog.String("span_id", span.SpanContext().SpanID().String()),
			)
		}
	}
	return h.base.Handle(ctx, r)
}

// WithAttrs returns a new handler with the given attributes pre-applied.
func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{base: h.base.WithAttrs(attrs)}
}

// WithGroup returns a new handler with the given group name.
func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{base: h.base.WithGroup(name)}
}

// --- Exported helper attribute constructors ---

// String creates a string slog.Attr.
func String(key, val string) slog.Attr {
	return slog.String(key, val)
}

// Int creates an integer slog.Attr.
func Int(key string, val int) slog.Attr {
	return slog.Int(key, val)
}

// Err creates an slog.Attr for an error value with key "error".
func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

// Any creates an slog.Attr for an arbitrary value.
func Any(key string, val any) slog.Attr {
	return slog.Any(key, val)
}

// Float64 creates a float64 slog.Attr.
func Float64(key string, val float64) slog.Attr {
	return slog.Float64(key, val)
}

// Duration creates an slog.Attr for a time.Duration value.
func Duration(key string, val time.Duration) slog.Attr {
	return slog.Duration(key, val)
}
