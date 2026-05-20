package usecase

import (
	"context"
	"log/slog"
	"parkir-pintar/pkg/logger"
)

type VerifyResult struct {
	Verified bool
	Message  string
}

//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/presence/usecase Usecase
type Usecase interface {
	VerifyPresence(ctx context.Context, reservationID string, floorNumber int, spotNumber int) (*VerifyResult, error)
}

func (uc *presenceUsecase) VerifyPresence(ctx context.Context, reservationID string, floorNumber int, spotNumber int) (*VerifyResult, error) {
	reading, err := uc.sensor.CheckSpotOccupancy(ctx, floorNumber, spotNumber)
	if err != nil {
		slog.Warn("sensor check failed, assuming presence verified (graceful degradation)",
			slog.String("reservation_id", reservationID),
			slog.Int("floor_number", floorNumber),
			slog.Int("spot_number", spotNumber),
			logger.Err(err))
		return &VerifyResult{
			Verified: true,
			Message:  "sensor unavailable, presence assumed",
		}, nil
	}

	if reading.Occupied {
		return &VerifyResult{
			Verified: true,
			Message:  "spot occupied, presence confirmed",
		}, nil
	}

	return &VerifyResult{
		Verified: false,
		Message:  "spot not occupied, driver may be at wrong spot",
	}, nil
}
