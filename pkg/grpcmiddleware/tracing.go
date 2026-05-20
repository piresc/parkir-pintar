package grpcmiddleware

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseFullMethod(fullMethod string) (service, method string) {
	if fullMethod == "" {
		return "", ""
	}

	trimmed := strings.TrimPrefix(fullMethod, "/")

	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return trimmed, ""
}

func (i *Interceptors) TracingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx, endSegment := i.tracer.StartSegment(ctx, info.FullMethod)
		defer endSegment()

		service, method := parseFullMethod(info.FullMethod)

		i.logger.Info("rpc started",
			slog.String("rpc.system", "grpc"),
			slog.String("rpc.service", service),
			slog.String("rpc.method", method),
			slog.String("rpc.full_method", info.FullMethod),
		)

		resp, err := handler(ctx, req)
		if err != nil {
			st, _ := status.FromError(err)
			i.logger.Error("rpc failed",
				slog.String("rpc.system", "grpc"),
				slog.String("rpc.service", service),
				slog.String("rpc.method", method),
				slog.String("rpc.full_method", info.FullMethod),
				slog.String("grpc.code", st.Code().String()),
				slog.String("error", err.Error()),
			)
		}

		return resp, err
	}
}

func (i *Interceptors) TracingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx, endSegment := i.tracer.StartSegment(ss.Context(), info.FullMethod)
		defer endSegment()

		service, method := parseFullMethod(info.FullMethod)

		i.logger.Info("rpc stream started",
			slog.String("rpc.system", "grpc"),
			slog.String("rpc.service", service),
			slog.String("rpc.method", method),
			slog.String("rpc.full_method", info.FullMethod),
		)

		err := handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
		if err != nil {
			st, _ := status.FromError(err)
			code := codes.Unknown
			if st != nil {
				code = st.Code()
			}
			i.logger.Error("rpc stream failed",
				slog.String("rpc.system", "grpc"),
				slog.String("rpc.service", service),
				slog.String("rpc.method", method),
				slog.String("rpc.full_method", info.FullMethod),
				slog.String("grpc.code", code.String()),
				slog.String("error", err.Error()),
			)
		}

		return err
	}
}
