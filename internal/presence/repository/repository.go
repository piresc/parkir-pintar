// Package repository provides Redis Geo operations for the presence domain.
package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	// spotGeoKey is the Redis key for storing parking spot geo locations.
	spotGeoKey = "presence:spots"
	// driverGeoKey is the Redis key for storing driver geo locations.
	driverGeoKey = "presence:drivers"
)

// Repository defines the data access interface for presence geo operations.
type Repository interface {
	AddSpotLocation(ctx context.Context, spotID string, lat, lng float64) error
	GetDistanceToSpot(ctx context.Context, driverID, spotID string) (float64, error)
	UpdateDriverLocation(ctx context.Context, driverID string, lat, lng float64) error
}

// redisRepository is the Redis-backed implementation of Repository.
type redisRepository struct {
	client *redis.Client
}

// NewRepository creates a new Repository backed by the given Redis client.
func NewRepository(client *redis.Client) Repository {
	return &redisRepository{client: client}
}

// AddSpotLocation stores a parking spot's GPS coordinates using Redis GEOADD.
func (r *redisRepository) AddSpotLocation(ctx context.Context, spotID string, lat, lng float64) error {
	err := r.client.GeoAdd(ctx, spotGeoKey, &redis.GeoLocation{
		Name:      spotID,
		Latitude:  lat,
		Longitude: lng,
	}).Err()
	if err != nil {
		return fmt.Errorf("geo add spot %s: %w", spotID, err)
	}
	return nil
}

// GetDistanceToSpot calculates the distance in meters between a driver and a spot.
// Both must have been previously stored via UpdateDriverLocation and AddSpotLocation.
func (r *redisRepository) GetDistanceToSpot(ctx context.Context, driverID, spotID string) (float64, error) {
	// Store driver and spot in the same geo set for distance calculation.
	// We use a combined key for distance queries.
	dist, err := r.client.GeoDist(ctx, spotGeoKey, driverID, spotID, "m").Result()
	if err != nil {
		return 0, fmt.Errorf("geo dist driver=%s spot=%s: %w", driverID, spotID, err)
	}
	return dist, nil
}

// UpdateDriverLocation stores a driver's current GPS coordinates using Redis GEOADD.
// The driver location is stored in the same geo set as spots to enable GEODIST.
func (r *redisRepository) UpdateDriverLocation(ctx context.Context, driverID string, lat, lng float64) error {
	// Store in the spot geo key so we can compute distance between driver and spot.
	err := r.client.GeoAdd(ctx, spotGeoKey, &redis.GeoLocation{
		Name:      driverID,
		Latitude:  lat,
		Longitude: lng,
	}).Err()
	if err != nil {
		return fmt.Errorf("geo add driver %s: %w", driverID, err)
	}
	// Also store in a separate driver-only key for tracking purposes.
	err = r.client.GeoAdd(ctx, driverGeoKey, &redis.GeoLocation{
		Name:      driverID,
		Latitude:  lat,
		Longitude: lng,
	}).Err()
	if err != nil {
		return fmt.Errorf("geo add driver tracking %s: %w", driverID, err)
	}
	return nil
}
