// Package e2e_test — user-selected spot contention integration test.
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Use assert for non-critical checks
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
// - Do not mock the database — these are integration tests against real PostgreSQL
package e2e_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestContention_ShouldSerializeAccess_WhenMultipleDriversSelectSameSpot
// verifies that when multiple drivers concurrently attempt to reserve the
// same spot, exactly one succeeds and the rest receive conflict errors.
//
// Validates: Requirements 7.1, 7.2, 7.3
func TestContention_ShouldSerializeAccess_WhenMultipleDriversSelectSameSpot(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db, "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	const numDrivers = 5
	driverIDs := make([]string, numDrivers)
	for i := range numDrivers {
		driverIDs[i], err = testhelpers.InsertTestDriver(ctx, env.db, "car")
		require.NoError(t, err)
	}

	// Get an available spot ID
	var spotID string
	err = env.db.QueryRowContext(ctx,
		"SELECT id FROM parking_spots WHERE vehicle_type = 'car' AND status = 'available' LIMIT 1").Scan(&spotID)
	require.NoError(t, err)

	// Act — All drivers attempt to reserve the same spot concurrently
	type result struct {
		reservation *model.Reservation
		err         error
	}
	results := make([]result, numDrivers)
	var wg sync.WaitGroup
	wg.Add(numDrivers)

	for i := range numDrivers {
		go func(idx int) {
			defer wg.Done()
			res, resErr := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
				DriverID:       driverIDs[idx],
				VehicleType:    "car",
				AssignmentMode: model.AssignmentUserSelected,
				SpotID:         spotID,
				IdempotencyKey: uuid.New().String(),
			})
			results[idx] = result{reservation: res, err: resErr}
		}(i)
	}
	wg.Wait()

	// Assert — Exactly one succeeds, others get conflict
	successCount := 0
	conflictCount := 0
	for _, r := range results {
		if r.err == nil && r.reservation != nil {
			successCount++
		} else {
			conflictCount++
		}
	}

	assert.Equal(t, 1, successCount, "exactly one driver should succeed")
	assert.Equal(t, numDrivers-1, conflictCount, "remaining drivers should get conflict")

	// Verify spot is "reserved"
	var spotStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", spotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "reserved", spotStatus)
}
