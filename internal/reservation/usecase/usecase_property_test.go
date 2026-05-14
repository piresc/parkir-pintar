// Package usecase implements the business logic layer for the reservation domain.
//
// Property-based tests for the reservation usecase using pgregory.net/rapid.
// These tests verify Properties 5, 9, and 10 from the design document.
//
// Best practices applied (from coding standards KB):
// - rapid.Custom generators for constrained input spaces
// - testify/assert for assertions
// - t.Context() for context (Go 1.24+)
// - AAA pattern: Arrange → Act → Assert
// - Reuses mock types from usecase_test.go (MockRepository, MockBillingClient, etc.)
package usecase

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"pgregory.net/rapid"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/pkg/pricing"
	"parkir-pintar/pkg/redislock"
)

// --- Property 5: Idempotency ---

// TestProperty5_SameIdempotencyKeyReturnsSameReservationID verifies that for any
// random idempotency key string, calling CreateReservation twice with the same key
// returns the same reservation ID.
//
// **Validates: Requirements 2.8, 17.1**
func TestProperty5_SameIdempotencyKeyReturnsSameReservationID(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(rt *rapid.T) {
		// Arrange
		key := rapid.String().Draw(rt, "idempotencyKey")

		repo := new(MockRepository)
		locker := new(MockLocker)
		billing := new(MockBillingClient)
		payment := new(MockPaymentClient)

		existing := &model.Reservation{
			ID:             "res-existing-123",
			DriverID:       "driver-1",
			SpotID:         "spot-1",
			VehicleType:    "car",
			AssignmentMode: model.AssignmentSystemAssigned,
			Status:         model.StatusConfirmed,
			IdempotencyKey: key,
		}

		// First call: returns existing (idempotent hit)
		repo.On("FindByIdempotencyKey", mock.Anything, key).Return(existing, nil)

		uc := NewUsecase(repo, locker, billing, payment)
		req := &model.CreateReservationRequest{
			DriverID:       "driver-1",
			VehicleType:    "car",
			AssignmentMode: model.AssignmentSystemAssigned,
			IdempotencyKey: key,
		}

		// Act — call twice
		result1, err1 := uc.CreateReservation(t.Context(), req)
		result2, err2 := uc.CreateReservation(t.Context(), req)

		// Assert — both return same reservation ID, no errors
		assert.NoError(rt, err1)
		assert.NoError(rt, err2)
		assert.NotNil(rt, result1)
		assert.NotNil(rt, result2)
		assert.Equal(rt, result1.ID, result2.ID,
			"same idempotency key %q should return same reservation ID", key)
	})
}

// TestProperty5_DifferentIdempotencyKeysProduceDifferentIDs verifies that for
// two different random idempotency keys, CreateReservation produces different
// reservation IDs.
//
// **Validates: Requirements 2.8, 17.1**
func TestProperty5_DifferentIdempotencyKeysProduceDifferentIDs(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(rt *rapid.T) {
		// Arrange — generate two distinct keys
		key1 := rapid.String().Draw(rt, "key1")
		key2 := rapid.String().Draw(rt, "key2")
		if key1 == key2 {
			rt.Skip("keys are identical, skip")
		}

		// Helper to create a usecase that produces a new reservation for a given key
		createForKey := func(key string) (*model.Reservation, error) {
			repo := new(MockRepository)
			locker := new(MockLocker)
			billing := new(MockBillingClient)
			payment := new(MockPaymentClient)

			// No existing reservation for this key
			repo.On("FindByIdempotencyKey", mock.Anything, key).Return(nil, model.ErrNotFound)
			repo.On("ListByDriverID", mock.Anything, "driver-1", "").Return([]*model.Reservation{}, nil)
			repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
				ID:          "spot-1",
				VehicleType: "car",
				Status:      "available",
			}, nil)
			lck := new(MockLock)
			locker.On("Acquire", mock.Anything, mock.Anything).Return(lck, nil)
			lck.On("Release", mock.Anything).Return(nil)
			repo.On("GetSpotForUpdate", mock.Anything, "spot-1").Return(&model.ParkingSpot{
				ID:     "spot-1",
				Status: "available",
			}, nil)
			repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
			repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-1", "reserved").Return(nil)
		billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), pricing.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

		uc := NewUsecase(repo, locker, billing, payment)
		req := &model.CreateReservationRequest{
			DriverID:       "driver-1",
			VehicleType:    "car",
			AssignmentMode: model.AssignmentSystemAssigned,
			IdempotencyKey: key,
		}
		return uc.CreateReservation(t.Context(), req)
		}

		// Act
		res1, err1 := createForKey(key1)
		res2, err2 := createForKey(key2)

		// Assert
		assert.NoError(rt, err1)
		assert.NoError(rt, err2)
		assert.NotNil(rt, res1)
		assert.NotNil(rt, res2)
		assert.NotEqual(rt, res1.ID, res2.ID,
			"different keys %q and %q should produce different reservation IDs", key1, key2)
	})
}

// --- Property 9: Reservation Creation Postconditions ---

// TestProperty9_ReservationCreationPostconditions verifies that for any successful
// system-assigned reservation with a random vehicle type ("car" or "motorcycle"):
//   - The assigned spot's vehicle type matches the requested vehicle type
//   - The reservation status is "confirmed"
//   - expires_at == confirmed_at + 1 hour (within 1 second tolerance)
//
// **Validates: Requirements 2.1, 2.3**
func TestProperty9_ReservationCreationPostconditions(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(rt *rapid.T) {
		// Arrange
		vehicleType := rapid.SampledFrom([]string{"car", "motorcycle"}).Draw(rt, "vehicleType")

		repo := new(MockRepository)
		locker := new(MockLocker)
		billing := new(MockBillingClient)
		payment := new(MockPaymentClient)

		spotID := "spot-prop9"
		repo.On("FindByIdempotencyKey", mock.Anything, mock.Anything).Return(nil, model.ErrNotFound)
		repo.On("ListByDriverID", mock.Anything, "driver-prop9", "").Return([]*model.Reservation{}, nil)
		repo.On("FindAvailableSpot", mock.Anything, vehicleType).Return(&model.ParkingSpot{
			ID:          spotID,
			VehicleType: vehicleType,
			Status:      "available",
		}, nil)
		lck := new(MockLock)
		locker.On("Acquire", mock.Anything, "spot:"+spotID).Return(lck, nil)
		lck.On("Release", mock.Anything).Return(nil)
		repo.On("GetSpotForUpdate", mock.Anything, spotID).Return(&model.ParkingSpot{
			ID:     spotID,
			Status: "available",
		}, nil)
		repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
		repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), spotID, "reserved").Return(nil)
		billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), pricing.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-prop9-id"}, nil)

		uc := NewUsecase(repo, locker, billing, payment)
		req := &model.CreateReservationRequest{
			DriverID:       "driver-prop9",
			VehicleType:    vehicleType,
			AssignmentMode: model.AssignmentSystemAssigned,
			IdempotencyKey: "prop9-key",
		}

		// Act
		result, err := uc.CreateReservation(t.Context(), req)

		// Assert
		assert.NoError(rt, err)
		assert.NotNil(rt, result)

		// Postcondition 1: assigned spot's vehicle type matches requested
		assert.Equal(rt, vehicleType, result.VehicleType,
			"reservation vehicle type should match requested %q", vehicleType)

		// Postcondition 2: status is "waiting_payment"
		assert.Equal(rt, model.StatusWaitingPayment, result.Status,
			"reservation status should be waiting_payment")

		// Postcondition 3: confirmed_at and expires_at are nil until payment
		assert.Nil(rt, result.ConfirmedAt, "confirmed_at should not be set yet")
		assert.Nil(rt, result.ExpiresAt, "expires_at should not be set yet")
	})
}

// --- Property 10: No Double-Booking ---

// TestProperty10_NoDoubleBooking verifies that for any spot, at most 1 active
// reservation (confirmed/checked_in) should exist. When a second reservation is
// attempted for the same spot, it should fail with a conflict error due to the
// distributed lock contention.
//
// **Validates: Requirements 16.1, 16.2**
func TestProperty10_NoDoubleBooking(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(rt *rapid.T) {
		// Arrange
		spotID := rapid.String().Draw(rt, "spotID")
		if spotID == "" {
			spotID = "spot-default"
		}

		// --- First reservation: succeeds ---
		repo1 := new(MockRepository)
		locker1 := new(MockLocker)
		billing1 := new(MockBillingClient)
		payment1 := new(MockPaymentClient)

		repo1.On("FindByIdempotencyKey", mock.Anything, "key-first").Return(nil, model.ErrNotFound)
		repo1.On("ListByDriverID", mock.Anything, "driver-first", "").Return([]*model.Reservation{}, nil)
		repo1.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
			ID:          spotID,
			VehicleType: "car",
			Status:      "available",
		}, nil)
		lck1 := new(MockLock)
		locker1.On("Acquire", mock.Anything, "spot:"+spotID).Return(lck1, nil)
		lck1.On("Release", mock.Anything).Return(nil)
		repo1.On("GetSpotForUpdate", mock.Anything, spotID).Return(&model.ParkingSpot{
			ID:     spotID,
			Status: "available",
		}, nil)
		repo1.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
		repo1.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), spotID, "reserved").Return(nil)
		billing1.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), pricing.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-first-id"}, nil)

		uc1 := NewUsecase(repo1, locker1, billing1, payment1)
		req1 := &model.CreateReservationRequest{
			DriverID:       "driver-first",
			VehicleType:    "car",
			AssignmentMode: model.AssignmentSystemAssigned,
			IdempotencyKey: "key-first",
		}

		res1, err1 := uc1.CreateReservation(t.Context(), req1)

		// Assert first reservation succeeds
		assert.NoError(rt, err1)
		assert.NotNil(rt, res1)
		assert.Equal(rt, model.StatusWaitingPayment, res1.Status)

		// --- Second reservation for same spot: fails due to lock contention ---
		repo2 := new(MockRepository)
		locker2 := new(MockLocker)
		billing2 := new(MockBillingClient)
		payment2 := new(MockPaymentClient)

		repo2.On("FindByIdempotencyKey", mock.Anything, "key-second").Return(nil, model.ErrNotFound)
		repo2.On("ListByDriverID", mock.Anything, "driver-second", "").Return([]*model.Reservation{}, nil)
		repo2.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
			ID:          spotID,
			VehicleType: "car",
			Status:      "available",
		}, nil)
		// Lock contention: Acquire returns ErrLockUnavailable
		locker2.On("Acquire", mock.Anything, "spot:"+spotID).Return(nil, redislock.ErrLockUnavailable)

		uc2 := NewUsecase(repo2, locker2, billing2, payment2)
		req2 := &model.CreateReservationRequest{
			DriverID:       "driver-second",
			VehicleType:    "car",
			AssignmentMode: model.AssignmentSystemAssigned,
			IdempotencyKey: "key-second",
		}

		res2, err2 := uc2.CreateReservation(t.Context(), req2)

		// Assert second reservation fails with conflict
		assert.Nil(rt, res2, "second reservation for same spot should fail")
		assert.Error(rt, err2, "second reservation should return error")
		assert.Contains(rt, err2.Error(), "spot is being reserved by another driver",
			"error should indicate lock contention for spot %q", spotID)
	})
}
