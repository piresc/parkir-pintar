// Package usecase implements the business logic layer for the search domain
// module. It provides availability queries with Redis caching (5s TTL) and
// graceful fallback to PostgreSQL on cache miss or Redis failure.
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Use context.Context as first parameter for consistency
// - Handle errors explicitly; never ignore errors
// - Keep interfaces small and focused
// - Mock at interface boundaries rather than concrete implementations
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/singleflight"

	"parkir-pintar/internal/search/model"
	"parkir-pintar/internal/search/repository"
)

// cacheTTL is the Redis cache time-to-live for availability and floor map data.
const cacheTTL = 5 * time.Second

// RedisClient defines the interface for Redis cache operations.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Usecase defines the business logic interface for search operations.
type Usecase interface {
	GetAvailability(ctx context.Context, req *model.GetAvailabilityRequest) ([]model.FloorAvailability, error)
	GetFloorMap(ctx context.Context, req *model.GetFloorMapRequest) ([]model.SpotDetails, error)
	GetSpotDetails(ctx context.Context, req *model.GetSpotDetailsRequest) (*model.SpotDetails, error)
}

// searchUsecase is the concrete implementation of Usecase.
type searchUsecase struct {
	repo  repository.Repository
	redis RedisClient
	sf    singleflight.Group
}

// NewUsecase creates a new search Usecase with all required dependencies.
func NewUsecase(repo repository.Repository, redis RedisClient) Usecase {
	return &searchUsecase{
		repo:  repo,
		redis: redis,
	}
}

// GetAvailability returns per-floor availability. It tries Redis cache first
// (key "availability:{vehicleType}", 5s TTL), falling back to PostgreSQL on
// cache miss or Redis error.
func (uc *searchUsecase) GetAvailability(ctx context.Context, req *model.GetAvailabilityRequest) ([]model.FloorAvailability, error) {
	cacheKey := fmt.Sprintf("availability:%s", req.VehicleType)

	// Try cache first
	cached, err := uc.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var floors []model.FloorAvailability
		if jsonErr := json.Unmarshal([]byte(cached), &floors); jsonErr == nil {
			return floors, nil
		}
		slog.Warn("search: failed to unmarshal cached availability", slog.Any("error", err))
	}

	// Cache miss or Redis error — fall back to DB (coalesced via singleflight)
	result, err, _ := uc.sf.Do(cacheKey, func() (interface{}, error) {
		floors, err := uc.repo.GetAvailabilityByVehicleType(ctx, req.VehicleType)
		if err != nil {
			return nil, fmt.Errorf("get availability: %w", err)
		}

		// Store in cache (non-critical)
		if data, jsonErr := json.Marshal(floors); jsonErr == nil {
			if setErr := uc.redis.Set(ctx, cacheKey, string(data), cacheTTL); setErr != nil {
				slog.Warn("search: failed to cache availability", slog.Any("error", setErr))
			}
		}

		return floors, nil
	})
	if err != nil {
		return nil, err
	}

	return result.([]model.FloorAvailability), nil
}

// GetFloorMap returns all spots on a floor. It tries Redis cache first
// (key "floormap:{floorNumber}", 5s TTL), falling back to PostgreSQL.
func (uc *searchUsecase) GetFloorMap(ctx context.Context, req *model.GetFloorMapRequest) ([]model.SpotDetails, error) {
	cacheKey := fmt.Sprintf("floormap:%d", req.FloorNumber)

	// Try cache first
	cached, err := uc.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var spots []model.SpotDetails
		if jsonErr := json.Unmarshal([]byte(cached), &spots); jsonErr == nil {
			return spots, nil
		}
		slog.Warn("search: failed to unmarshal cached floor map", slog.Any("error", err))
	}

	// Cache miss or Redis error — fall back to DB (coalesced via singleflight)
	result, err, _ := uc.sf.Do(cacheKey, func() (interface{}, error) {
		spots, err := uc.repo.GetFloorSpots(ctx, req.FloorNumber)
		if err != nil {
			return nil, fmt.Errorf("get floor map: %w", err)
		}

		// Store in cache (non-critical)
		if data, jsonErr := json.Marshal(spots); jsonErr == nil {
			if setErr := uc.redis.Set(ctx, cacheKey, string(data), cacheTTL); setErr != nil {
				slog.Warn("search: failed to cache floor map", slog.Any("error", setErr))
			}
		}

		return spots, nil
	})
	if err != nil {
		return nil, err
	}

	return result.([]model.SpotDetails), nil
}

// GetSpotDetails returns details for a single spot. No caching — direct DB query.
func (uc *searchUsecase) GetSpotDetails(ctx context.Context, req *model.GetSpotDetailsRequest) (*model.SpotDetails, error) {
	spot, err := uc.repo.GetSpotByID(ctx, req.SpotID)
	if err != nil {
		return nil, fmt.Errorf("get spot details: %w", err)
	}
	return spot, nil
}
