// Package usecase implements the business logic layer for the reservation domain.
//
// Overlap detection tests per PRD §17.1 verify that:
// - A spot whose status changes between FindAvailableSpot and GetSpotForUpdate is rejected
// - Reservations succeed when spots are available (baseline for overlap context)
// - Different vehicle types can reserve their matching available spots
package usecase

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
)

// TestCreateReservation_ShouldReject_WhenSpotAlreadyReserved verifies that when
// GetSpotForUpdate returns a spot with status != "available" (simulating a race
// where another request acquired the spot between FindAvailableSpot and
// GetSpotForUpdate), the reservation is rejected with a CONFLICT error containing
// "spot no longer available". CreateReservationTx and UpdateSpotStatusTx are NOT called.
func TestCreateReservation_ShouldReject_WhenSpotAlreadyReserved(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	repo.On("FindByIdempotencyKey", mock.Anything, "overlap-key").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-race",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	redis.On("SetNX", mock.Anything, "lock:spot:spot-race", "locked", 30*time.Second).Return(true, nil)
	redis.On("Delete", mock.Anything, "lock:spot:spot-race").Return(nil)
	// Race condition: another request reserved the spot between FindAvailableSpot and GetSpotForUpdate
	repo.On("GetSpotForUpdate", mock.Anything, "spot-race").Return(&model.ParkingSpot{
		ID:          "spot-race",
		VehicleType: "car",
		Status:      "reserved",
	}, nil)

	uc := NewUsecase(repo, redis, natsClient, billing, payment)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "overlap-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spot no longer available")
	// Transaction methods must NOT be called since we returned before the transaction block
	repo.AssertNotCalled(t, "CreateReservationTx")
	repo.AssertNotCalled(t, "UpdateSpotStatusTx")
	billing.AssertNotCalled(t, "StartBilling")
	natsClient.AssertNotCalled(t, "Publish")
	payment.AssertNotCalled(t, "ProcessPayment")
	repo.AssertExpectations(t)
	redis.AssertExpectations(t)
}

// TestCreateReservation_ShouldSucceed_WhenSpotIsAvailable verifies the baseline
// happy path: when a spot is available, reservation proceeds normally. This
// complements the rejection test by confirming the positive case.
func TestCreateReservation_ShouldSucceed_WhenSpotIsAvailable(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	repo.On("FindByIdempotencyKey", mock.Anything, "diff-spot-key").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-a",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	redis.On("SetNX", mock.Anything, "lock:spot:spot-a", "locked", 30*time.Second).Return(true, nil)
	redis.On("Delete", mock.Anything, "lock:spot:spot-a").Return(nil)
	repo.On("GetSpotForUpdate", mock.Anything, "spot-a").Return(&model.ParkingSpot{
		ID:          "spot-a",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-a", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), billingmodel.BookingFee, mock.AnythingOfType("string")).Return(nil)
	natsClient.On("Publish", "reservation.confirmed", mock.Anything).Return(nil)

	uc := NewUsecase(repo, redis, natsClient, billing, payment)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "diff-spot-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.StatusConfirmed, result.Status)
	assert.Equal(t, "spot-a", result.SpotID)
	assert.Equal(t, "driver-1", result.DriverID)
	assert.Equal(t, model.AssignmentSystemAssigned, result.AssignmentMode)
	assert.NotNil(t, result.ConfirmedAt)
	assert.NotNil(t, result.ExpiresAt)
	repo.AssertExpectations(t)
	redis.AssertExpectations(t)
	natsClient.AssertExpectations(t)
	billing.AssertExpectations(t)
}

// TestCreateReservation_ShouldSucceed_WhenMotorcycleSpotIsAvailable verifies
// that a motorcycle-type reservation succeeds when an available motorcycle
// spot is found. This ensures vehicle-type filtering in FindAvailableSpot
// works correctly for the overlap-detection context.
func TestCreateReservation_ShouldSucceed_WhenMotorcycleSpotIsAvailable(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	repo.On("FindByIdempotencyKey", mock.Anything, "boundary-key").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "motorcycle").Return(&model.ParkingSpot{
		ID:          "spot-boundary",
		VehicleType: "motorcycle",
		Status:      "available",
	}, nil)
	redis.On("SetNX", mock.Anything, "lock:spot:spot-boundary", "locked", 30*time.Second).Return(true, nil)
	redis.On("Delete", mock.Anything, "lock:spot:spot-boundary").Return(nil)
	repo.On("GetSpotForUpdate", mock.Anything, "spot-boundary").Return(&model.ParkingSpot{
		ID:          "spot-boundary",
		VehicleType: "motorcycle",
		Status:      "available",
	}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-boundary", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), billingmodel.BookingFee, mock.AnythingOfType("string")).Return(nil)
	natsClient.On("Publish", "reservation.confirmed", mock.Anything).Return(nil)

	uc := NewUsecase(repo, redis, natsClient, billing, payment)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "motorcycle",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "boundary-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.StatusConfirmed, result.Status)
	assert.Equal(t, "spot-boundary", result.SpotID)
	assert.Equal(t, "motorcycle", result.VehicleType)
	assert.Equal(t, model.AssignmentSystemAssigned, result.AssignmentMode)
	assert.NotNil(t, result.ConfirmedAt)
	assert.NotNil(t, result.ExpiresAt)
	repo.AssertExpectations(t)
	redis.AssertExpectations(t)
	natsClient.AssertExpectations(t)
	billing.AssertExpectations(t)
}
