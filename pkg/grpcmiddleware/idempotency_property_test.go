// Bug Condition: concurrentSameKey
package grpcmiddleware

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"parkir-pintar/pkg/config"
	pkgredis "parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/tracing"
)

func newTestRedisClient(t *testing.T) (*pkgredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	rc, err := pkgredis.NewClient(config.RedisConfig{
		Host:     host,
		Port:     port,
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = rc.Close()
		mr.Close()
	})

	return rc, mr
}

// TestIdempotencyInterceptor_ShouldExecuteHandlerOnce_WhenConcurrentSameKey
// launches 2 concurrent gRPC requests with the same idempotency key and verifies
func TestIdempotencyInterceptor_ShouldExecuteHandlerOnce_WhenConcurrentSameKey(t *testing.T) {
	rc, _ := newTestRedisClient(t)

	interceptors := NewInterceptors("test-secret", slog.Default(), tracing.NewNoOpTracer(), rc)

	cfg := IdempotencyConfig{
		TTL:     30 * time.Second,
		Methods: []string{"/test.Service/CreateOrder"},
	}

	interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)

	var handlerExecutions atomic.Int64

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerExecutions.Add(1)
		time.Sleep(10 * time.Millisecond)
		return map[string]string{"result": "ok"}, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/CreateOrder",
	}

	// Act — launch 2 concurrent requests with the same idempotency key
	var wg sync.WaitGroup
	wg.Add(2)

	for range 2 {
		go func() {
			defer wg.Done()
			ctx := metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("x-idempotency-key", "same-key-123"),
			)
			_, _ = interceptor(ctx, "test-request", info, handler)
		}()
	}

	wg.Wait()

	executions := handlerExecutions.Load()
	assert.Equal(t, int64(1), executions,
		"handler should execute exactly 1 time with same idempotency key, but executed %d times — GET-then-SET race detected",
		executions)
}
