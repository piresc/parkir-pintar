// Package grpcmiddleware provides bug condition exploration tests for gRPC idempotency atomicity.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - Each test is isolated with its own mock setup
// - Mock at interface boundaries rather than concrete implementations
//
// **Validates: Requirements 2.11** (Property 8 from design)
//
// Bug Condition: concurrentSameKey
// Expected: exactly 1 handler execution
// Counterexample on unfixed code: handler executed twice (GET-then-SET race)
//
// CRITICAL: This test is expected to FAIL on unfixed code.
// DO NOT fix the code or the test when it fails.
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

// newTestRedisClient creates a miniredis server and a RedisClient connected to it.
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
// only 1 handler execution occurs. On unfixed code, both requests get a Redis
// cache miss and both execute the handler.
//
// **Validates: Requirements 2.11**
func TestIdempotencyInterceptor_ShouldExecuteHandlerOnce_WhenConcurrentSameKey(t *testing.T) {
	// Arrange — set up miniredis for real Redis operations
	rc, _ := newTestRedisClient(t)

	interceptors := NewInterceptors("test-secret", slog.Default(), tracing.NewNoOpTracer(), rc)

	cfg := IdempotencyConfig{
		TTL:     30 * time.Second,
		Methods: []string{"/test.Service/CreateOrder"},
	}

	interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)

	// Track handler executions
	var handlerExecutions atomic.Int64

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerExecutions.Add(1)
		// Simulate some work to widen the race window
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

	// Assert — handler should execute exactly once
	executions := handlerExecutions.Load()
	assert.Equal(t, int64(1), executions,
		"handler should execute exactly 1 time with same idempotency key, but executed %d times — GET-then-SET race detected",
		executions)
}
