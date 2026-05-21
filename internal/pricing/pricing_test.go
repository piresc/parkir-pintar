package pricing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateSessionFee_MinimumOneHour(t *testing.T) {
	checkIn := time.Date(2026, 1, 15, 10, 0, 0, 0, wib)
	checkOut := time.Date(2026, 1, 15, 10, 15, 0, 0, wib) // 15 min

	result := CalculateSessionFee(checkIn, checkOut)

	assert.Equal(t, int64(5_000), result.ParkingFee) // 1 hour minimum
	assert.Equal(t, 1, result.BilledHours)
	assert.Equal(t, 15, result.DurationMinutes)
	assert.Equal(t, int64(0), result.OvernightFee)
	assert.False(t, result.IsOvernight)
}

func TestCalculateSessionFee_ExactTwoHours(t *testing.T) {
	checkIn := time.Date(2026, 1, 15, 10, 0, 0, 0, wib)
	checkOut := time.Date(2026, 1, 15, 12, 0, 0, 0, wib)

	result := CalculateSessionFee(checkIn, checkOut)

	assert.Equal(t, int64(10_000), result.ParkingFee)
	assert.Equal(t, 2, result.BilledHours)
	assert.False(t, result.IsOvernight)
}

func TestCalculateSessionFee_CeilingHours(t *testing.T) {
	checkIn := time.Date(2026, 1, 15, 10, 0, 0, 0, wib)
	checkOut := time.Date(2026, 1, 15, 12, 1, 0, 0, wib) // 2h1m → 3 hours

	result := CalculateSessionFee(checkIn, checkOut)

	assert.Equal(t, int64(15_000), result.ParkingFee)
	assert.Equal(t, 3, result.BilledHours)
}

func TestCalculateSessionFee_OneMidnightCrossed(t *testing.T) {
	checkIn := time.Date(2026, 1, 15, 22, 0, 0, 0, wib)
	checkOut := time.Date(2026, 1, 16, 2, 0, 0, 0, wib) // crosses 1 midnight

	result := CalculateSessionFee(checkIn, checkOut)

	assert.Equal(t, int64(20_000), result.ParkingFee) // 4 hours
	assert.Equal(t, int64(20_000), result.OvernightFee)
	assert.Equal(t, 1, result.NightsCrossed)
	assert.True(t, result.IsOvernight)
}

func TestCalculateSessionFee_TwoMidnightsCrossed(t *testing.T) {
	checkIn := time.Date(2026, 1, 15, 23, 0, 0, 0, wib)
	checkOut := time.Date(2026, 1, 17, 1, 0, 0, 0, wib) // crosses 2 midnights

	result := CalculateSessionFee(checkIn, checkOut)

	assert.Equal(t, int64(40_000), result.OvernightFee) // 20k × 2
	assert.Equal(t, 2, result.NightsCrossed)
	assert.True(t, result.IsOvernight)
	assert.Equal(t, 26, result.BilledHours)
	assert.Equal(t, int64(130_000), result.ParkingFee) // 26 × 5k
}

func TestCalculateSessionFee_ThreeMidnightsCrossed(t *testing.T) {
	checkIn := time.Date(2026, 1, 15, 20, 0, 0, 0, wib)
	checkOut := time.Date(2026, 1, 18, 8, 0, 0, 0, wib) // 3 midnights

	result := CalculateSessionFee(checkIn, checkOut)

	assert.Equal(t, int64(60_000), result.OvernightFee) // 20k × 3
	assert.Equal(t, 3, result.NightsCrossed)
}

func TestCalculateSessionFee_SameDayNoOvernight(t *testing.T) {
	checkIn := time.Date(2026, 1, 15, 8, 0, 0, 0, wib)
	checkOut := time.Date(2026, 1, 15, 23, 59, 0, 0, wib)

	result := CalculateSessionFee(checkIn, checkOut)

	assert.Equal(t, int64(0), result.OvernightFee)
	assert.Equal(t, 0, result.NightsCrossed)
	assert.False(t, result.IsOvernight)
}

func TestCalculateTotal(t *testing.T) {
	total := CalculateTotal(5_000, 15_000, 20_000)
	assert.Equal(t, int64(40_000), total)
}

func TestCalculateTotal_AllZero(t *testing.T) {
	total := CalculateTotal(0, 0, 0)
	assert.Equal(t, int64(0), total)
}
