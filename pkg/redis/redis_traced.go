package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"

	"parkir-pintar/pkg/tracing"
)

// TracedRedisClient wraps RedisClient with automatic OTEL tracing.
type TracedRedisClient struct {
	*RedisClient
	tracer tracing.Tracer
}

// NewTracedRedisClient creates a new traced Redis client.
func NewTracedRedisClient(client *RedisClient, tracer tracing.Tracer) *TracedRedisClient {
	return &TracedRedisClient{RedisClient: client, tracer: tracer}
}

// Get retrieves a value by key with automatic tracing.
func (r *TracedRedisClient) Get(ctx context.Context, key string) (string, error) {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.Get(ctx, key)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "GET", "redis")
	defer done()
	return r.RedisClient.Get(ctx, key)
}

// Set stores a key-value pair with automatic tracing.
func (r *TracedRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.Set(ctx, key, value, expiration)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "SET", "redis")
	defer done()
	return r.RedisClient.Set(ctx, key, value, expiration)
}

// SetNX sets value if key doesn't exist with automatic tracing.
func (r *TracedRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.SetNX(ctx, key, value, expiration)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "SETNX", "redis")
	defer done()
	return r.RedisClient.SetNX(ctx, key, value, expiration)
}

// Delete removes a key with automatic tracing.
func (r *TracedRedisClient) Delete(ctx context.Context, key string) error {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.Delete(ctx, key)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "DEL", "redis")
	defer done()
	return r.RedisClient.Delete(ctx, key)
}

// HMSet sets multiple hash fields with automatic tracing.
func (r *TracedRedisClient) HMSet(ctx context.Context, key string, values map[string]interface{}) error {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.HMSet(ctx, key, values)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "HMSET", "redis")
	defer done()
	return r.RedisClient.HMSet(ctx, key, values)
}

// HGetAll gets all fields in a hash with automatic tracing.
func (r *TracedRedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.HGetAll(ctx, key)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "HGETALL", "redis")
	defer done()
	return r.RedisClient.HGetAll(ctx, key)
}

// HMGet gets specified fields of a hash with automatic tracing.
func (r *TracedRedisClient) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.HMGet(ctx, key, fields...)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "HMGET", "redis")
	defer done()
	return r.RedisClient.HMGet(ctx, key, fields...)
}

// GeoAdd adds geospatial data with automatic tracing.
func (r *TracedRedisClient) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.GeoAdd(ctx, key, longitude, latitude, member)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "GEOADD", "redis")
	defer done()
	return r.RedisClient.GeoAdd(ctx, key, longitude, latitude, member)
}

// GeoRadius finds members within a radius with automatic tracing.
func (r *TracedRedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, unit string) ([]redis.GeoLocation, error) {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.GeoRadius(ctx, key, longitude, latitude, radius, unit)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "GEORADIUS", "redis")
	defer done()
	return r.RedisClient.GeoRadius(ctx, key, longitude, latitude, radius, unit)
}

// SAdd adds members to a set with automatic tracing.
func (r *TracedRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.SAdd(ctx, key, members...)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "SADD", "redis")
	defer done()
	return r.RedisClient.SAdd(ctx, key, members...)
}

// SIsMember checks if a value is a member of a set with automatic tracing.
func (r *TracedRedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.SIsMember(ctx, key, member)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "SISMEMBER", "redis")
	defer done()
	return r.RedisClient.SIsMember(ctx, key, member)
}

// SRem removes members from a set with automatic tracing.
func (r *TracedRedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.SRem(ctx, key, members...)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "SREM", "redis")
	defer done()
	return r.RedisClient.SRem(ctx, key, members...)
}

// ZRem removes members from a sorted set with automatic tracing.
func (r *TracedRedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.ZRem(ctx, key, members...)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "ZREM", "redis")
	defer done()
	return r.RedisClient.ZRem(ctx, key, members...)
}

// Expire sets an expiration on a key with automatic tracing.
func (r *TracedRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if !r.tracer.IsEnabled() {
		return r.RedisClient.Expire(ctx, key, expiration)
	}
	ctx, done := r.tracer.StartDatabase(ctx, "EXPIRE", "redis")
	defer done()
	return r.RedisClient.Expire(ctx, key, expiration)
}

// Close closes the Redis client.
func (r *TracedRedisClient) Close() error {
	return r.RedisClient.Close()
}
