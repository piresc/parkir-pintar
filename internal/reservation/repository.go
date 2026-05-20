package reservation

import (
	"context"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/reservation/model"
)

// Repository defines the data access interface for reservations and parking spots.
//
//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks parkir-pintar/internal/reservation Repository
type Repository interface {
	FindByIdempotencyKey(ctx context.Context, key string) (*model.Reservation, error)
	FindAvailableSpot(ctx context.Context, vehicleType string) (*model.ParkingSpot, error)
	GetSpotByID(ctx context.Context, spotID string) (*model.ParkingSpot, error)
	GetSpotForUpdate(ctx context.Context, spotID string) (*model.ParkingSpot, error)
	GetSpotForUpdateTx(ctx context.Context, tx *sqlx.Tx, spotID string) (*model.ParkingSpot, error)
	CreateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error
	UpdateSpotStatusTx(ctx context.Context, tx *sqlx.Tx, spotID string, status string) error
	UpdateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error
	GetByID(ctx context.Context, id string) (*model.Reservation, error)
	GetByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id string) (*model.Reservation, error)
	WithTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error
	ListByDriverID(ctx context.Context, driverID string, status string) ([]*model.Reservation, error)
}
