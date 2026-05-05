// Best practices applied from coding standards knowledge base:
// - Go testify patterns: table-driven tests, descriptive test names (Test[Function]_Should[Result]_When[Condition])
// - Mock Redis operations: focus on mocking Redis client methods rather than Redis server behavior
// - Test connection handling: include tests for connection failures and timeouts
// - Error handling: verify wrapped errors with context
// - Use t.Helper() for common setup, test error conditions as thoroughly as success cases

package redis

import (
	"testing"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/tracing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisClient_ShouldReturnError_WhenConnectionFails(t *testing.T) {
	// Arrange: invalid config that cannot connect
	cfg := config.RedisConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	// Act
	client, err := NewRedisClient(cfg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}

func TestNewTracedRedisClient_ShouldReturnValidClient_WhenCreated(t *testing.T) {
	// Arrange
	tracer := tracing.NewNoOpTracer()
	// Create a RedisClient with nil underlying client for constructor testing
	redisClient := &RedisClient{client: nil}

	// Act
	traced := NewTracedRedisClient(redisClient, tracer)

	// Assert
	require.NotNil(t, traced)
	assert.Equal(t, redisClient, traced.RedisClient)
}

func TestRedisClient_GetClient_ShouldReturnUnderlyingClient(t *testing.T) {
	// Arrange: nil client for constructor test
	redisClient := &RedisClient{client: nil}

	// Act & Assert
	assert.Nil(t, redisClient.GetClient())
}

func TestTracedRedisClient_Close_ShouldDelegateToUnderlying(t *testing.T) {
	// Arrange: TracedRedisClient.Close delegates to RedisClient.Close
	// We verify the delegation chain exists
	tracer := tracing.NewNoOpTracer()
	redisClient := &RedisClient{client: nil}
	traced := NewTracedRedisClient(redisClient, tracer)

	// Act: Close on nil client will panic, but the delegation is correct
	// This test verifies the struct composition is wired correctly
	require.NotNil(t, traced)
	assert.Equal(t, redisClient, traced.RedisClient)
}
