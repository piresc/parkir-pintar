// Package testhelpers provides shared utilities for both E2E test layers.
//
// Best practices applied (from Go testify testing standards):
// - Use t.Helper() to mark assertion helpers so failures report the caller's line
// - Use require (not assert) for assertions that must pass to continue
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive error messages in assertions
// - Keep helpers focused on a single responsibility
package testhelpers

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// AssertSpotCount verifies that the total number of rows in the parking_spots
// table equals the expected count.
//
// Validates: Requirement 2.2 — parking_spots contains exactly 400 rows after migration.
func AssertSpotCount(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM parking_spots").Scan(&count)
	require.NoError(t, err, "failed to query parking_spots count")
	require.Equal(t, expected, count, "parking_spots total count mismatch")
}

// AssertSpotStatusCounts verifies that the sum of spots across all statuses
// (available + reserved + occupied) equals 400. This ensures no spots are
// lost or duplicated during reservation lifecycle operations.
//
// Validates: Requirement 2.2 — spot inventory conservation invariant.
func AssertSpotStatusCounts(t *testing.T, db *sqlx.DB) {
	t.Helper()

	type statusCount struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}

	var counts []statusCount
	err := db.SelectContext(context.Background(), &counts,
		"SELECT status, COUNT(*) AS count FROM parking_spots GROUP BY status")
	require.NoError(t, err, "failed to query parking_spots status counts")

	total := 0
	for _, sc := range counts {
		total += sc.Count
	}
	require.Equal(t, 400, total,
		"sum of parking_spots across all statuses must equal 400, got status breakdown: %v", counts)
}

// AssertBillingTotal verifies that the billing_records.total_amount for a given
// reservation equals the sum of its individual fee fields:
// booking_fee + parking_fee + overnight_fee.
//
// Validates: Requirement 5.4 — billing total consistency.
func AssertBillingTotal(t *testing.T, db *sqlx.DB, reservationID string) {
	t.Helper()

	var billing struct {
		BookingFee   int64 `db:"booking_fee"`
		ParkingFee   int64 `db:"parking_fee"`
		OvernightFee int64 `db:"overnight_fee"`
		TotalAmount  int64 `db:"total_amount"`
	}

	err := db.GetContext(context.Background(), &billing,
		`SELECT booking_fee, parking_fee, overnight_fee, total_amount
		 FROM billing_records WHERE reservation_id = $1`, reservationID)
	require.NoError(t, err, "failed to query billing_records for reservation %s", reservationID)

	expectedTotal := billing.BookingFee + billing.ParkingFee + billing.OvernightFee
	require.Equal(t, expectedTotal, billing.TotalAmount,
		"billing total mismatch for reservation %s: booking=%d + parking=%d + overnight=%d = %d, but total_amount=%d",
		reservationID, billing.BookingFee, billing.ParkingFee, billing.OvernightFee, expectedTotal, billing.TotalAmount)
}
