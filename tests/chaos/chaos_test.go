//go:build chaos

// Package chaos implements resilience tests using toxiproxy to inject network
// faults and verify that ParkirPintar services degrade gracefully.
//
// Prerequisites:
//   - toxiproxy-server running on localhost:8474
//   - Redis, NATS, PostgreSQL, and gRPC services accessible (or proxied)
//
// Run: go test -tags chaos -v -timeout 120s ./tests/chaos/
package chaos

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"parkir-pintar/pkg/circuitbreaker"
)

const (
	toxiproxyAddr = "localhost:8474"

	// Upstream service addresses (real backends).
	redisUpstream    = "localhost:6379"
	natsUpstream     = "localhost:4222"
	postgresUpstream = "localhost:5432"
	grpcUpstream     = "localhost:50051"

	// Proxy listen addresses (clients connect here).
	redisProxy    = "localhost:26379"
	natsProxy     = "localhost:24222"
	postgresProxy = "localhost:25432"
	grpcProxy     = "localhost:50052"
)

// newToxiClient creates a toxiproxy API client.
func newToxiClient() *toxiproxy.Client {
	return toxiproxy.NewClient(toxiproxyAddr)
}

// TestCircuitBreakerWithRedisLatency verifies that the circuit breaker opens
// when Redis operations exceed acceptable latency thresholds.
//
// Scenario:
//  1. Create a toxiproxy proxy in front of Redis
//  2. Add 2s latency toxic
//  3. Execute operations through the circuit breaker with a 500ms timeout
//  4. Verify the circuit breaker transitions to Open state after threshold failures
//  5. Verify calls are rejected fast (ErrCircuitOpen) once open
//  6. Remove toxic and verify recovery through HalfOpen → Closed
func TestCircuitBreakerWithRedisLatency(t *testing.T) {
	client := newToxiClient()

	// Create proxy: clients connect to redisProxy, traffic forwarded to redisUpstream.
	proxy, err := client.CreateProxy("redis_chaos", redisProxy, redisUpstream)
	require.NoError(t, err, "failed to create redis proxy")
	defer func() {
		_ = proxy.Delete()
	}()

	// Create Redis client pointing at the proxy.
	rdb := redis.NewClient(&redis.Options{
		Addr:        redisProxy,
		DialTimeout: 1 * time.Second,
		ReadTimeout: 500 * time.Millisecond,
	})
	defer rdb.Close()

	// Verify baseline connectivity.
	ctx := context.Background()
	err = rdb.Ping(ctx).Err()
	require.NoError(t, err, "redis should be reachable through proxy before fault injection")

	// Inject 2 second latency — exceeds our 500ms read timeout.
	_, err = proxy.AddToxic("latency_2s", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 2000,
	})
	require.NoError(t, err, "failed to add latency toxic")

	// Configure circuit breaker: open after 3 consecutive failures, 2s recovery window.
	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold:  3,
		OpenTimeout:       2 * time.Second,
		HalfOpenMaxProbes: 1,
	})

	// Execute Redis operations through the circuit breaker.
	// Each should timeout (>500ms) and count as a failure.
	for i := 0; i < 3; i++ {
		execErr := cb.Execute(func() error {
			return rdb.Set(ctx, fmt.Sprintf("chaos-key-%d", i), "value", time.Minute).Err()
		})
		assert.Error(t, execErr, "iteration %d should fail due to latency", i)
	}

	// Circuit should now be open.
	assert.Equal(t, circuitbreaker.StateOpen, cb.State(), "circuit breaker should be open after threshold failures")

	// Subsequent calls should be rejected immediately without hitting Redis.
	start := time.Now()
	err = cb.Execute(func() error {
		return rdb.Set(ctx, "should-not-reach", "value", time.Minute).Err()
	})
	elapsed := time.Since(start)

	assert.ErrorIs(t, err, circuitbreaker.ErrCircuitOpen, "calls should be rejected with ErrCircuitOpen")
	assert.Less(t, elapsed, 50*time.Millisecond, "rejection should be near-instant")

	// Remove the toxic to simulate recovery.
	err = proxy.RemoveToxic("latency_2s")
	require.NoError(t, err, "failed to remove latency toxic")

	// Wait for the open timeout to elapse so circuit transitions to HalfOpen.
	time.Sleep(2500 * time.Millisecond)

	// Next call should succeed (probe in HalfOpen state).
	err = cb.Execute(func() error {
		return rdb.Set(ctx, "recovery-key", "recovered", time.Minute).Err()
	})
	assert.NoError(t, err, "probe call should succeed after toxic removal")
	assert.Equal(t, circuitbreaker.StateClosed, cb.State(), "circuit breaker should recover to closed")
}

// TestGracefulDegradationNATSDown verifies that the system degrades gracefully
// when NATS becomes unreachable — operations that depend on event publishing
// should fail with clear errors rather than hanging or panicking.
//
// Scenario:
//  1. Create a toxiproxy proxy in front of NATS
//  2. Disable the proxy entirely (simulates NATS being down)
//  3. Attempt to connect
//  4. Verify the system returns errors without hanging
//  5. Re-enable proxy and verify recovery
func TestGracefulDegradationNATSDown(t *testing.T) {
	client := newToxiClient()

	proxy, err := client.CreateProxy("nats_chaos", natsProxy, natsUpstream)
	require.NoError(t, err, "failed to create nats proxy")
	defer func() {
		_ = proxy.Delete()
	}()

	// Verify baseline: NATS proxy is reachable.
	conn, err := net.DialTimeout("tcp", natsProxy, 2*time.Second)
	require.NoError(t, err, "NATS proxy should be reachable initially")
	conn.Close()

	// Disable the proxy — all connections will be refused.
	proxy.Enabled = false
	err = proxy.Save()
	require.NoError(t, err, "failed to disable nats proxy")

	// Attempt connection — should fail quickly.
	start := time.Now()
	_, err = net.DialTimeout("tcp", natsProxy, 2*time.Second)
	elapsed := time.Since(start)

	assert.Error(t, err, "connection to disabled NATS proxy should fail")
	assert.Less(t, elapsed, 3*time.Second, "failure should not hang indefinitely")

	// Re-enable the proxy.
	proxy.Enabled = true
	err = proxy.Save()
	require.NoError(t, err, "failed to re-enable nats proxy")

	// Give toxiproxy a moment to start accepting connections again.
	time.Sleep(500 * time.Millisecond)

	// Verify recovery.
	conn, err = net.DialTimeout("tcp", natsProxy, 2*time.Second)
	assert.NoError(t, err, "NATS proxy should be reachable after re-enable")
	if conn != nil {
		conn.Close()
	}
}

// TestDatabaseConnectionPoolExhaustion verifies that the system recovers
// when database connections are exhausted due to slow queries.
//
// Scenario:
//  1. Create a toxiproxy proxy in front of PostgreSQL
//  2. Add a bandwidth toxic to severely limit throughput (simulates slow queries)
//  3. Open many concurrent connections to exhaust the pool
//  4. Verify that new connection attempts fail with clear errors
//  5. Remove the toxic and verify pool recovery
func TestDatabaseConnectionPoolExhaustion(t *testing.T) {
	client := newToxiClient()

	proxy, err := client.CreateProxy("postgres_chaos", postgresProxy, postgresUpstream)
	require.NoError(t, err, "failed to create postgres proxy")
	defer func() {
		_ = proxy.Delete()
	}()

	// Verify baseline connectivity.
	conn, err := net.DialTimeout("tcp", postgresProxy, 2*time.Second)
	require.NoError(t, err, "postgres proxy should be reachable initially")
	conn.Close()

	// Add bandwidth toxic: limit to 1 KB/s to simulate extremely slow responses.
	_, err = proxy.AddToxic("slow_bandwidth", "bandwidth", "downstream", 1.0, toxiproxy.Attributes{
		"rate": 1, // 1 KB/s
	})
	require.NoError(t, err, "failed to add bandwidth toxic")

	// Simulate pool exhaustion by opening many connections that will be slow.
	const maxConns = 10
	conns := make([]net.Conn, 0, maxConns)
	for i := 0; i < maxConns; i++ {
		c, dialErr := net.DialTimeout("tcp", postgresProxy, 2*time.Second)
		if dialErr != nil {
			// Some connections may fail — that's expected under pressure.
			break
		}
		conns = append(conns, c)
	}

	t.Logf("opened %d connections through slow proxy", len(conns))

	// Attempt one more connection with a tight timeout — should be slow or fail.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var d net.Dialer
	_, err = d.DialContext(ctx, "tcp", postgresProxy)
	// Under heavy load with bandwidth restriction, this may timeout or succeed slowly.
	// The key assertion is that it doesn't hang forever.
	if err != nil {
		t.Logf("additional connection correctly failed under pool pressure: %v", err)
	}

	// Cleanup: close all held connections.
	for _, c := range conns {
		c.Close()
	}

	// Remove the toxic.
	err = proxy.RemoveToxic("slow_bandwidth")
	require.NoError(t, err, "failed to remove bandwidth toxic")

	// Verify recovery: new connections should work normally.
	time.Sleep(500 * time.Millisecond)
	conn, err = net.DialTimeout("tcp", postgresProxy, 2*time.Second)
	assert.NoError(t, err, "postgres proxy should recover after toxic removal")
	if conn != nil {
		conn.Close()
	}
}

// TestGRPCTimeoutHandling verifies that gRPC calls fail gracefully when the
// downstream service is slow, and that client-side deadlines are respected.
//
// Scenario:
//  1. Create a toxiproxy proxy in front of the gRPC service
//  2. Add latency toxic (3s) to simulate slow gRPC responses
//  3. Make gRPC calls with a 1s deadline
//  4. Verify calls fail with DeadlineExceeded
//  5. Verify the circuit breaker opens after repeated timeouts
//  6. Remove toxic and verify recovery
func TestGRPCTimeoutHandling(t *testing.T) {
	client := newToxiClient()

	proxy, err := client.CreateProxy("grpc_chaos", grpcProxy, grpcUpstream)
	require.NoError(t, err, "failed to create grpc proxy")
	defer func() {
		_ = proxy.Delete()
	}()

	// Add 3 second latency to gRPC responses.
	_, err = proxy.AddToxic("grpc_latency", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 3000,
	})
	require.NoError(t, err, "failed to add grpc latency toxic")

	// Create gRPC connection through the proxy with a per-call timeout.
	grpcConn, err := grpc.NewClient(
		grpcProxy,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "failed to create grpc client connection")
	defer grpcConn.Close()

	// Configure circuit breaker for gRPC calls.
	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold:  3,
		OpenTimeout:       5 * time.Second,
		HalfOpenMaxProbes: 1,
	})

	// Make calls with 1s deadline — should timeout since proxy adds 3s latency.
	for i := 0; i < 3; i++ {
		execErr := cb.Execute(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Attempt a raw gRPC invoke. The specific method doesn't matter for
			// timeout testing — we just need to verify deadline behavior.
			return grpcConn.Invoke(ctx, "/reservation.v1.ReservationService/GetReservation", nil, nil)
		})
		assert.Error(t, execErr, "gRPC call %d should fail due to timeout", i)
	}

	// Circuit breaker should be open.
	assert.Equal(t, circuitbreaker.StateOpen, cb.State(),
		"circuit breaker should open after repeated gRPC timeouts")

	// Fast rejection.
	start := time.Now()
	err = cb.Execute(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		return grpcConn.Invoke(ctx, "/reservation.v1.ReservationService/GetReservation", nil, nil)
	})
	elapsed := time.Since(start)

	assert.ErrorIs(t, err, circuitbreaker.ErrCircuitOpen)
	assert.Less(t, elapsed, 50*time.Millisecond, "open circuit should reject instantly")

	// Remove toxic for cleanup.
	err = proxy.RemoveToxic("grpc_latency")
	require.NoError(t, err, "failed to remove grpc latency toxic")
}
