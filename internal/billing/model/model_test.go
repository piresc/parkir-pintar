// Package model defines domain structs and request types for the billing module.
//
// Best practices applied (from coding standards KB):
// - Table-driven tests with t.Run() subtests for comprehensive pricing coverage
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - t.Context() for context (Go 1.24+)
// - WIB timezone (UTC+7) for all test times
// - Each subtest is isolated and self-descriptive
package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"parkir-pintar/pkg/pricing"
)

// wib returns the WIB (UTC+7) timezone used for all test times.
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
			// Arrange
			_ = t.Context()

			// Act
			result := pricing.CalculateSessionFee(tt.checkIn, tt.checkOut)

			// Assert
			assert.Equal(t, tt.billedHours, result.BilledHours)
			assert.Equal(t, tt.parkingFee, result.ParkingFee)
			assert.Equal(t, tt.isOvernight, result.IsOvernight)
			assert.Equal(t, tt.overnightFee, result.OvernightFee)
			assert.Equal(t, tt.durationMinutes, result.DurationMinutes)
		})
	}
}

func TestCalculateCancellationFee_ShouldReturnCorrectFee(t *testing.T) {
	baseTime := time.Date(2026, 4, 24, 10, 0, 0, 0, wib())

	tests := []struct {
		name        string
		confirmedAt time.Time
		cancelledAt time.Time
		expectedFee int64
	}{
		{
			name:        "within 2 min is free",
			confirmedAt: baseTime,
			cancelledAt: baseTime.Add(1 * time.Minute),
			expectedFee: 0,
		},
		{
			name:        "exactly 2 min is free",
			confirmedAt: baseTime,
			cancelledAt: baseTime.Add(2 * time.Minute),
			expectedFee: 0,
		},
		{
			name:        "after 2 min charges fee",
			confirmedAt: baseTime,
			cancelledAt: baseTime.Add(3 * time.Minute),
			expectedFee: 5_000,
		},
		{
			name:        "after 5 min charges fee",
			confirmedAt: baseTime,
			cancelledAt: baseTime.Add(5 * time.Minute),
			expectedFee: 5_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_ = t.Context()

			// Act
			fee := pricing.CalculateCancellationFee(tt.confirmedAt, tt.cancelledAt)

			// Assert
			assert.Equal(t, tt.expectedFee, fee)
		})
	}
}

func TestCalculateTotal_ShouldSumAllFees(t *testing.T) {
	tests := []struct {
		name           string
		record         *BillingRecord
		expectedTotal  int64
		totalGEBooking bool
	}{
		{
			name: "all fees present",
			record: &BillingRecord{
				BookingFee:      5_000,
				ParkingFee:      10_000,
				OvernightFee:    20_000,
				CancellationFee: 0,
				PenaltyAmount:   200_000,
			},
			expectedTotal:  235_000,
			totalGEBooking: true,
		},
		{
			name: "booking only",
			record: &BillingRecord{
				BookingFee:      5_000,
				ParkingFee:      0,
				OvernightFee:    0,
				CancellationFee: 0,
				PenaltyAmount:   0,
			},
			expectedTotal:  5_000,
			totalGEBooking: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_ = t.Context()

			// Act
			total := pricing.CalculateTotal(tt.record.BookingFee, tt.record.ParkingFee,
				tt.record.OvernightFee, tt.record.CancellationFee, tt.record.PenaltyAmount)

			// Assert
			assert.Equal(t, tt.expectedTotal, total)
			if tt.totalGEBooking {
				assert.GreaterOrEqual(t, total, tt.record.BookingFee,
					"total must be >= booking_fee")
			}
		})
	}
}
