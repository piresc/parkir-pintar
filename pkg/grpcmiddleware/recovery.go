package grpcmiddleware

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryUnaryInterceptor returns a grpc.UnaryServerInterceptor that recovers
// from panics in the handler chain. On panic, it logs the panic value and stack
// trace at ERROR level and returns a gRPC Internal status code with the message
// "internal server error".
func (i *Interceptors) RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if p := recover(); p != nil {
				i.logger.Error("panic recovered",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())),
				)
				resp = nil
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// RecoveryStreamInterceptor returns a grpc.StreamServerInterceptor that
// recovers from panics in the stream handler chain. On panic, it logs the
// panic value and stack trace at ERROR level and returns a gRPC Internal
// status code with the message "internal server error".
func (i *Interceptors) RecoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if p := recover(); p != nil {
				i.logger.Error("panic recovered",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}
