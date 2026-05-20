// - Never ignore errors
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

func NewLogger(cfg config.LoggerConfig) *slog.Logger {
	return NewLoggerWithProvider(cfg, nil)
}

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
		baseHandler = slog.NewJSONHandler(os.Stdout, opts)
	}

	handler := &otelHandler{base: baseHandler}

	if lp != nil {
		otelBridge := otelslog.NewHandler("parkir-pintar",
			otelslog.WithLoggerProvider(lp),
		)
		fanout := &fanoutHandler{handlers: []slog.Handler{handler, otelBridge}}
		return slog.New(fanout)
	}

	return slog.New(handler)
}

type fanoutHandler struct {
	handlers []slog.Handler
}

func (f *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

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

func (f *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: handlers}
}

func (f *fanoutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &fanoutHandler{handlers: handlers}
}

const (
	levelDebug = "debug"
	levelInfo  = "info"
)

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

type otelHandler struct {
	base slog.Handler
}

func (h *otelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

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

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{base: h.base.WithAttrs(attrs)}
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{base: h.base.WithGroup(name)}
}

func String(key, val string) slog.Attr {
	return slog.String(key, val)
}

func Int(key string, val int) slog.Attr {
	return slog.Int(key, val)
}

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

func Any(key string, val any) slog.Attr {
	return slog.Any(key, val)
}

func Float64(key string, val float64) slog.Attr {
	return slog.Float64(key, val)
}

func Duration(key string, val time.Duration) slog.Attr {
	return slog.Duration(key, val)
}
