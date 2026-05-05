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
// booking_fee + parking_fee + overnight_fee + cancellation_fee + penalty_amount.
//
// Validates: Requirement 5.4 — billing total consistency.
func AssertBillingTotal(t *testing.T, db *sqlx.DB, reservationID string) {
	t.Helper()

	var billing struct {
		BookingFee      int64 `db:"booking_fee"`
		ParkingFee      int64 `db:"parking_fee"`
		OvernightFee    int64 `db:"overnight_fee"`
		CancellationFee int64 `db:"cancellation_fee"`
		PenaltyAmount   int64 `db:"penalty_amount"`
		TotalAmount     int64 `db:"total_amount"`
	}

	err := db.GetContext(context.Background(), &billing,
		`SELECT booking_fee, parking_fee, overnight_fee, cancellation_fee, penalty_amount, total_amount
		 FROM billing_records WHERE reservation_id = $1`, reservationID)
	require.NoError(t, err, "failed to query billing_records for reservation %s", reservationID)

	expectedTotal := billing.BookingFee + billing.ParkingFee + billing.OvernightFee +
		billing.CancellationFee + billing.PenaltyAmount
	require.Equal(t, expectedTotal, billing.TotalAmount,
		"billing total mismatch for reservation %s: booking=%d + parking=%d + overnight=%d + cancellation=%d + penalty=%d = %d, but total_amount=%d",
		reservationID, billing.BookingFee, billing.ParkingFee, billing.OvernightFee,
		billing.CancellationFee, billing.PenaltyAmount, expectedTotal, billing.TotalAmount)
}

// AssertNoPenalty verifies that no penalty records exist in the penalties table
// for the given reservation ID.
//
// Validates: Requirement 10.4 — no penalty for free cancellation.
func AssertNoPenalty(t *testing.T, db *sqlx.DB, reservationID string) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM penalties WHERE reservation_id = $1", reservationID).Scan(&count)
	require.NoError(t, err, "failed to query penalties for reservation %s", reservationID)
	require.Equal(t, 0, count,
		"expected no penalties for reservation %s, but found %d", reservationID, count)
}

// AssertPenaltyExists verifies that a penalty record exists in the penalties
// table for the given reservation with the specified penalty_type and amount.
//
// Validates: Requirement 9.3 — wrong-spot penalty exists with correct type and amount.
func AssertPenaltyExists(t *testing.T, db *sqlx.DB, reservationID, penaltyType string, amount int64) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM penalties
		 WHERE reservation_id = $1 AND penalty_type = $2 AND amount = $3`,
		reservationID, penaltyType, amount).Scan(&count)
	require.NoError(t, err,
		"failed to query penalties for reservation %s, type %s", reservationID, penaltyType)
	require.Greater(t, count, 0,
		"expected penalty record for reservation %s with type=%s and amount=%d, but found none",
		reservationID, penaltyType, amount)
}

// AssertNoPenaltyExists verifies that no penalty record exists in the penalties
// table for the given reservation with the specified penalty_type.
//
// Validates: PRD — booking fee is the only no-show cost, no additional penalty.
func AssertNoPenaltyExists(t *testing.T, db *sqlx.DB, reservationID, penaltyType string) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM penalties
		 WHERE reservation_id = $1 AND penalty_type = $2`,
		reservationID, penaltyType).Scan(&count)
	require.NoError(t, err,
		"failed to query penalties for reservation %s, type %s", reservationID, penaltyType)
	require.Equal(t, 0, count,
		"expected no penalty record for reservation %s with type=%s, but found %d",
		reservationID, penaltyType, count)
}
