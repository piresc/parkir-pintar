package pricing

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: grpc-jwt-pkg-integration, Property 14: Parking fee calculation
// **Validates: Requirements 11.1, 11.9**
//
// For any valid check-in and check-out time pair where check-out is after
// check-in, the parking fee SHALL equal ceil(duration_in_hours) × 5,000 IDR,
// with a minimum of 5,000 IDR.
func TestProperty14_ParkingFeeCalculation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random check-in time within a reasonable range.
		baseUnix := rapid.Int64Range(0, 2_000_000_000).Draw(t, "checkInUnix")
		checkIn := time.Unix(baseUnix, 0).UTC()

		// Duration between 1 second and 72 hours.
		durationSec := rapid.Int64Range(1, 72*3600).Draw(t, "durationSec")
		checkOut := checkIn.Add(time.Duration(durationSec) * time.Second)

		session := ParkingSession{CheckIn: checkIn, CheckOut: checkOut}

		fee, billedHours, err := CalculateParkingFee(session)
		require.NoError(t, err)

		// Expected: ceil(duration_in_hours) × 5000, minimum 5000
		duration := checkOut.Sub(checkIn)
		expectedHours := math.Ceil(duration.Hours())
		if expectedHours < 1 {
			expectedHours = 1
		}
		expectedFee := int64(expectedHours) * 5000

		assert.Equal(t, expectedFee, fee, "fee must equal ceil(hours)*5000")
		assert.Equal(t, int(expectedHours), billedHours, "billedHours must equal ceil(hours)")
		assert.GreaterOrEqual(t, fee, int64(5000), "minimum fee is 5000 IDR")
	})
}

// Feature: grpc-jwt-pkg-integration, Property 15: Overnight fee determination
// **Validates: Requirements 11.2**
//
// For any parking session, the overnight fee SHALL be 20,000 IDR if the session
// crosses midnight in WIB (UTC+7), and 0 IDR otherwise.
func TestProperty15_OvernightFeeDetermination(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseUnix := rapid.Int64Range(0, 2_000_000_000).Draw(t, "checkInUnix")
		checkIn := time.Unix(baseUnix, 0).UTC()

		durationSec := rapid.Int64Range(1, 72*3600).Draw(t, "durationSec")
		checkOut := checkIn.Add(time.Duration(durationSec) * time.Second)

		session := ParkingSession{CheckIn: checkIn, CheckOut: checkOut}

		fee := OvernightFee(session)

		// Determine expected: different calendar dates in WIB means overnight.
		inWIB := checkIn.In(WIB)
		outWIB := checkOut.In(WIB)
		inY, inM, inD := inWIB.Date()
		outY, outM, outD := outWIB.Date()
		crossesMidnight := inY != outY || inM != outM || inD != outD

		if crossesMidnight {
			assert.Equal(t, int64(20000), fee, "overnight fee must be 20000 IDR when session crosses midnight in WIB")
		} else {
			assert.Equal(t, int64(0), fee, "overnight fee must be 0 IDR when session does not cross midnight in WIB")
		}
	})
}

// Feature: grpc-jwt-pkg-integration, Property 16: Cancellation fee based on elapsed time
// **Validates: Requirements 11.4, 11.5**
//
// For any confirmation time and cancellation time, the cancellation fee SHALL
// be 0 IDR if the elapsed time is within 2 minutes, and 5,000 IDR if the
// elapsed time exceeds 2 minutes.
func TestProperty16_CancellationFee(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseUnix := rapid.Int64Range(0, 2_000_000_000).Draw(t, "confirmedAtUnix")
		confirmedAt := time.Unix(baseUnix, 0).UTC()

		// Elapsed time between 0 and 30 minutes to cover both branches well.
		elapsedSec := rapid.Int64Range(0, 30*60).Draw(t, "elapsedSec")
		cancelledAt := confirmedAt.Add(time.Duration(elapsedSec) * time.Second)

		input := CancellationInput{ConfirmedAt: confirmedAt, CancelledAt: cancelledAt}

		fee := CancellationFee(input)

		gracePeriod := 2 * time.Minute
		elapsed := cancelledAt.Sub(confirmedAt)

		if elapsed <= gracePeriod {
			assert.Equal(t, int64(0), fee, "cancellation within 2 min grace period must be 0 IDR")
		} else {
			assert.Equal(t, int64(5000), fee, "cancellation after 2 min must be 5000 IDR")
		}
	})
}

// Feature: grpc-jwt-pkg-integration, Property 17: Total amount is sum of components
// **Validates: Requirements 11.8**
//
// For any BillBreakdown with arbitrary non-negative fee components,
// CalculateTotal SHALL return a value equal to
// BookingFee + ParkingFee + OvernightFee + CancellationFee + PenaltyAmount.
func TestProperty17_TotalAmountIsSum(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		breakdown := BillBreakdown{
			BookingFee:      rapid.Int64Range(0, 1_000_000).Draw(t, "bookingFee"),
			ParkingFee:      rapid.Int64Range(0, 1_000_000).Draw(t, "parkingFee"),
			OvernightFee:    rapid.Int64Range(0, 1_000_000).Draw(t, "overnightFee"),
			CancellationFee: rapid.Int64Range(0, 1_000_000).Draw(t, "cancellationFee"),
			PenaltyAmount:   rapid.Int64Range(0, 1_000_000).Draw(t, "penaltyAmount"),
		}

		total := CalculateTotal(breakdown)

		expectedTotal := breakdown.BookingFee +
			breakdown.ParkingFee +
			breakdown.OvernightFee +
			breakdown.CancellationFee +
			breakdown.PenaltyAmount

		assert.Equal(t, expectedTotal, total, "CalculateTotal must equal sum of all fee components")
	})
}
