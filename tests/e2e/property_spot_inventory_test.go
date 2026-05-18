// Package e2e_test — Property 1: Spot Inventory Conservation.
//
// Uses pgregory.net/rapid to generate random sequences of reservation lifecycle
// operations against real PostgreSQL, then asserts the spot inventory invariant.
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Follow AAA (Arrange-Act-Assert) structure
// - Do not mock the database — these are integration tests against real PostgreSQL
package e2e_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestProperty1_SpotInventoryConservation verifies that for any sequence of
// reservation lifecycle operations (create, cancel, expire), the total count
// of parking_spots remains 400 and available+reserved+occupied == 400.
//
// **Validates: Requirements 2.2, 5.6, 5.7, 8.3, 10.3, 11.3**
func TestProperty1_SpotInventoryConservation(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctx := context.Background()

		// Clean tables for isolation
		err := testhelpers.TruncateTables(ctx, env.db,
			"payments", "billing_records", "reservations", "drivers")
		require.NoError(t, err)

		// Insert a test driver
		driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
		require.NoError(t, err)

		// Track active reservation IDs for cancel/expire operations
		var activeIDs []string

		// Generate random sequence of operations
		numOps := rapid.IntRange(1, 10).Draw(rt, "numOps")

		for i := 0; i < numOps; i++ {
			op := rapid.SampledFrom([]string{"create", "cancel", "expire"}).Draw(rt, "op")

			switch op {
			case "create":
				vt := rapid.SampledFrom([]string{"car", "motorcycle"}).Draw(rt, "vehicleType")
				res, createErr := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
					DriverID:       driverID,
					VehicleType:    vt,
					AssignmentMode: model.AssignmentSystemAssigned,
					IdempotencyKey: uuid.New().String(),
				})
				if createErr == nil && res != nil {
					activeIDs = append(activeIDs, res.ID)
				}

			case "cancel":
				if len(activeIDs) > 0 {
					idx := rapid.IntRange(0, len(activeIDs)-1).Draw(rt, "cancelIdx")
					_, _ = env.reservationUC.CancelReservation(ctx, &model.CancelReservationRequest{
						ReservationID: activeIDs[idx],
					})
				}

			case "expire":
				if len(activeIDs) > 0 {
					idx := rapid.IntRange(0, len(activeIDs)-1).Draw(rt, "expireIdx")
					_ = env.reservationUC.ExpireReservation(ctx, &model.ExpireReservationRequest{
						ReservationID: activeIDs[idx],
					})
				}
			}
		}

		// Assert invariants after the sequence
		testhelpers.AssertSpotCount(t, env.db, 400)
		testhelpers.AssertSpotStatusCounts(t, env.db)
	})
}
