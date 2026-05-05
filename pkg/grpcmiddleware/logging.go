package grpcmiddleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingUnaryInterceptor returns a grpc.UnaryServerInterceptor that logs the
// full method name, gRPC status code, and duration in milliseconds for each
// unary RPC. Successful calls (codes.OK) are logged at INFO level; all other
// status codes are logged at ERROR level.
func (i *Interceptors) LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		st, _ := status.FromError(err)
		code := codes.OK
		if st != nil {
			code = st.Code()
		}

		attrs := []slog.Attr{
			slog.String("grpc.method", info.FullMethod),
			slog.String("grpc.code", code.String()),
			slog.Float64("duration_ms", float64(duration.Milliseconds())),
		}

		if code == codes.OK {
			i.logger.LogAttrs(ctx, slog.LevelInfo, "grpc call completed", attrs...)
		} else {
			i.logger.LogAttrs(ctx, slog.LevelError, "grpc call completed", attrs...)
		}

		return resp, err
	}
}

// LoggingStreamInterceptor returns a grpc.StreamServerInterceptor that logs
// the full method name, gRPC status code, and duration in milliseconds for
// each streaming RPC. Successful calls (codes.OK) are logged at INFO level;
// all other status codes are logged at ERROR level.
func (i *Interceptors) LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)
		st, _ := status.FromError(err)
		code := codes.OK
		if st != nil {
			code = st.Code()
		}

		attrs := []slog.Attr{
			slog.String("grpc.method", info.FullMethod),
			slog.String("grpc.code", code.String()),
			slog.Float64("duration_ms", float64(duration.Milliseconds())),
		}

		if code == codes.OK {
			i.logger.LogAttrs(ss.Context(), slog.LevelInfo, "grpc call completed", attrs...)
		} else {
			i.logger.LogAttrs(ss.Context(), slog.LevelError, "grpc call completed", attrs...)
		}

		return err
	}
}
