package usecase

import (
	"context"
	"log/slog"
	"parkir-pintar/internal/presence"
	"parkir-pintar/pkg/logger"
)

//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/presence Usecase

func (uc *presenceUsecase) VerifyPresence(ctx context.Context, reservationID string, floorNumber int, spotNumber int) (*presence.VerifyResult, error) {
	reading, err := uc.sensor.CheckSpotOccupancy(ctx, floorNumber, spotNumber)
	if err != nil {
		slog.Warn("sensor check failed, assuming presence verified (graceful degradation)",
			slog.String("reservation_id", reservationID),
			slog.Int("floor_number", floorNumber),
			slog.Int("spot_number", spotNumber),
			logger.Err(err))
		return &presence.VerifyResult{
			Verified: true,
			Message:  "sensor unavailable, presence assumed",
		}, nil
	}

	if reading.Occupied {
		return &presence.VerifyResult{
			Verified: true,
			Message:  "spot occupied, presence confirmed",
		}, nil
	}

	return &presence.VerifyResult{
		Verified: false,
		Message:  "spot not occupied, driver may be at wrong spot",
	}, nil
}
