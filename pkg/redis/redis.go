// Package redis provides a Redis client with connection pooling, full data
// structure support (strings, hashes, sets, sorted sets, geo), and a traced
// wrapper for automatic OTEL span creation.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"parkir-pintar/pkg/config"
)

// RedisClient represents a Redis client with connection pooling.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client with the given configuration.
// It configures address, password, DB, and pool size, then verifies
// connectivity with a 5-second timeout ping.
func NewRedisClient(cfg config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Verify connection with 5s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// GetClient returns the underlying redis.Client instance.
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Get retrieves a value by key.
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set stores a key-value pair with an optional expiration.
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// SetNX sets value if key doesn't exist (Set if Not eXists).
// Returns true if key was set, false if key already exists.
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// Delete removes a key.
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// HMSet sets multiple hash fields.
func (r *RedisClient) HMSet(ctx context.Context, key string, values map[string]interface{}) error {
	return r.client.HMSet(ctx, key, values).Err()
}

// HGetAll gets all fields in a hash.
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// HMGet gets specified fields of a hash.
func (r *RedisClient) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	return r.client.HMGet(ctx, key, fields...).Result()
}

// GeoAdd adds geospatial data to a sorted set.
func (r *RedisClient) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	return r.client.GeoAdd(ctx, key, &redis.GeoLocation{
		Longitude: longitude,
		Latitude:  latitude,
		Name:      member,
	}).Err()
}

// GeoRadius finds members within a radius from a point.
func (r *RedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, unit string) ([]redis.GeoLocation, error) {
	return r.client.GeoRadius(ctx, key, longitude, latitude, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      unit,
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
	}).Result()
}

// SAdd adds members to a set.
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}

// SIsMember checks if a value is a member of a set.
func (r *RedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.client.SIsMember(ctx, key, member).Result()
}

// SRem removes members from a set.
func (r *RedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SRem(ctx, key, members...).Err()
}

// ZRem removes members from a sorted set.
func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.ZRem(ctx, key, members...).Err()
}

// Expire sets an expiration on a key.
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// Close closes the Redis client.
func (r *RedisClient) Close() error {
	return r.client.Close()
}
