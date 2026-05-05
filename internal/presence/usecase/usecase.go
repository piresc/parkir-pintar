// Package usecase implements the business logic layer for the presence domain
// module. It orchestrates location streaming, geofence arrival detection,
// wrong-spot detection, and presence retrieval, coordinating with the
// repository, Redis (streams), and NATS (event publishing).
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"parkir-pintar/internal/presence/model"
	"parkir-pintar/internal/presence/repository"
)

// RedisClient defines the interface for Redis stream operations.
type RedisClient interface {
	XAdd(ctx context.Context, stream string, values map[string]interface{}) error
	Delete(ctx context.Context, key string) error
}

// NATSClient defines the interface for NATS JetStream event publishing.
type NATSClient interface {
	Publish(subject string, data []byte) error
}

// Usecase defines the business logic interface for presence operations.
type Usecase interface {
	StreamLocation(ctx context.Context, update *model.LocationUpdate) error
	DetectArrival(ctx context.Context, lat, lng, centerLat, centerLng, radiusMeters float64, reservationID string) (*model.ArrivalResult, error)
	DetectWrongSpot(ctx context.Context, lat, lng, spotLat, spotLng, thresholdMeters float64, reservationID string) (*model.WrongSpotResult, error)
	GetPresence(ctx context.Context, reservationID string) (*model.PresenceLog, error)
}

// presenceUsecase is the concrete implementation of Usecase.
type presenceUsecase struct {
	repo  repository.Repository
	redis RedisClient
	nats  NATSClient
}

// NewUsecase creates a new presence Usecase with all required dependencies.
func NewUsecase(repo repository.Repository, redis RedisClient, nats NATSClient) Usecase {
	return &presenceUsecase{
		repo:  repo,
		redis: redis,
		nats:  nats,
	}
}

// StreamLocation saves a location update to Redis stream and persists it to PostgreSQL.
func (uc *presenceUsecase) StreamLocation(ctx context.Context, update *model.LocationUpdate) error {
	// Save to Redis stream
	streamKey := fmt.Sprintf("presence:%s", update.ReservationID)
	values := map[string]interface{}{
		"reservation_id": update.ReservationID,
		"latitude":       update.Latitude,
		"longitude":      update.Longitude,
		"accuracy":       update.Accuracy,
		"timestamp":      update.Timestamp.Format(time.RFC3339Nano),
	}
	if err := uc.redis.XAdd(ctx, streamKey, values); err != nil {
		slog.Error("failed to add to Redis stream",
			slog.String("stream", streamKey),
			slog.Any("error", err))
	}

	// Persist to PostgreSQL
	log := &model.PresenceLog{
		ID:            uuid.New().String(),
		ReservationID: update.ReservationID,
		Latitude:      update.Latitude,
		Longitude:     update.Longitude,
		Accuracy:      update.Accuracy,
		RecordedAt:    update.Timestamp,
	}
	if err := uc.repo.SavePresenceLog(ctx, log); err != nil {
		return fmt.Errorf("stream location persist: %w", err)
	}

	return nil
}

// DetectArrival checks if a driver has arrived within the parking geofence.
// If arrived, publishes a presence.arrival event via NATS.
func (uc *presenceUsecase) DetectArrival(ctx context.Context, lat, lng, centerLat, centerLng, radiusMeters float64, reservationID string) (*model.ArrivalResult, error) {
	arrived := model.DetectArrival(lat, lng, centerLat, centerLng, radiusMeters)

	result := &model.ArrivalResult{
		Arrived:       arrived,
		ReservationID: reservationID,
		DetectedAt:    time.Now(),
	}

	if arrived {
		uc.publishEvent("presence.arrival", result)
	}

	return result, nil
}

// DetectWrongSpot checks if a driver is parked in the wrong spot by comparing
// the Haversine distance against the threshold. If distance > threshold,
// publishes a presence.wrong_spot event via NATS.
func (uc *presenceUsecase) DetectWrongSpot(ctx context.Context, lat, lng, spotLat, spotLng, thresholdMeters float64, reservationID string) (*model.WrongSpotResult, error) {
	distance := model.HaversineDistance(lat, lng, spotLat, spotLng)
	isWrongSpot := distance > thresholdMeters

	result := &model.WrongSpotResult{
		IsWrongSpot:    isWrongSpot,
		DistanceMeters: distance,
	}

	if isWrongSpot {
		event := map[string]interface{}{
			"reservation_id":  reservationID,
			"is_wrong_spot":   true,
			"distance_meters": distance,
		}
		data, err := json.Marshal(event)
		if err != nil {
			slog.Error("failed to marshal wrong spot event", slog.Any("error", err))
			return result, nil
		}
		if err := uc.nats.Publish("presence.wrong_spot", data); err != nil {
			slog.Error("failed to publish wrong spot event", slog.Any("error", err))
		}
	}

	return result, nil
}

// GetPresence retrieves the latest presence log for a reservation from the repository.
func (uc *presenceUsecase) GetPresence(ctx context.Context, reservationID string) (*model.PresenceLog, error) {
	log, err := uc.repo.GetPresenceByReservation(ctx, reservationID)
	if err != nil {
		return nil, fmt.Errorf("get presence: %w", err)
	}
	return log, nil
}

// publishEvent serializes the data and publishes it to NATS.
// Errors are logged but do not fail the operation.
func (uc *presenceUsecase) publishEvent(subject string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal presence event",
			slog.String("subject", subject),
			slog.Any("error", err))
		return
	}
	if err := uc.nats.Publish(subject, payload); err != nil {
		slog.Error("failed to publish presence event",
			slog.String("subject", subject),
			slog.Any("error", err))
	}
}
