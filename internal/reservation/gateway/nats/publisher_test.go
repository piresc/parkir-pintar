package nats

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgnats "parkir-pintar/pkg/nats"
)

func TestNewPublisher_ShouldReturnNonNil(t *testing.T) {
	pub := NewPublisher(nil)
	require.NotNil(t, pub)
}

func TestPublishSpotUpdated_ShouldReturnError_WhenPublisherClientIsNil(t *testing.T) {
	// A pkgnats.Publisher with nil client returns ErrNilClient.
	innerPub := pkgnats.NewPublisher(nil)
	pub := NewPublisher(innerPub)

	event := SpotUpdatedEvent{
		SpotID:      "spot-123",
		FloorNumber: 1,
		SpotNumber:  5,
		VehicleType: "car",
		SpotCode:    "A-05",
		Status:      "reserved",
		UpdatedAt:   time.Now(),
	}

	err := pub.PublishSpotUpdated(context.Background(), event)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkgnats.ErrNilClient)
}

func TestPublishReservationEvent_ShouldReturnError_WhenPublisherClientIsNil(t *testing.T) {
	innerPub := pkgnats.NewPublisher(nil)
	pub := NewPublisher(innerPub)

	event := ReservationEvent{
		ReservationID: "res-456",
		DriverID:      "driver-789",
		SpotID:        "spot-123",
		VehicleType:   "car",
		Status:        "confirmed",
		Timestamp:     time.Now(),
	}

	err := pub.PublishReservationEvent(context.Background(), "reservation.analytics.confirmed", event)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkgnats.ErrNilClient)
}

func TestPublishSpotUpdated_ShouldPanic_WhenPublisherIsNil(t *testing.T) {
	pub := NewPublisher(nil)

	event := SpotUpdatedEvent{
		SpotID:      "spot-123",
		FloorNumber: 1,
		SpotNumber:  5,
		VehicleType: "car",
		SpotCode:    "A-05",
		Status:      "available",
		UpdatedAt:   time.Now(),
	}

	assert.Panics(t, func() {
		_ = pub.PublishSpotUpdated(context.Background(), event)
	})
}

func TestPublishReservationEvent_ShouldPanic_WhenPublisherIsNil(t *testing.T) {
	pub := NewPublisher(nil)

	event := ReservationEvent{
		ReservationID: "res-001",
		DriverID:      "driver-001",
		SpotID:        "spot-001",
		VehicleType:   "motorcycle",
		Status:        "created",
		Timestamp:     time.Now(),
	}

	assert.Panics(t, func() {
		_ = pub.PublishReservationEvent(context.Background(), "reservation.analytics.created", event)
	})
}

func TestSpotUpdatedEvent_ShouldBeTypeAlias(t *testing.T) {
	// Verify the type alias works correctly.
	event := SpotUpdatedEvent{
		SpotID:      "spot-abc",
		FloorNumber: 2,
		SpotNumber:  10,
		VehicleType: "car",
		SpotCode:    "B-10",
		Status:      "occupied",
		UpdatedAt:   time.Now(),
	}
	assert.Equal(t, "spot-abc", event.SpotID)
	assert.Equal(t, "occupied", event.Status)
}

func TestReservationEvent_ShouldBeTypeAlias(t *testing.T) {
	// Verify the type alias works correctly.
	event := ReservationEvent{
		ReservationID: "res-xyz",
		DriverID:      "driver-xyz",
		SpotID:        "spot-xyz",
		VehicleType:   "car",
		Status:        "cancelled",
		Timestamp:     time.Now(),
	}
	assert.Equal(t, "res-xyz", event.ReservationID)
	assert.Equal(t, "cancelled", event.Status)
}
