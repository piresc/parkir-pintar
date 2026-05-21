package search

import (
	"context"
	"time"

	"parkir-pintar/internal/search/model"
)

// SpotData represents a parking spot for the read model sync layer.
type SpotData struct {
	ID          string
	FloorNumber int
	SpotNumber  int
	VehicleType string
	SpotCode    string
	Status      string
}

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks parkir-pintar/internal/search Usecase
type Usecase interface {
	GetAvailability(ctx context.Context, req *model.GetAvailabilityRequest) ([]model.FloorAvailability, error)
	GetFloorMap(ctx context.Context, req *model.GetFloorMapRequest) ([]model.SpotDetails, error)
	GetSpotDetails(ctx context.Context, req *model.GetSpotDetailsRequest) (*model.SpotDetails, error)
	HandleSpotUpdated(ctx context.Context, spot SpotData) error
}

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks parkir-pintar/internal/search Repository
type Repository interface {
	GetAvailabilityByVehicleType(ctx context.Context, vehicleType string) ([]model.FloorAvailability, error)
	GetFloorSpots(ctx context.Context, floorNumber int) ([]model.SpotDetails, error)
	GetSpotByID(ctx context.Context, spotID string) (*model.SpotDetails, error)
}

//go:generate mockgen -destination=mocks/mock_read_model_repository.go -package=mocks parkir-pintar/internal/search ReadModelRepository
type ReadModelRepository interface {
	UpsertSpot(ctx context.Context, spot SpotData) error
	DeleteSpot(ctx context.Context, spotID string) error
}

//go:generate mockgen -destination=mocks/mock_redis_client.go -package=mocks parkir-pintar/internal/search RedisClient
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}
