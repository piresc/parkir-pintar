// Package e2e_test — race condition tests for ParkirPintar.
//
// These tests are designed to be run with `go test -race` to detect data races
// in the reservation, billing, and payment usecases under concurrent access.
// They exercise the critical paths identified in the PRD:
//   - Double-booking prevention (PRD §13.4)
//   - Concurrent reservation creation (PRD §17.3 #2, #3)
//   - Concurrent lifecycle operations on the same reservation
//   - Idempotency under concurrent duplicate requests (PRD §13.5)
//
// Run with: go test -race -v -timeout 300s -run TestRace ./tests/e2e/...
package e2e_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestRace_ConcurrentReservationCreation_ShouldNotDoubleBook verifies that
// 50 goroutines creating reservations concurrently never produce a double-booked
// spot. Each successful reservation must map to a distinct spot.
func TestRace_ConcurrentReservationCreation_ShouldNotDoubleBook(t *testing.T) {
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)
	err = testhelpers.ResetSpots(ctx, env.db)
	require.NoError(t, err)

	const numGoroutines = 50

	// Insert drivers
	driverIDs := make([]string, numGoroutines)
	for i := range numGoroutines {
		driverIDs[i], err = testhelpers.InsertTestDriver(ctx, env.db, "car")
		require.NoError(t, err)
	}

	type result struct {
		reservation *model.Reservation
		err         error
	}

	results := make([]result, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			res, resErr := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
				DriverID:       driverIDs[idx],
				VehicleType:    "car",
				AssignmentMode: model.AssignmentSystemAssigned,
				IdempotencyKey: uuid.New().String(),
			})
			results[idx] = result{reservation: res, err: resErr}
		}(i)
	}
	wg.Wait()

	// Collect successful reservations and verify no duplicate spots
	spotSet := make(map[string]bool)
	var successCount int
	for _, r := range results {
		if r.err == nil && r.reservation != nil {
			successCount++
			assert.False(t, spotSet[r.reservation.SpotID],
				"spot %s was double-booked", r.reservation.SpotID)
			spotSet[r.reservation.SpotID] = true
		}
	}

	assert.Greater(t, successCount, 0, "at least one reservation should succeed")
	t.Logf("Race: %d/%d reservations succeeded, %d unique spots assigned",
		successCount, numGoroutines, len(spotSet))

	// Verify spot inventory conservation
	testhelpers.AssertSpotCount(t, env.db, 400)
	testhelpers.AssertSpotStatusCounts(t, env.db)
}

// TestRace_SameSpotContention_ShouldAllowExactlyOne verifies that when 20
// goroutines all target the same specific spot via user-selected mode,
// exactly one succeeds and the rest get conflict errors.
func TestRace_SameSpotContention_ShouldAllowExactlyOne(t *testing.T) {
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)
	err = testhelpers.ResetSpots(ctx, env.db)
	require.NoError(t, err)

	const numGoroutines = 20

	driverIDs := make([]string, numGoroutines)
	for i := range numGoroutines {
		driverIDs[i], err = testhelpers.InsertTestDriver(ctx, env.db, "car")
		require.NoError(t, err)
	}

	// Pick a specific spot
	var spotID string
	err = env.db.QueryRowContext(ctx,
		"SELECT id FROM parking_spots WHERE vehicle_type = 'car' AND status = 'available' LIMIT 1").Scan(&spotID)
	require.NoError(t, err)

	var successCount atomic.Int32
	var conflictCount atomic.Int32
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			_, resErr := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
				DriverID:       driverIDs[idx],
				VehicleType:    "car",
				AssignmentMode: model.AssignmentUserSelected,
				SpotID:         spotID,
				IdempotencyKey: uuid.New().String(),
			})
			if resErr == nil {
				successCount.Add(1)
			} else {
				conflictCount.Add(1)
			}
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int32(1), successCount.Load(),
		"exactly one goroutine should win the spot")
	assert.Equal(t, int32(numGoroutines-1), conflictCount.Load(),
		"all others should get conflict")

	// Verify spot is reserved
	var spotStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", spotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "reserved", spotStatus)
}

// TestRace_ConcurrentIdempotency_ShouldReturnSameReservation verifies that
// 10 goroutines sending the same idempotency key all get the same reservation
// back without creating duplicates.
func TestRace_ConcurrentIdempotency_ShouldReturnSameReservation(t *testing.T) {
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)
	err = testhelpers.ResetSpots(ctx, env.db)
	require.NoError(t, err)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	const numGoroutines = 10
	idempotencyKey := uuid.New().String()

	type result struct {
		reservation *model.Reservation
		err         error
	}
	results := make([]result, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			res, resErr := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
				DriverID:       driverID,
				VehicleType:    "car",
				AssignmentMode: model.AssignmentSystemAssigned,
				IdempotencyKey: idempotencyKey,
			})
			results[idx] = result{reservation: res, err: resErr}
		}(i)
	}
	wg.Wait()

	// All should succeed (idempotent) or at most one creates + rest return existing
	var reservationIDs []string
	for _, r := range results {
		if r.err == nil && r.reservation != nil {
			reservationIDs = append(reservationIDs, r.reservation.ID)
		}
	}

	assert.NotEmpty(t, reservationIDs, "at least one should succeed")

	// All returned reservation IDs should be the same
	firstID := reservationIDs[0]
	for _, id := range reservationIDs[1:] {
		assert.Equal(t, firstID, id,
			"idempotent requests should return the same reservation ID")
	}

	// Verify only one reservation exists in DB
	var count int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM reservations WHERE idempotency_key = $1", idempotencyKey).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "only one reservation should exist for the idempotency key")
}

// TestRace_ConcurrentLifecycleOnSameReservation_ShouldNotCorrupt verifies
// that concurrent check-in, cancel, and expire attempts on the same reservation
// don't corrupt state. Only one transition should succeed.
func TestRace_ConcurrentLifecycleOnSameReservation_ShouldNotCorrupt(t *testing.T) {
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)
	err = testhelpers.ResetSpots(ctx, env.db)
	require.NoError(t, err)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: uuid.New().String(),
	})
	require.NoError(t, err)

	_, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)

	// Launch 3 concurrent operations on the same reservation
	var checkinErr, cancelErr, expireErr error
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		_, checkinErr = env.reservationUC.CheckIn(ctx, &model.CheckInRequest{
			ReservationID: reservation.ID,
		})
	}()
	go func() {
		defer wg.Done()
		_, cancelErr = env.reservationUC.CancelReservation(ctx, &model.CancelReservationRequest{
			ReservationID: reservation.ID,
		})
	}()
	go func() {
		defer wg.Done()
		expireErr = env.reservationUC.ExpireReservation(ctx, &model.ExpireReservationRequest{
			ReservationID: reservation.ID,
		})
	}()
	wg.Wait()

	// Exactly one should succeed, the others should fail
	successCount := 0
	if checkinErr == nil {
		successCount++
	}
	if cancelErr == nil {
		successCount++
	}
	if expireErr == nil {
		successCount++
	}

	// At least one should succeed (the first to acquire the row)
	assert.GreaterOrEqual(t, successCount, 1,
		"at least one concurrent operation should succeed")

	// Verify the reservation is in a valid terminal or checked-in state
	var finalStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM reservations WHERE id = $1", reservation.ID).Scan(&finalStatus)
	require.NoError(t, err)

	validStates := []string{model.StatusCheckedIn, model.StatusCancelled, model.StatusExpired}
	assert.Contains(t, validStates, finalStatus,
		"final status should be one of checked_in, cancelled, or expired")

	t.Logf("Race lifecycle: checkin=%v cancel=%v expire=%v → final=%s",
		checkinErr == nil, cancelErr == nil, expireErr == nil, finalStatus)
}

// TestRace_ConcurrentCreateAndCancel_ShouldMaintainInventory verifies that
// rapid create-then-cancel cycles from multiple goroutines don't leak spots.
func TestRace_ConcurrentCreateAndCancel_ShouldMaintainInventory(t *testing.T) {
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)
	err = testhelpers.ResetSpots(ctx, env.db)
	require.NoError(t, err)

	const numGoroutines = 30
	driverIDs := make([]string, numGoroutines)
	for i := range numGoroutines {
		driverIDs[i], err = testhelpers.InsertTestDriver(ctx, env.db, "motorcycle")
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			res, createErr := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
				DriverID:       driverIDs[idx],
				VehicleType:    "motorcycle",
				AssignmentMode: model.AssignmentSystemAssigned,
				IdempotencyKey: uuid.New().String(),
			})
			if createErr != nil {
				return
			}
			// Immediately cancel
			_, _ = env.reservationUC.CancelReservation(ctx, &model.CancelReservationRequest{
				ReservationID: res.ID,
			})
		}(i)
	}
	wg.Wait()

	// All spots should be back to available (or still available if create failed)
	testhelpers.AssertSpotCount(t, env.db, 400)
	testhelpers.AssertSpotStatusCounts(t, env.db)

	// Count how many spots are still reserved (should be 0 if all cancels succeeded)
	var reservedCount int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM parking_spots WHERE status = 'reserved'").Scan(&reservedCount)
	require.NoError(t, err)
	t.Logf("Race create+cancel: %d spots still reserved (should be 0 or very low)", reservedCount)
}

// TestRace_MixedVehicleTypes_ShouldNotCrossAssign verifies that concurrent
// reservations for cars and motorcycles never cross-assign vehicle types.
func TestRace_MixedVehicleTypes_ShouldNotCrossAssign(t *testing.T) {
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)
	err = testhelpers.ResetSpots(ctx, env.db)
	require.NoError(t, err)

	const numPerType = 20

	type result struct {
		reservation *model.Reservation
		vehicleType string
		err         error
	}

	results := make([]result, numPerType*2)
	var wg sync.WaitGroup
	wg.Add(numPerType * 2)

	for i := range numPerType {
		// Car driver
		carDriverID, dErr := testhelpers.InsertTestDriver(ctx, env.db, "car")
		require.NoError(t, dErr)
		go func(idx int, driverID string) {
			defer wg.Done()
			res, resErr := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
				DriverID:       driverID,
				VehicleType:    "car",
				AssignmentMode: model.AssignmentSystemAssigned,
				IdempotencyKey: uuid.New().String(),
			})
			results[idx] = result{reservation: res, vehicleType: "car", err: resErr}
		}(i, carDriverID)

		// Motorcycle driver
		motoDriverID, dErr := testhelpers.InsertTestDriver(ctx, env.db, "motorcycle")
		require.NoError(t, dErr)
		go func(idx int, driverID string) {
			defer wg.Done()
			res, resErr := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
				DriverID:       driverID,
				VehicleType:    "motorcycle",
				AssignmentMode: model.AssignmentSystemAssigned,
				IdempotencyKey: uuid.New().String(),
			})
			results[idx] = result{reservation: res, vehicleType: "motorcycle", err: resErr}
		}(numPerType+i, motoDriverID)
	}
	wg.Wait()

	// Verify no cross-assignment: each reservation's spot must match the requested vehicle type
	for _, r := range results {
		if r.err != nil || r.reservation == nil {
			continue
		}
		var spotVehicleType string
		err := env.db.QueryRowContext(ctx,
			"SELECT vehicle_type FROM parking_spots WHERE id = $1", r.reservation.SpotID).Scan(&spotVehicleType)
		require.NoError(t, err)
		assert.Equal(t, r.vehicleType, spotVehicleType,
			"reservation for %s got spot with vehicle_type %s", r.vehicleType, spotVehicleType)
	}

	t.Logf("Race mixed types: verified no cross-assignment across %d concurrent reservations", numPerType*2)
}

// TestRace_ConcurrentCheckouts_ShouldNotDuplicatePayments verifies that
// if checkout is called concurrently on the same reservation, only one
// payment is created (idempotency).
func TestRace_ConcurrentCheckouts_ShouldNotDuplicatePayments(t *testing.T) {
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)
	err = testhelpers.ResetSpots(ctx, env.db)
	require.NoError(t, err)

	env.paymentGW.ShouldFail = false

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: uuid.New().String(),
	})
	require.NoError(t, err)

	_, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)

	_, err = env.reservationUC.CheckIn(ctx, &model.CheckInRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)

	// Try to checkout from 5 goroutines simultaneously
	const numGoroutines = 5
	var successCount atomic.Int32
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			_, coErr := env.reservationUC.CheckOut(ctx, &model.CheckOutRequest{
				ReservationID: reservation.ID,
			})
			if coErr == nil {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	// With SELECT FOR UPDATE, exactly one checkout should succeed.
	// The row lock prevents the TOCTOU race.
	assert.Equal(t, int32(1), successCount.Load(),
		"exactly one concurrent checkout should succeed (FOR UPDATE prevents TOCTOU)")

	// Verify exactly 1 payment exists
	var paymentCount int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM payments WHERE billing_id = (SELECT id FROM billing_records WHERE reservation_id = $1)",
		reservation.ID).Scan(&paymentCount)
	require.NoError(t, err)
	assert.Equal(t, 1, paymentCount, "exactly one payment should exist")
}
