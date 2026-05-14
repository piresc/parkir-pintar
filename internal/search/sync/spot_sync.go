package sync

import (
	"context"
	"fmt"
)

type SpotData struct {
	ID          string `json:"id"`
	FloorNumber int    `json:"floor_number"`
	SpotNumber  int    `json:"spot_number"`
	VehicleType string `json:"vehicle_type"`
	SpotCode    string `json:"spot_code"`
	Status      string `json:"status"`
}

type SpotRepository interface {
	UpsertSpot(ctx context.Context, spot SpotData) error
	DeleteSpot(ctx context.Context, spotID string) error
}

type SpotSync struct {
	repo SpotRepository
}

func NewSpotSync(repo SpotRepository) *SpotSync {
	return &SpotSync{repo: repo}
}

func (s *SpotSync) HandleSpotUpdated(ctx context.Context, spot SpotData) error {
	if err := s.repo.UpsertSpot(ctx, spot); err != nil {
		return fmt.Errorf("upsert spot read model: %w", err)
	}
	return nil
}
