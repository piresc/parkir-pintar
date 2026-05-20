// Best practices applied from Go testing guidelines:
// - Descriptive test names using ShouldXXX_WhenYYY pattern
// - AAA (Arrange-Act-Assert) structure
// - testify assertions (assert, require)
// - Direct interceptor invocation with mock handlers (no bufconn needed)
// - bytes.Buffer + slog.NewJSONHandler for capturing log output
// - miniredis for Redis-backed interceptors (idempotency)
// - spyTracer pattern from tracing_property_test.go for tracing tests
// - mockServerStream from auth_test.go reused (same package)
// - newTestRedisClient from idempotency_property_test.go reused (same package)

package grpcmiddleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// ---------------------------------------------------------------------------
// Recovery interceptor tests
// ---------------------------------------------------------------------------

func TestRecoveryUnaryInterceptor_ShouldReturnInternal_WhenPanicString(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RecoveryUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		panic("something went wrong")
	}

	// Act
	resp, err := interceptor(context.Background(), nil, info, handler)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal server error", st.Message())
}

func TestRecoveryUnaryInterceptor_ShouldReturnInternal_WhenPanicError(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RecoveryUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		panic(errors.New("error panic"))
	}

	// Act
	resp, err := interceptor(context.Background(), nil, info, handler)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal server error", st.Message())
}

func TestRecoveryUnaryInterceptor_ShouldPassThrough_WhenNoPanic(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RecoveryUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	// Act
	resp, err := interceptor(context.Background(), nil, info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

// ---------------------------------------------------------------------------
// Tracing interceptor tests
// ---------------------------------------------------------------------------

func TestTracingUnaryInterceptor_ShouldStartSegment_WhenCalled(t *testing.T) {
	// Arrange
	spy := newSpyTracer()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, spy, nil)
	interceptor := interceptors.TracingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/parking.ReservationService/CreateReservation"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	// Act
	resp, err := interceptor(context.Background(), nil, info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.Equal(t, "/parking.ReservationService/CreateReservation", spy.lastSegmentName())

	// Verify structured log fields.
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.GreaterOrEqual(t, len(lines), 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "grpc", entry["rpc.system"])
	assert.Equal(t, "parking.ReservationService", entry["rpc.service"])
	assert.Equal(t, "CreateReservation", entry["rpc.method"])
}

func TestTracingUnaryInterceptor_ShouldLogError_WhenHandlerFails(t *testing.T) {
	// Arrange
	spy := newSpyTracer()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, spy, nil)
	interceptor := interceptors.TracingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/FailMethod"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, status.Errorf(codes.NotFound, "not found")
	}

	// Act
	_, err := interceptor(context.Background(), nil, info, handler)

	// Assert
	require.Error(t, err)

	// Find the ERROR log line.
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.GreaterOrEqual(t, len(lines), 2, "expect at least 2 log lines (info + error)")

	var errorEntry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[1], &errorEntry))
	assert.Equal(t, "ERROR", errorEntry["level"])
	assert.Equal(t, "NotFound", errorEntry["grpc.code"])
}

func TestTracingStreamInterceptor_ShouldStartSegment_WhenCalled(t *testing.T) {
	// Arrange
	spy := newSpyTracer()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, spy, nil)
	interceptor := interceptors.TracingStreamInterceptor()
	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"}
	ss := &mockServerStream{ctx: context.Background()}

	handler := func(_ interface{}, _ grpc.ServerStream) error {
		return nil
	}

	// Act
	err := interceptor(nil, ss, info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "/test.Service/StreamMethod", spy.lastSegmentName())

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.GreaterOrEqual(t, len(lines), 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "grpc", entry["rpc.system"])
	assert.Equal(t, "test.Service", entry["rpc.service"])
	assert.Equal(t, "StreamMethod", entry["rpc.method"])
}

// ---------------------------------------------------------------------------
// Rate limit interceptor tests
// ---------------------------------------------------------------------------

func TestRateLimitUnaryInterceptor_ShouldAllow_WhenUnderLimit(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RateLimitUnaryInterceptor(RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   0,
	})
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	addr, err := net.ResolveTCPAddr("tcp", "192.168.1.1:5000")
	require.NoError(t, err)
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	// Act — send 3 requests (under burst of 5)
	for i := 0; i < 3; i++ {
		resp, err := interceptor(ctx, nil, info, handler)

		// Assert
		require.NoError(t, err, "request %d should be allowed", i+1)
		assert.Equal(t, "ok", resp)
	}
}

func TestRateLimitUnaryInterceptor_ShouldReject_WhenOverLimit(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RateLimitUnaryInterceptor(RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         2,
		CleanupInterval:   0,
	})
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	addr, err := net.ResolveTCPAddr("tcp", "10.0.0.1:8080")
	require.NoError(t, err)
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	// Exhaust the burst.
	for i := 0; i < 2; i++ {
		_, err := interceptor(ctx, nil, info, handler)
		require.NoError(t, err)
	}

	// Act — next request should be rejected.
	resp, err := interceptor(ctx, nil, info, handler)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Equal(t, "rate limit exceeded", st.Message())
}

func TestRateLimitUnaryInterceptor_ShouldTrackPerClient(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RateLimitUnaryInterceptor(RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         1,
		CleanupInterval:   0,
	})
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	// Client A — exhaust its single token.
	addrA, err := net.ResolveTCPAddr("tcp", "10.0.0.1:1111")
	require.NoError(t, err)
	ctxA := peer.NewContext(context.Background(), &peer.Peer{Addr: addrA})

	_, err = interceptor(ctxA, nil, info, handler)
	require.NoError(t, err)

	// Client A should now be rate limited.
	_, err = interceptor(ctxA, nil, info, handler)
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.ResourceExhausted, st.Code())

	// Act — Client B should still be allowed (separate bucket).
	addrB, err := net.ResolveTCPAddr("tcp", "10.0.0.2:2222")
	require.NoError(t, err)
	ctxB := peer.NewContext(context.Background(), &peer.Peer{Addr: addrB})

	resp, err := interceptor(ctxB, nil, info, handler)

	// Assert
	require.NoError(t, err, "client B should have its own bucket")
	assert.Equal(t, "ok", resp)
}

// ---------------------------------------------------------------------------
// Logging interceptor tests
// ---------------------------------------------------------------------------

func TestLoggingUnaryInterceptor_ShouldLogInfo_WhenSuccess(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, nil, nil)
	interceptor := interceptors.LoggingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	// Act
	resp, err := interceptor(context.Background(), nil, info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "GetUser", entry["grpc.method"])
	assert.Equal(t, "OK", entry["grpc.code"])

	durationVal, ok := entry["duration_ms"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, durationVal, float64(0))
}

func TestLoggingUnaryInterceptor_ShouldLogError_WhenFailure(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, nil, nil)
	interceptor := interceptors.LoggingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/DeleteUser"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, status.Errorf(codes.PermissionDenied, "forbidden")
	}

	// Act
	_, err := interceptor(context.Background(), nil, info, handler)

	// Assert
	require.Error(t, err)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "WARN", entry["level"])
	assert.Equal(t, "DeleteUser", entry["grpc.method"])
	assert.Equal(t, "PermissionDenied", entry["grpc.code"])
}

// ---------------------------------------------------------------------------
// Idempotency interceptor tests
// ---------------------------------------------------------------------------

func TestIdempotencyUnaryInterceptor_ShouldCacheResponse_WhenFirstCall(t *testing.T) {
	// Arrange
	rc, _ := newTestRedisClient(t)
	interceptors := NewInterceptors("", nil, nil, rc)

	method := "/test.Service/CreateOrder"
	cfg := IdempotencyConfig{
		TTL:     10 * time.Second,
		Methods: []string{method},
	}
	interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)
	info := &grpc.UnaryServerInfo{FullMethod: method}

	md := metadata.Pairs("x-idempotency-key", "key-abc-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	callCount := 0
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		callCount++
		return map[string]interface{}{"order_id": "ord-1"}, nil
	}

	// Act
	resp, err := interceptor(ctx, nil, info, handler)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, callCount, "handler must be invoked on first call")
}

func TestIdempotencyUnaryInterceptor_ShouldAllowSecondCall_WhenFirstCompleted(t *testing.T) {
	// Arrange
	rc, _ := newTestRedisClient(t)
	interceptors := NewInterceptors("", nil, nil, rc)

	method := "/test.Service/CreateOrder"
	cfg := IdempotencyConfig{
		TTL:     10 * time.Second,
		Methods: []string{method},
	}
	interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)
	info := &grpc.UnaryServerInfo{FullMethod: method}

	md := metadata.Pairs("x-idempotency-key", "key-xyz-789")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	callCount := 0
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		callCount++
		return map[string]interface{}{"order_id": "ord-2"}, nil
	}

	// First call — acquires and releases the deduplication key.
	resp1, err := interceptor(ctx, nil, info, handler)
	require.NoError(t, err)
	assert.NotNil(t, resp1)
	assert.Equal(t, 1, callCount)

	// Act — second call with same key after first completed (key released).
	// The deduplication-only approach allows retries after completion;
	// the handler's own idempotency (DB constraints) handles deduplication.
	resp2, err := interceptor(ctx, nil, info, handler)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp2)
	assert.Equal(t, 2, callCount, "handler should be invoked again after first call completed (deduplication-only mode)")
}

func TestIdempotencyUnaryInterceptor_ShouldReturnInvalidArgument_WhenKeyMissing(t *testing.T) {
	// Arrange
	rc, _ := newTestRedisClient(t)
	interceptors := NewInterceptors("", nil, nil, rc)

	method := "/test.Service/CreateOrder"
	cfg := IdempotencyConfig{
		TTL:     10 * time.Second,
		Methods: []string{method},
	}
	interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)
	info := &grpc.UnaryServerInfo{FullMethod: method}

	// Context without idempotency key metadata.
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	// Act
	resp, err := interceptor(ctx, nil, info, handler)

	// Assert
	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "missing idempotency key", st.Message())
}

func TestIdempotencyUnaryInterceptor_ShouldPassThrough_WhenMethodNotEnforced(t *testing.T) {
	// Arrange
	rc, _ := newTestRedisClient(t)
	interceptors := NewInterceptors("", nil, nil, rc)

	cfg := IdempotencyConfig{
		TTL:     10 * time.Second,
		Methods: []string{"/test.Service/CreateOrder"},
	}
	interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)

	// Use a method NOT in the enforcement list.
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetOrder"}

	// No idempotency key needed for non-enforced methods.
	ctx := context.Background()

	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "order-data", nil
	}

	// Act
	resp, err := interceptor(ctx, nil, info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "order-data", resp)
	assert.True(t, handlerCalled, "handler must be called for non-enforced methods")
}
