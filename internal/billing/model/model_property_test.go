package model

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"

	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/pkg/pricing"
)

func wibLoc() *time.Location {
	return time.FixedZone("WIB", 7*60*60)
}

func genCheckInCheckOut() *rapid.Generator[[2]time.Time] {
	return rapid.Custom[[2]time.Time](func(t *rapid.T) [2]time.Time {
		loc := wibLoc()
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day") // safe for all months
		hour := rapid.IntRange(0, 23).Draw(t, "hour")
		minute := rapid.IntRange(0, 59).Draw(t, "minute")

		checkIn := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)

		durationSec := rapid.Int64Range(60, 48*3600).Draw(t, "durationSec")
		checkOut := checkIn.Add(time.Duration(durationSec) * time.Second)

		return [2]time.Time{checkIn, checkOut}
	})
}

func genSameDaySession() *rapid.Generator[[2]time.Time] {
	return rapid.Custom[[2]time.Time](func(t *rapid.T) [2]time.Time {
		loc := wibLoc()
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		startHour := rapid.IntRange(0, 22).Draw(t, "startHour")
		startMin := rapid.IntRange(0, 59).Draw(t, "startMin")

		checkIn := time.Date(year, time.Month(month), day, startHour, startMin, 0, 0, loc)

		endOfDay := time.Date(year, time.Month(month), day, 23, 59, 59, 0, loc)
		maxDuration := endOfDay.Sub(checkIn)
		if maxDuration < time.Minute {
			return [2]time.Time{checkIn, checkIn.Add(time.Minute)}
		}

		durationSec := rapid.Int64Range(60, int64(maxDuration.Seconds())).Draw(t, "durationSec")
		checkOut := checkIn.Add(time.Duration(durationSec) * time.Second)

		return [2]time.Time{checkIn, checkOut}
	})
}

func genCrossDaySession() *rapid.Generator[[2]time.Time] {
	return rapid.Custom[[2]time.Time](func(t *rapid.T) [2]time.Time {
		loc := wibLoc()
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 27).Draw(t, "day") // leave room for next day
		startHour := rapid.IntRange(0, 23).Draw(t, "startHour")
		startMin := rapid.IntRange(0, 59).Draw(t, "startMin")

		checkIn := time.Date(year, time.Month(month), day, startHour, startMin, 0, 0, loc)

		nextDay := time.Date(year, time.Month(month), day+1, 0, 0, 0, 0, loc)
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

func TestProperty1_PricingCorrectness(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		times := genCheckInCheckOut().Draw(t, "session")
		checkIn, checkOut := times[0], times[1]

		result := pricing.CalculateSessionFee(checkIn, checkOut)

		duration := checkOut.Sub(checkIn)
		expectedBilledHours := int(math.Ceil(duration.Hours()))
		if expectedBilledHours < 1 {
			expectedBilledHours = 1
		}
		assert.Equal(t, expectedBilledHours, result.BilledHours,
			"billedHours should be ceil(duration_in_hours) for duration %v", duration)

		assert.GreaterOrEqual(t, result.BilledHours, 1,
			"billedHours must be at least 1")

		assert.Equal(t, int64(result.BilledHours)*constants.HourlyRate, result.ParkingFee,
			"parkingFee should be billedHours * %d", constants.HourlyRate)

		expectedMinutes := int(duration.Minutes())
		assert.Equal(t, expectedMinutes, result.DurationMinutes,
			"durationMinutes should match actual minutes")
	})
}

func TestProperty2_OvernightDetection_SameDay(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		times := genSameDaySession().Draw(t, "sameDaySession")
		checkIn, checkOut := times[0], times[1]

		result := pricing.CalculateSessionFee(checkIn, checkOut)

		assert.False(t, result.IsOvernight,
			"same-day session should not be overnight: checkIn=%v checkOut=%v", checkIn, checkOut)
		assert.Equal(t, int64(0), result.OvernightFee,
			"overnightFee should be 0 for same-day session")
	})
}

func TestProperty2_OvernightDetection_CrossDay(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		times := genCrossDaySession().Draw(t, "crossDaySession")
		checkIn, checkOut := times[0], times[1]

		result := pricing.CalculateSessionFee(checkIn, checkOut)

		assert.True(t, result.IsOvernight,
			"cross-day session should be overnight: checkIn=%v checkOut=%v", checkIn, checkOut)
		assert.Greater(t, result.OvernightFee, int64(0),
			"overnightFee should be > 0 for cross-day session")
	})
}

func genBillingRecord() *rapid.Generator[*BillingRecord] {
	return rapid.Custom[*BillingRecord](func(t *rapid.T) *BillingRecord {
		return &BillingRecord{
			BookingFee:   rapid.Int64Range(0, 100_000).Draw(t, "bookingFee"),
			ParkingFee:   rapid.Int64Range(0, 500_000).Draw(t, "parkingFee"),
			OvernightFee: rapid.Int64Range(0, 20_000).Draw(t, "overnightFee"),
		}
	})
}

func TestProperty6_BillingTotalInvariant(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		record := genBillingRecord().Draw(t, "record")

		total := pricing.CalculateTotal(record.BookingFee, record.ParkingFee, record.OvernightFee)

		expectedTotal := record.BookingFee + record.ParkingFee + record.OvernightFee
		assert.Equal(t, expectedTotal, total,
			"total should equal sum of all fee fields")

		// Assert — total >= bookingFee (since all other fees are >= 0)
		assert.GreaterOrEqual(t, total, record.BookingFee,
			"total must be >= bookingFee when other fees are non-negative")
	})
}
