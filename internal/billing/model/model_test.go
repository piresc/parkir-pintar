package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"parkir-pintar/internal/billing/pricing"
)

func wib() *time.Location {
	return time.FixedZone("WIB", 7*60*60)
}

func TestCalculateSessionFee_ShouldComputeCorrectFees(t *testing.T) {
	loc := wib()

	tests := []struct {
		name            string
		checkIn         time.Time
		checkOut        time.Time
		billedHours     int
		parkingFee      int64
		isOvernight     bool
		overnightFee    int64
		durationMinutes int
	}{
		{
			name:            "standard 2h session",
			checkIn:         time.Date(2026, 4, 24, 10, 0, 0, 0, loc),
			checkOut:        time.Date(2026, 4, 24, 12, 0, 0, 0, loc),
			billedHours:     2,
			parkingFee:      10_000,
			isOvernight:     false,
			overnightFee:    0,
			durationMinutes: 120,
		},
		{
			name:            "partial hour rounds up",
			checkIn:         time.Date(2026, 4, 24, 14, 0, 0, 0, loc),
			checkOut:        time.Date(2026, 4, 24, 15, 30, 0, 0, loc),
			billedHours:     2,
			parkingFee:      10_000,
			isOvernight:     false,
			overnightFee:    0,
			durationMinutes: 90,
		},
		{
			name:            "minimum 1 billed hour",
			checkIn:         time.Date(2026, 4, 24, 10, 0, 0, 0, loc),
			checkOut:        time.Date(2026, 4, 24, 10, 1, 0, 0, loc),
			billedHours:     1,
			parkingFee:      5_000,
			isOvernight:     false,
			overnightFee:    0,
			durationMinutes: 1,
		},
		{
			name:            "exact 1 hour",
			checkIn:         time.Date(2026, 4, 24, 10, 0, 0, 0, loc),
			checkOut:        time.Date(2026, 4, 24, 11, 0, 0, 0, loc),
			billedHours:     1,
			parkingFee:      5_000,
			isOvernight:     false,
			overnightFee:    0,
			durationMinutes: 60,
		},
		{
			name:            "overnight session crosses midnight",
			checkIn:         time.Date(2026, 4, 24, 22, 0, 0, 0, loc),
			checkOut:        time.Date(2026, 4, 25, 6, 0, 0, 0, loc),
			billedHours:     8,
			parkingFee:      40_000,
			isOvernight:     true,
			overnightFee:    20_000,
			durationMinutes: 480,
		},
		{
			name:            "same day no overnight",
			checkIn:         time.Date(2026, 4, 24, 8, 0, 0, 0, loc),
			checkOut:        time.Date(2026, 4, 24, 20, 0, 0, 0, loc),
			billedHours:     12,
			parkingFee:      60_000,
			isOvernight:     false,
			overnightFee:    0,
			durationMinutes: 720,
		},
		{
			name:            "duration minutes accuracy for 2h30m",
			checkIn:         time.Date(2026, 4, 24, 10, 0, 0, 0, loc),
			checkOut:        time.Date(2026, 4, 24, 12, 30, 0, 0, loc),
			billedHours:     3,
			parkingFee:      15_000,
			isOvernight:     false,
			overnightFee:    0,
			durationMinutes: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = t.Context()

			result := pricing.CalculateSessionFee(tt.checkIn, tt.checkOut)

			assert.Equal(t, tt.billedHours, result.BilledHours)
			assert.Equal(t, tt.parkingFee, result.ParkingFee)
			assert.Equal(t, tt.isOvernight, result.IsOvernight)
			assert.Equal(t, tt.overnightFee, result.OvernightFee)
			assert.Equal(t, tt.durationMinutes, result.DurationMinutes)
		})
	}
}

func TestCalculateTotal_ShouldSumAllFees(t *testing.T) {
	tests := []struct {
		name           string
		bookingFee     int64
		parkingFee     int64
		overnightFee   int64
		expectedTotal  int64
		totalGEBooking bool
	}{
		{
			name:           "all fees present",
			bookingFee:     5_000,
			parkingFee:     10_000,
			overnightFee:   20_000,
			expectedTotal:  35_000,
			totalGEBooking: true,
		},
		{
			name:           "booking only",
			bookingFee:     5_000,
			parkingFee:     0,
			overnightFee:   0,
			expectedTotal:  5_000,
			totalGEBooking: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = t.Context()

			total := pricing.CalculateTotal(tt.bookingFee, tt.parkingFee, tt.overnightFee)

			assert.Equal(t, tt.expectedTotal, total)
			if tt.totalGEBooking {
				assert.GreaterOrEqual(t, total, tt.bookingFee,
					"total must be >= booking_fee")
			}
		})
	}
}
