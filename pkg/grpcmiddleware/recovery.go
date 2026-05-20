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

func (i *Interceptors) RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return recovery.UnaryServerInterceptor(
		recovery.WithRecoveryHandlerContext(i.panicHandler()),
	)
}

func (i *Interceptors) panicHandler() recovery.RecoveryHandlerFuncContext {
	return func(ctx context.Context, p any) error {
		i.logger.ErrorContext(ctx, "panic recovered",
			slog.Any("panic", p),
			slog.String("stack", string(debug.Stack())),
		)
		return status.Errorf(codes.Internal, "internal server error")
	}
}
