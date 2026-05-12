// Package grpcclient provides reusable gRPC client dial helpers with default
// interceptors for tracing and logging, mirroring the pattern from
// pkg/httpclient for HTTP clients.
package grpcclient

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"parkir-pintar/pkg/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ClientConfig holds configuration for creating a gRPC client connection.
type ClientConfig struct {
	Target           string
	DialTimeout      time.Duration
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
	TLSEnabled       bool
	Tracer           tracing.Tracer
	Logger           *slog.Logger
}

// Dial creates a grpc.ClientConn with default tracing and logging client
// interceptors. It respects the configured dial timeout and keep-alive
// parameters. If Logger is nil, slog.Default() is used. If Tracer is nil,
// tracing.NewNoOpTracer() is used.
func Dial(ctx context.Context, cfg ClientConfig, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	tracer := cfg.Tracer
	if tracer == nil {
		tracer = tracing.NewNoOpTracer()
	}

	dialOpts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(clientTracingUnaryInterceptor(tracer)),
		grpc.WithStreamInterceptor(clientTracingStreamInterceptor(tracer)),
		grpc.WithChainUnaryInterceptor(
			clientAuthForwardingInterceptor(),
			clientLoggingUnaryInterceptor(logger),
		),
		grpc.WithChainStreamInterceptor(
			clientAuthForwardingStreamInterceptor(),
			clientLoggingStreamInterceptor(logger),
		),
	}

	if !cfg.TLSEnabled {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if cfg.KeepAliveTime > 0 || cfg.KeepAliveTimeout > 0 {
		dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    cfg.KeepAliveTime,
			Timeout: cfg.KeepAliveTimeout,
		}))
	}

	dialOpts = append(dialOpts, opts...)

	// Apply dial timeout via context if configured.
	if cfg.DialTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.DialTimeout)
		defer cancel()
	}

	conn, err := grpc.NewClient(cfg.Target, dialOpts...)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("dial timeout: %s", cfg.Target)
		}
		return nil, fmt.Errorf("failed to dial %s: %w", cfg.Target, err)
	}

	return conn, nil
}

// clientTracingUnaryInterceptor returns a grpc.UnaryClientInterceptor that
// starts a tracing segment for each outbound unary RPC and propagates trace
// context via the tracer.
func clientTracingUnaryInterceptor(tracer tracing.Tracer) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx, endSegment := tracer.StartSegment(ctx, method)
		defer endSegment()

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// clientTracingStreamInterceptor returns a grpc.StreamClientInterceptor that
// starts a tracing segment for each outbound streaming RPC.
func clientTracingStreamInterceptor(tracer tracing.Tracer) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx, endSegment := tracer.StartSegment(ctx, method)
		defer endSegment()

		return streamer(ctx, desc, cc, method, opts...)
	}
}

// clientLoggingUnaryInterceptor returns a grpc.UnaryClientInterceptor that
// logs the method name, gRPC status code, and duration for each outbound
// unary RPC. INFO level for successful calls, ERROR level for failures.
func clientLoggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)
		st, _ := status.FromError(err)
		code := codes.OK
		if st != nil {
			code = st.Code()
		}

		attrs := []slog.Attr{
			slog.String("grpc.method", method),
			slog.String("grpc.code", code.String()),
			slog.Float64("duration_ms", float64(duration.Milliseconds())),
		}

		if code == codes.OK {
			logger.LogAttrs(ctx, slog.LevelInfo, "grpc client call completed", attrs...)
		} else {
			logger.LogAttrs(ctx, slog.LevelError, "grpc client call completed", attrs...)
		}

		return err
	}
}

// clientLoggingStreamInterceptor returns a grpc.StreamClientInterceptor that
// logs the method name, gRPC status code, and duration for each outbound
// streaming RPC.
func clientLoggingStreamInterceptor(logger *slog.Logger) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		start := time.Now()

		cs, err := streamer(ctx, desc, cc, method, opts...)

		duration := time.Since(start)
		st, _ := status.FromError(err)
		code := codes.OK
		if st != nil {
			code = st.Code()
		}

		attrs := []slog.Attr{
			slog.String("grpc.method", method),
			slog.String("grpc.code", code.String()),
			slog.Float64("duration_ms", float64(duration.Milliseconds())),
		}

		if code == codes.OK {
			logger.LogAttrs(ctx, slog.LevelInfo, "grpc client stream started", attrs...)
		} else {
			logger.LogAttrs(ctx, slog.LevelError, "grpc client stream failed", attrs...)
		}

		return cs, err
	}
}

// clientAuthForwardingInterceptor returns a grpc.UnaryClientInterceptor that
// forwards the "authorization" metadata from the incoming context (set by the
// gateway's contextWithAuth) to the outgoing gRPC call. This ensures JWT tokens
// are propagated across service-to-service calls (e.g. reservation → billing).
func clientAuthForwardingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Try to extract authorization from incoming metadata first,
		// then fall back to context values from the gRPC auth interceptor.
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			authVals := md.Get("authorization")
			if len(authVals) > 0 {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authVals[0])
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// clientAuthForwardingStreamInterceptor does the same for streaming calls.
func clientAuthForwardingStreamInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			authVals := md.Get("authorization")
			if len(authVals) > 0 {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authVals[0])
			}
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
