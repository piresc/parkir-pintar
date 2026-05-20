// - Handle errors explicitly; never ignore errors
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"parkir-pintar/internal/search/model"
)

const cacheTTL = 5 * time.Second

//go:generate mockgen -destination=../mocks/mock_redis_client.go -package=mocks parkir-pintar/internal/search/usecase RedisClient
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/search/usecase Usecase
type Usecase interface {
	GetAvailability(ctx context.Context, req *model.GetAvailabilityRequest) ([]model.FloorAvailability, error)
	GetFloorMap(ctx context.Context, req *model.GetFloorMapRequest) ([]model.SpotDetails, error)
	GetSpotDetails(ctx context.Context, req *model.GetSpotDetailsRequest) (*model.SpotDetails, error)
}

func (uc *searchUsecase) GetAvailability(ctx context.Context, req *model.GetAvailabilityRequest) ([]model.FloorAvailability, error) {
	cacheKey := fmt.Sprintf("availability:%s", req.VehicleType)

	cached, err := uc.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var floors []model.FloorAvailability
		jsonErr := json.Unmarshal([]byte(cached), &floors)
		if jsonErr == nil {
			return floors, nil
		}
		slog.Warn("search: failed to unmarshal cached availability", slog.Any("error", jsonErr))
	}

	// Use detached context to prevent first-caller cancellation from affecting
	result, err, _ := uc.sf.Do(cacheKey, func() (interface{}, error) {
		sfCtx, sfCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer sfCancel()

		floors, err := uc.repo.GetAvailabilityByVehicleType(sfCtx, req.VehicleType)
		if err != nil {
			return nil, fmt.Errorf("get availability: %w", err)
		}

		if data, jsonErr := json.Marshal(floors); jsonErr == nil {
			if setErr := uc.redis.Set(sfCtx, cacheKey, string(data), cacheTTL); setErr != nil {
				slog.Warn("search: failed to cache availability", slog.Any("error", setErr))
			}
		}

		return floors, nil
	})
	if err != nil {
		return nil, err
	}

	floors, ok := result.([]model.FloorAvailability)
	if !ok {
		return nil, fmt.Errorf("unexpected type in singleflight result")
	}
	return floors, nil
}

func (uc *searchUsecase) GetFloorMap(ctx context.Context, req *model.GetFloorMapRequest) ([]model.SpotDetails, error) {
	cacheKey := fmt.Sprintf("floormap:%d", req.FloorNumber)

	cached, err := uc.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var spots []model.SpotDetails
		jsonErr := json.Unmarshal([]byte(cached), &spots)
		if jsonErr == nil {
			return spots, nil
		}
		slog.Warn("search: failed to unmarshal cached floor map", slog.Any("error", jsonErr))
	}

	// Use detached context to prevent first-caller cancellation from affecting
	result, err, _ := uc.sf.Do(cacheKey, func() (interface{}, error) {
		sfCtx, sfCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer sfCancel()

		spots, err := uc.repo.GetFloorSpots(sfCtx, req.FloorNumber)
		if err != nil {
			return nil, fmt.Errorf("get floor map: %w", err)
		}

		if data, jsonErr := json.Marshal(spots); jsonErr == nil {
			if setErr := uc.redis.Set(sfCtx, cacheKey, string(data), cacheTTL); setErr != nil {
				slog.Warn("search: failed to cache floor map", slog.Any("error", setErr))
			}
		}

		return spots, nil
	})
	if err != nil {
		return nil, err
	}

	spots, ok := result.([]model.SpotDetails)
	if !ok {
		return nil, fmt.Errorf("unexpected type in singleflight result")
	}
	return spots, nil
}

func (uc *searchUsecase) GetSpotDetails(ctx context.Context, req *model.GetSpotDetailsRequest) (*model.SpotDetails, error) {
	spot, err := uc.repo.GetSpotByID(ctx, req.SpotID)
	if err != nil {
		return nil, fmt.Errorf("get spot details: %w", err)
	}
	return spot, nil
}
