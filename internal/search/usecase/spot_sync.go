package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"parkir-pintar/internal/search"
	"parkir-pintar/pkg/logger"
)

func (uc *searchUsecase) HandleSpotUpdated(ctx context.Context, spot search.SpotData) error {
	if err := uc.readModelRepo.UpsertSpot(ctx, spot); err != nil {
		return fmt.Errorf("upsert spot read model: %w", err)
	}

	// Invalidate affected cache keys
	keys := []string{
		fmt.Sprintf("availability:%s", spot.VehicleType),
		fmt.Sprintf("floormap:%d", spot.FloorNumber),
	}
	for _, key := range keys {
		if err := uc.redis.Delete(ctx, key); err != nil {
			slog.Warn("failed to invalidate cache", slog.String("key", key), logger.Err(err))
		}
	}

	return nil
}
