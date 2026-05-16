package grpcmiddleware

import (
	"context"
	"net"

	"parkir-pintar/pkg/ratelimit"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimitConfig is an alias for the shared ratelimit.Config.
type RateLimitConfig = ratelimit.Config

// RateLimitUnaryInterceptor returns a grpc.UnaryServerInterceptor that
// enforces per-client rate limiting using golang.org/x/time/rate (token bucket).
// The client is identified by its peer address from the gRPC transport.
// Requests exceeding the limit receive a gRPC ResourceExhausted status code.
func (i *Interceptors) RateLimitUnaryInterceptor(cfg RateLimitConfig) grpc.UnaryServerInterceptor {
	store := ratelimit.NewStore(cfg)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		key := "unknown"
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			host, _, err := net.SplitHostPort(p.Addr.String())
			if err != nil {
				key = p.Addr.String()
			} else {
				key = host
			}
		}

		if !store.Allow(key) {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
