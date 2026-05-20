// Package reservation defines the domain interfaces for the reservation service.
// Implementations live in sub-packages (usecase/, repository/, handler/).
package reservation

import (
	"context"

	"parkir-pintar/internal/reservation/model"
)

// Usecase defines the business logic interface for the reservation lifecycle.
//
//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks parkir-pintar/internal/reservation Usecase
type Usecase interface {
	CreateReservation(ctx context.Context, req *model.CreateReservationRequest) (*model.Reservation, error)
	GetReservation(ctx context.Context, id string, callerID string) (*model.Reservation, error)
	CancelReservation(ctx context.Context, req *model.CancelReservationRequest) (*model.Reservation, error)
	CheckIn(ctx context.Context, req *model.CheckInRequest) (*model.CheckInResponse, error)
	CheckOut(ctx context.Context, req *model.CheckOutRequest) (*model.CheckOutResponse, error)
	ConfirmReservation(ctx context.Context, req *model.ConfirmReservationRequest) (*model.Reservation, error)
	CompleteCheckout(ctx context.Context, req *model.CompleteCheckoutRequest) (*model.CheckOutResponse, error)
	ExpireReservation(ctx context.Context, req *model.ExpireReservationRequest) error
	FailReservation(ctx context.Context, req *model.FailReservationRequest) error
	ListByDriver(ctx context.Context, driverID string, status string) ([]*model.Reservation, error)
}
