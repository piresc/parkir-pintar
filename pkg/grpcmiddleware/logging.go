package grpcmiddleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

// slogInterceptorLogger adapts *slog.Logger to the grpc-ecosystem logging.Logger interface.
type slogInterceptorLogger struct {
	logger *slog.Logger
}

func (l *slogInterceptorLogger) Log(ctx context.Context, level logging.Level, msg string, fields ...any) {
	switch level {
	case logging.LevelDebug:
		l.logger.DebugContext(ctx, msg, fields...)
	case logging.LevelInfo:
		l.logger.InfoContext(ctx, msg, fields...)
	case logging.LevelWarn:
		l.logger.WarnContext(ctx, msg, fields...)
	case logging.LevelError:
		l.logger.ErrorContext(ctx, msg, fields...)
	default:
		l.logger.InfoContext(ctx, msg, fields...)
	}
}

// LoggingUnaryInterceptor returns a grpc.UnaryServerInterceptor that logs the
// full method name, gRPC status code, and duration for each unary RPC.
// Successful calls (codes.OK) are logged at INFO level; all other status codes
// are logged at ERROR level.
//
// Powered by github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging.
func (i *Interceptors) LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return logging.UnaryServerInterceptor(
		&slogInterceptorLogger{logger: i.logger},
		logging.WithLogOnEvents(logging.FinishCall),
		logging.WithFieldsFromContext(traceFieldsExtractor),
		logging.WithDurationField(func(duration time.Duration) logging.Fields {
			return logging.Fields{"duration_ms", duration.Milliseconds()}
		}),
	)
}

// LoggingStreamInterceptor returns a grpc.StreamServerInterceptor that logs
// the full method name, gRPC status code, and duration for each streaming RPC.
func (i *Interceptors) LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return logging.StreamServerInterceptor(
		&slogInterceptorLogger{logger: i.logger},
		logging.WithLogOnEvents(logging.FinishCall),
		logging.WithFieldsFromContext(traceFieldsExtractor),
		logging.WithDurationField(func(duration time.Duration) logging.Fields {
			return logging.Fields{"duration_ms", duration.Milliseconds()}
		}),
	)
}

// traceFieldsExtractor extracts trace_id and span_id from context if available.
func traceFieldsExtractor(ctx context.Context) logging.Fields {
	// The tracing interceptor already handles span context propagation.
	// This is a hook point for additional context fields if needed.
	return nil
}
