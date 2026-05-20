package bootstrap

import (
	"context"

	grpcgw "parkir-pintar/internal/reservation/gateway/grpc"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/internal/reservation/usecase"
)

type usecaseExpirerAdapter struct {
	uc usecase.Usecase
}

func (a *usecaseExpirerAdapter) ExpireReservation(ctx context.Context, reservationID string) error {
	return a.uc.ExpireReservation(ctx, &model.ExpireReservationRequest{
		ReservationID: reservationID,
	})
}

type usecaseFailerAdapter struct {
	uc usecase.Usecase
}

func (a *usecaseFailerAdapter) FailReservation(ctx context.Context, reservationID string, _ string) error {
	return a.uc.FailReservation(ctx, &model.FailReservationRequest{
		ReservationID: reservationID,
	})
}

type presenceClientAdapter struct {
	inner grpcgw.PresenceClient
}

func (a *presenceClientAdapter) VerifyPresence(ctx context.Context, driverID string, reservationID string, floorNumber int, spotNumber int) (*usecase.PresenceResult, error) {
	result, err := a.inner.VerifyPresence(ctx, driverID, reservationID, floorNumber, spotNumber)
	if err != nil {
		return nil, err
	}
	return &usecase.PresenceResult{
		Verified: result.Verified,
		Message:  result.Message,
	}, nil
}
