package redis

import (
	"testing"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/tracing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_ShouldReturnError_WhenConnectionFails(t *testing.T) {
	// Arrange: invalid config that cannot connect
	cfg := config.RedisConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	client, err := NewClient(cfg)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}

func TestNewTracedRedisClient_ShouldReturnValidClient_WhenCreated(t *testing.T) {
	tracer := tracing.NewNoOpTracer()
	redisClient := &Client{client: nil}

	traced := NewTracedRedisClient(redisClient, tracer)

	require.NotNil(t, traced)
	assert.Equal(t, redisClient, traced.Client)
}

func TestRedisClient_GetClient_ShouldReturnUnderlyingClient(t *testing.T) {
	redisClient := &Client{client: nil}

	assert.Nil(t, redisClient.GetClient())
}

func TestTracedRedisClient_Close_ShouldDelegateToUnderlying(t *testing.T) {
	tracer := tracing.NewNoOpTracer()
	redisClient := &Client{client: nil}
	traced := NewTracedRedisClient(redisClient, tracer)

	require.NotNil(t, traced)
	assert.Equal(t, redisClient, traced.Client)
}
