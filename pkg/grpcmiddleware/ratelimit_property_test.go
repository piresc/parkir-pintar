package grpcmiddleware

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"pgregory.net/rapid"
)

// Feature: grpc-jwt-pkg-integration, Property 6: Token bucket rate limiting
// **Validates: Requirements 5.2, 5.3**
//
// For any RequestsPerSecond R and BurstSize B, a fresh rate limiter SHALL
// allow exactly B requests immediately, and after consuming all tokens, the
// next request SHALL be rejected with ResourceExhausted.
func TestProperty6_TokenBucketRateLimiting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		burstSize := rapid.IntRange(1, 50).Draw(t, "burstSize")
		rps := rapid.IntRange(1, 100).Draw(t, "requestsPerSecond")

		interceptors := NewInterceptors("", nil, nil, nil)
		interceptor := interceptors.RateLimitUnaryInterceptor(RateLimitConfig{
			RequestsPerSecond: rps,
			BurstSize:         burstSize,
			CleanupInterval:   0, // disable cleanup goroutine for tests
		})

		info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

		// Create a context with a fake peer address.
		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
		require.NoError(t, err)
		ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})

		successHandler := func(_ context.Context, _ interface{}) (interface{}, error) {
			return "ok", nil
		}

		// A fresh bucket starts with burstSize tokens. Exactly burstSize
		// requests should be allowed immediately.
		for i := 0; i < burstSize; i++ {
			resp, err := interceptor(ctx, nil, info, successHandler)
			assert.NoError(t, err, "request %d of %d should be allowed", i+1, burstSize)
			assert.Equal(t, "ok", resp)
		}

		// The next request should be rejected because all tokens are consumed.
		resp, err := interceptor(ctx, nil, info, successHandler)
		assert.Nil(t, resp, "response must be nil when rate limited")
		require.Error(t, err, "request after burst exhaustion must be rejected")

		st, ok := status.FromError(err)
		require.True(t, ok, "error must be a gRPC status")
		assert.Equal(t, codes.ResourceExhausted, st.Code(), "must return ResourceExhausted")
		assert.Equal(t, "rate limit exceeded", st.Message(), "must return correct message")
	})
}
