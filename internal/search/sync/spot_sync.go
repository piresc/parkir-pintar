package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

func (s *SpotSync) HandleNATSEvent(ctx context.Context, subject string, data []byte) {
	var spot SpotData
	if err := json.Unmarshal(data, &spot); err != nil {
		slog.Warn("spot sync: failed to unmarshal event", slog.String("subject", subject), slog.Any("error", err))
		return
	}
	if err := s.HandleSpotUpdated(ctx, spot); err != nil {
		slog.Warn("spot sync: failed to upsert spot", slog.String("spot_id", spot.ID), slog.Any("error", err))
	}
}
