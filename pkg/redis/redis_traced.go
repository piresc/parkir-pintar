package redis

import (
	"context"
	"time"

	"parkir-pintar/pkg/tracing"
)

// TracedRedisClient wraps Client with automatic OTEL tracing.
type TracedRedisClient struct {
	*Client
	tracer tracing.Tracer
}

// NewTracedRedisClient creates a new traced Redis client.
func NewTracedRedisClient(client *Client, tracer tracing.Tracer) *TracedRedisClient {
	return &TracedRedisClient{Client: client, tracer: tracer}
}

// Get retrieves a value by key with automatic tracing.
func (r *TracedRedisClient) Get(ctx context.Context, key string) (string, error) {
	if !r.tracer.IsEnabled() {
		return r.Client.Get(ctx, key)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "GET", "redis")
	defer done()
	return r.Client.Get(ctx, key)
}

// Set stores a key-value pair with automatic tracing.
func (r *TracedRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !r.tracer.IsEnabled() {
		return r.Client.Set(ctx, key, value, expiration)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "SET", "redis")
	defer done()
	return r.Client.Set(ctx, key, value, expiration)
}

// SetNX sets value if key doesn't exist with automatic tracing.
func (r *TracedRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	if !r.tracer.IsEnabled() {
		return r.Client.SetNX(ctx, key, value, expiration)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "SETNX", "redis")
	defer done()
	return r.Client.SetNX(ctx, key, value, expiration)
}

// Delete removes a key with automatic tracing.
func (r *TracedRedisClient) Delete(ctx context.Context, key string) error {
	if !r.tracer.IsEnabled() {
		return r.Client.Delete(ctx, key)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "DEL", "redis")
	defer done()
	return r.Client.Delete(ctx, key)
}

// Close closes the Redis client.
func (r *TracedRedisClient) Close() error {
	return r.Client.Close()
}
