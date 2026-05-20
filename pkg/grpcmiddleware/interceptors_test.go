
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

func TestRecoveryUnaryInterceptor_ShouldReturnInternal_WhenPanicString(t *testing.T) {
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RecoveryUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		panic("something went wrong")
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal server error", st.Message())
}

func TestRecoveryUnaryInterceptor_ShouldReturnInternal_WhenPanicError(t *testing.T) {
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RecoveryUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		panic(errors.New("error panic"))
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal server error", st.Message())
}

func TestRecoveryUnaryInterceptor_ShouldPassThrough_WhenNoPanic(t *testing.T) {
	interceptors := NewInterceptors("", nil, nil, nil)
	interceptor := interceptors.RecoveryUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestTracingUnaryInterceptor_ShouldStartSegment_WhenCalled(t *testing.T) {
	spy := newSpyTracer()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, spy, nil)
	interceptor := interceptors.TracingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/parking.ReservationService/CreateReservation"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.Equal(t, "/parking.ReservationService/CreateReservation", spy.lastSegmentName())

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.GreaterOrEqual(t, len(lines), 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "grpc", entry["rpc.system"])
	assert.Equal(t, "parking.ReservationService", entry["rpc.service"])
	assert.Equal(t, "CreateReservation", entry["rpc.method"])
}

func TestTracingUnaryInterceptor_ShouldLogError_WhenHandlerFails(t *testing.T) {
	spy := newSpyTracer()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, spy, nil)
	interceptor := interceptors.TracingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/FailMethod"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, status.Errorf(codes.NotFound, "not found")
	}

	_, err := interceptor(context.Background(), nil, info, handler)

	require.Error(t, err)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.GreaterOrEqual(t, len(lines), 2, "expect at least 2 log lines (info + error)")

	var errorEntry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[1], &errorEntry))
	assert.Equal(t, "ERROR", errorEntry["level"])
	assert.Equal(t, "NotFound", errorEntry["grpc.code"])
}

func TestTracingStreamInterceptor_ShouldStartSegment_WhenCalled(t *testing.T) {
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

	err := interceptor(nil, ss, info, handler)

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

func TestRateLimitUnaryInterceptor_ShouldAllow_WhenUnderLimit(t *testing.T) {
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

	for i := 0; i < 3; i++ {
		resp, err := interceptor(ctx, nil, info, handler)

		require.NoError(t, err, "request %d should be allowed", i+1)
		assert.Equal(t, "ok", resp)
	}
}

func TestRateLimitUnaryInterceptor_ShouldReject_WhenOverLimit(t *testing.T) {
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

	for i := 0; i < 2; i++ {
		_, err := interceptor(ctx, nil, info, handler)
		require.NoError(t, err)
	}

	resp, err := interceptor(ctx, nil, info, handler)

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Equal(t, "rate limit exceeded", st.Message())
}

func TestRateLimitUnaryInterceptor_ShouldTrackPerClient(t *testing.T) {
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

	addrA, err := net.ResolveTCPAddr("tcp", "10.0.0.1:1111")
	require.NoError(t, err)
	ctxA := peer.NewContext(context.Background(), &peer.Peer{Addr: addrA})

	_, err = interceptor(ctxA, nil, info, handler)
	require.NoError(t, err)

	_, err = interceptor(ctxA, nil, info, handler)
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.ResourceExhausted, st.Code())

	addrB, err := net.ResolveTCPAddr("tcp", "10.0.0.2:2222")
	require.NoError(t, err)
	ctxB := peer.NewContext(context.Background(), &peer.Peer{Addr: addrB})

	resp, err := interceptor(ctxB, nil, info, handler)

	require.NoError(t, err, "client B should have its own bucket")
	assert.Equal(t, "ok", resp)
}

func TestLoggingUnaryInterceptor_ShouldLogInfo_WhenSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, nil, nil)
	interceptor := interceptors.LoggingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

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
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptors := NewInterceptors("", logger, nil, nil)
	interceptor := interceptors.LoggingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/DeleteUser"}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, status.Errorf(codes.PermissionDenied, "forbidden")
	}

	_, err := interceptor(context.Background(), nil, info, handler)

	require.Error(t, err)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "WARN", entry["level"])
	assert.Equal(t, "DeleteUser", entry["grpc.method"])
	assert.Equal(t, "PermissionDenied", entry["grpc.code"])
}

func TestIdempotencyUnaryInterceptor_ShouldCacheResponse_WhenFirstCall(t *testing.T) {
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

	resp, err := interceptor(ctx, nil, info, handler)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, callCount, "handler must be invoked on first call")
}

func TestIdempotencyUnaryInterceptor_ShouldAllowSecondCall_WhenFirstCompleted(t *testing.T) {
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

	resp1, err := interceptor(ctx, nil, info, handler)
	require.NoError(t, err)
	assert.NotNil(t, resp1)
	assert.Equal(t, 1, callCount)

	resp2, err := interceptor(ctx, nil, info, handler)

	require.NoError(t, err)
	assert.NotNil(t, resp2)
	assert.Equal(t, 2, callCount, "handler should be invoked again after first call completed (deduplication-only mode)")
}

func TestIdempotencyUnaryInterceptor_ShouldReturnInvalidArgument_WhenKeyMissing(t *testing.T) {
	rc, _ := newTestRedisClient(t)
	interceptors := NewInterceptors("", nil, nil, rc)

	method := "/test.Service/CreateOrder"
	cfg := IdempotencyConfig{
		TTL:     10 * time.Second,
		Methods: []string{method},
	}
	interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)
	info := &grpc.UnaryServerInfo{FullMethod: method}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "ok", nil
	}

	resp, err := interceptor(ctx, nil, info, handler)

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "missing idempotency key", st.Message())
}

func TestIdempotencyUnaryInterceptor_ShouldPassThrough_WhenMethodNotEnforced(t *testing.T) {
	rc, _ := newTestRedisClient(t)
	interceptors := NewInterceptors("", nil, nil, rc)

	cfg := IdempotencyConfig{
		TTL:     10 * time.Second,
		Methods: []string{"/test.Service/CreateOrder"},
	}
	interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetOrder"}

	ctx := context.Background()

	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "order-data", nil
	}

	resp, err := interceptor(ctx, nil, info, handler)

	require.NoError(t, err)
	assert.Equal(t, "order-data", resp)
	assert.True(t, handlerCalled, "handler must be called for non-enforced methods")
}
