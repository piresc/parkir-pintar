package grpcmiddleware

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryUnaryInterceptor returns a grpc.UnaryServerInterceptor that recovers
// from panics in the handler chain. On panic, it logs the panic value and stack
// trace at ERROR level and returns a gRPC Internal status code with the message
// "internal server error".
//
// Powered by github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery.
func (i *Interceptors) RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return recovery.UnaryServerInterceptor(
		recovery.WithRecoveryHandlerContext(i.panicHandler()),
	)
}

// panicHandler returns a recovery handler that logs the panic and returns Internal.
func (i *Interceptors) panicHandler() recovery.RecoveryHandlerFuncContext {
	return func(ctx context.Context, p any) error {
		i.logger.ErrorContext(ctx, "panic recovered",
			slog.Any("panic", p),
			slog.String("stack", string(debug.Stack())),
		)
		return status.Errorf(codes.Internal, "internal server error")
	}
}
