// Package model defines domain structs and request types for the billing module.
//
// Property-based tests for the billing pricing engine using pgregory.net/rapid.
// These tests verify Properties 1, 2, 3, and 6 from the design document.
//
// Best practices applied (from coding standards KB):
// - rapid.Custom generators for constrained time inputs in WIB timezone
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - t.Context() for context (Go 1.24+)
// - No mocks — tests exercise pure functions directly
package model

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"

	"parkir-pintar/pkg/pricing"
)

// wibLoc returns the WIB (UTC+7) timezone location.
func wibLoc() *time.Location {
	return time.FixedZone("WIB", 7*60*60)
}

// genCheckInCheckOut generates a random (checkIn, checkOut) pair in WIB timezone
// where checkIn is strictly before checkOut, with duration between 1 minute and 48 hours.
func genCheckInCheckOut() *rapid.Generator[[2]time.Time] {
	return rapid.Custom[[2]time.Time](func(t *rapid.T) [2]time.Time {
		loc := wibLoc()
		// Base time: random year 2020-2030, month, day, hour, minute
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day") // safe for all months
		hour := rapid.IntRange(0, 23).Draw(t, "hour")
		minute := rapid.IntRange(0, 59).Draw(t, "minute")

		checkIn := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)

		// Duration between 1 minute and 48 hours (in seconds)
		durationSec := rapid.Int64Range(60, 48*3600).Draw(t, "durationSec")
		checkOut := checkIn.Add(time.Duration(durationSec) * time.Second)

		return [2]time.Time{checkIn, checkOut}
	})
}

// genSameDaySession generates a (checkIn, checkOut) pair guaranteed to be on the same
// calendar day in WIB, so no midnight crossing occurs.
func genSameDaySession() *rapid.Generator[[2]time.Time] {
	return rapid.Custom[[2]time.Time](func(t *rapid.T) [2]time.Time {
		loc := wibLoc()
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		// Start hour 0-22 so there's room for at least 1 minute on the same day
		startHour := rapid.IntRange(0, 22).Draw(t, "startHour")
		startMin := rapid.IntRange(0, 59).Draw(t, "startMin")

		checkIn := time.Date(year, time.Month(month), day, startHour, startMin, 0, 0, loc)

		// End of day in seconds from checkIn
		endOfDay := time.Date(year, time.Month(month), day, 23, 59, 59, 0, loc)
		maxDuration := endOfDay.Sub(checkIn)
		if maxDuration < time.Minute {
			// If less than 1 minute to end of day, just use 1 minute
			return [2]time.Time{checkIn, checkIn.Add(time.Minute)}
		}

		durationSec := rapid.Int64Range(60, int64(maxDuration.Seconds())).Draw(t, "durationSec")
		checkOut := checkIn.Add(time.Duration(durationSec) * time.Second)

		return [2]time.Time{checkIn, checkOut}
	})
}

// genCrossDaySession generates a (checkIn, checkOut) pair guaranteed to cross midnight
// in WIB timezone.
func genCrossDaySession() *rapid.Generator[[2]time.Time] {
	return rapid.Custom[[2]time.Time](func(t *rapid.T) [2]time.Time {
		loc := wibLoc()
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 27).Draw(t, "day") // leave room for next day
		// Start in the evening so we cross midnight
		startHour := rapid.IntRange(0, 23).Draw(t, "startHour")
		startMin := rapid.IntRange(0, 59).Draw(t, "startMin")

		checkIn := time.Date(year, time.Month(month), day, startHour, startMin, 0, 0, loc)

		// checkOut is on the next calendar day (or later)
		nextDay := time.Date(year, time.Month(month), day+1, 0, 0, 0, 0, loc)
		// Add 1 minute to 24 hours past midnight
		minAfterMidnight := nextDay.Sub(checkIn)
		if minAfterMidnight < time.Minute {
			minAfterMidnight = time.Minute
		}
		maxExtra := int64(24 * 3600) // up to 24h after midnight
		extraSec := rapid.Int64Range(int64(minAfterMidnight.Seconds())+1, int64(minAfterMidnight.Seconds())+maxExtra).Draw(t, "extraSec")
		checkOut := checkIn.Add(time.Duration(extraSec) * time.Second)

		return [2]time.Time{checkIn, checkOut}
	})
}

// --- Property 1: Pricing Correctness ---

// TestProperty1_PricingCorrectness verifies that for any checkIn before checkOut
// (duration 1min to 48h):
//   - billedHours == ceil(duration_in_hours)
//   - billedHours >= 1
//   - parkingFee == billedHours * 5000
//   - durationMinutes == int(checkOut.Sub(checkIn).Minutes())
//
// **Validates: Requirements 8.1, 8.2, 8.5**
func TestProperty1_PricingCorrectness(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		times := genCheckInCheckOut().Draw(t, "session")
		checkIn, checkOut := times[0], times[1]

		// Act
		result := pricing.CalculateSessionFee(checkIn, checkOut)

		// Assert — billedHours == ceil(duration_in_hours)
		duration := checkOut.Sub(checkIn)
		expectedBilledHours := int(math.Ceil(duration.Hours()))
		if expectedBilledHours < 1 {
			expectedBilledHours = 1
		}
		assert.Equal(t, expectedBilledHours, result.BilledHours,
			"billedHours should be ceil(duration_in_hours) for duration %v", duration)

		// Assert — billedHours >= 1
		assert.GreaterOrEqual(t, result.BilledHours, 1,
			"billedHours must be at least 1")

		// Assert — parkingFee == billedHours * HourlyRate
		assert.Equal(t, int64(result.BilledHours)*pricing.HourlyRate, result.ParkingFee,
			"parkingFee should be billedHours * %d", pricing.HourlyRate)

		// Assert — durationMinutes == int(duration.Minutes())
		expectedMinutes := int(duration.Minutes())
		assert.Equal(t, expectedMinutes, result.DurationMinutes,
			"durationMinutes should match actual minutes")
	})
}

// --- Property 2: Overnight Detection ---

// TestProperty2_OvernightDetection_SameDay verifies that for any session that stays
// within the same calendar day in WIB, overnightFee is 0.
//
// **Validates: Requirements 8.3, 8.4**
func TestProperty2_OvernightDetection_SameDay(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		times := genSameDaySession().Draw(t, "sameDaySession")
		checkIn, checkOut := times[0], times[1]

		// Act
		result := pricing.CalculateSessionFee(checkIn, checkOut)

		// Assert
		assert.False(t, result.IsOvernight,
			"same-day session should not be overnight: checkIn=%v checkOut=%v", checkIn, checkOut)
		assert.Equal(t, int64(0), result.OvernightFee,
			"overnightFee should be 0 for same-day session")
	})
}

// TestProperty2_OvernightDetection_CrossDay verifies that for any session that crosses
// midnight in WIB, overnightFee > 0.
//
// **Validates: Requirements 8.3, 8.4**
func TestProperty2_OvernightDetection_CrossDay(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		times := genCrossDaySession().Draw(t, "crossDaySession")
		checkIn, checkOut := times[0], times[1]

		// Act
		result := pricing.CalculateSessionFee(checkIn, checkOut)

		// Assert
		assert.True(t, result.IsOvernight,
			"cross-day session should be overnight: checkIn=%v checkOut=%v", checkIn, checkOut)
		assert.Greater(t, result.OvernightFee, int64(0),
			"overnightFee should be > 0 for cross-day session")
	})
}

// --- Property 3: Cancellation Fee Rules ---

// genConfirmedCancelled generates a (confirmedAt, cancelledAt) pair in WIB timezone
// where cancelledAt is after confirmedAt, with elapsed time between 0s and 1 hour.
func genConfirmedCancelled() *rapid.Generator[[2]time.Time] {
	return rapid.Custom[[2]time.Time](func(t *rapid.T) [2]time.Time {
		loc := wibLoc()
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		hour := rapid.IntRange(0, 23).Draw(t, "hour")
		minute := rapid.IntRange(0, 59).Draw(t, "minute")

		confirmedAt := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)

		// Elapsed between 0 and 3600 seconds (1 hour)
		elapsedSec := rapid.Int64Range(0, 3600).Draw(t, "elapsedSec")
		cancelledAt := confirmedAt.Add(time.Duration(elapsedSec) * time.Second)

		return [2]time.Time{confirmedAt, cancelledAt}
	})
}

// TestProperty3_CancellationFeeRules verifies that for any confirmedAt before cancelledAt:
//   - fee == 0 if elapsed <= 2 minutes
//   - fee == 5000 if elapsed > 2 minutes
//   - fee is always either 0 or 5000 (no other values)
//
// **Validates: Requirements 3.1, 3.2, 9.1, 9.2, 9.3**
func TestProperty3_CancellationFeeRules(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		times := genConfirmedCancelled().Draw(t, "cancellation")
		confirmedAt, cancelledAt := times[0], times[1]
		elapsed := cancelledAt.Sub(confirmedAt)

		// Act
		fee := pricing.CalculateCancellationFee(confirmedAt, cancelledAt)

		// Assert — fee is one of exactly two values
		assert.Contains(t, []int64{0, pricing.CancelFee}, fee,
			"cancellation fee must be 0 or %d, got %d", pricing.CancelFee, fee)

		// Assert — correct value based on elapsed time
		if elapsed <= pricing.CancelFreeWindow {
			assert.Equal(t, int64(0), fee,
				"fee should be 0 when elapsed %v <= %v", elapsed, pricing.CancelFreeWindow)
		} else {
			assert.Equal(t, pricing.CancelFee, fee,
				"fee should be %d when elapsed %v > %v", pricing.CancelFee, elapsed, pricing.CancelFreeWindow)
		}
	})
}

// --- Property 6: Billing Total Invariant ---

// genBillingRecord generates a BillingRecord with random non-negative fee fields.
func genBillingRecord() *rapid.Generator[*BillingRecord] {
	return rapid.Custom[*BillingRecord](func(t *rapid.T) *BillingRecord {
		return &BillingRecord{
			BookingFee:      rapid.Int64Range(0, 100_000).Draw(t, "bookingFee"),
			ParkingFee:      rapid.Int64Range(0, 500_000).Draw(t, "parkingFee"),
			OvernightFee:    rapid.Int64Range(0, 20_000).Draw(t, "overnightFee"),
			CancellationFee: rapid.Int64Range(0, 5_000).Draw(t, "cancellationFee"),
			PenaltyAmount:   rapid.Int64Range(0, 200_000).Draw(t, "penaltyAmount"),
		}
	})
}

// TestProperty6_BillingTotalInvariant verifies that for any BillingRecord with random
// non-negative fee fields:
//   - pricing.CalculateTotal == sum of all fee fields
//   - total >= bookingFee (when bookingFee > 0 and other fees >= 0)
//
// **Validates: Requirements 10.3, 13.3**
func TestProperty6_BillingTotalInvariant(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		record := genBillingRecord().Draw(t, "record")

		// Act
		total := pricing.CalculateTotal(record.BookingFee, record.ParkingFee, record.OvernightFee,
			record.CancellationFee, record.PenaltyAmount)

		// Assert — total == sum of all fee fields
		expectedTotal := record.BookingFee + record.ParkingFee + record.OvernightFee +
			record.CancellationFee + record.PenaltyAmount
		assert.Equal(t, expectedTotal, total,
			"total should equal sum of all fee fields")

		// Assert — total >= bookingFee (since all other fees are >= 0)
		assert.GreaterOrEqual(t, total, record.BookingFee,
			"total must be >= bookingFee when other fees are non-negative")
	})
}
