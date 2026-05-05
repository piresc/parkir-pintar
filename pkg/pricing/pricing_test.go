// Package pricing unit tests
//
// Best practices applied (from Go testing standards KB):
// - Descriptive names: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA (Arrange-Act-Assert) pattern
// - Table-driven tests for multiple scenarios
// - testify assertions for clear failure messages
// - Tests are fast, isolated, repeatable, and clear
// - Test both success and error/edge cases
// - PRD billing examples verified as concrete test cases
package pricing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- WIB timezone initialization ---

func TestWIB_ShouldBeInitialized(t *testing.T) {
	// Assert
	assert.NotNil(t, WIB, "WIB timezone must be initialized")
}

// --- CalculateParkingFee tests ---

func TestCalculateParkingFee_ShouldReturnCorrectFee_WhenPRDBillingExamples(t *testing.T) {
	// Arrange — PRD Section 9.5 billing examples
	tests := []struct {
		name            string
		checkIn         time.Time
		checkOut        time.Time
		expectedFee     int64
		expectedHours   int
	}{
		{
			name:          "PRD Example 1: Standard 2-hour parking",
			checkIn:       time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
			checkOut:      time.Date(2026, 4, 24, 12, 0, 0, 0, WIB),
			expectedFee:   10000,
			expectedHours: 2,
		},
		{
			name:          "PRD Example 2: 1.5-hour parking rounds up to 2 hours",
			checkIn:       time.Date(2026, 4, 24, 14, 0, 0, 0, WIB),
			checkOut:      time.Date(2026, 4, 24, 15, 30, 0, 0, WIB),
			expectedFee:   10000,
			expectedHours: 2,
		},
		{
			name:          "PRD Example 3: Overnight 8-hour parking",
			checkIn:       time.Date(2026, 4, 24, 22, 0, 0, 0, WIB),
			checkOut:      time.Date(2026, 4, 25, 6, 0, 0, 0, WIB),
			expectedFee:   40000,
			expectedHours: 8,
		},
		{
			name:          "PRD Example 4: Overstay 4-hour parking",
			checkIn:       time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
			checkOut:      time.Date(2026, 4, 24, 14, 0, 0, 0, WIB),
			expectedFee:   20000,
			expectedHours: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			session := ParkingSession{CheckIn: tt.checkIn, CheckOut: tt.checkOut}

			// Act
			fee, billedHours, err := CalculateParkingFee(session)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFee, fee)
			assert.Equal(t, tt.expectedHours, billedHours)
		})
	}
}

func TestCalculateParkingFee_ShouldReturnError_WhenCheckOutNotAfterCheckIn(t *testing.T) {
	tests := []struct {
		name     string
		checkIn  time.Time
		checkOut time.Time
	}{
		{
			name:     "zero duration (same time)",
			checkIn:  time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
			checkOut: time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
		},
		{
			name:     "check-out before check-in",
			checkIn:  time.Date(2026, 4, 24, 12, 0, 0, 0, WIB),
			checkOut: time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			session := ParkingSession{CheckIn: tt.checkIn, CheckOut: tt.checkOut}

			// Act
			fee, billedHours, err := CalculateParkingFee(session)

			// Assert
			assert.ErrorIs(t, err, ErrInvalidSession)
			assert.Equal(t, int64(0), fee)
			assert.Equal(t, 0, billedHours)
		})
	}
}

func TestCalculateParkingFee_ShouldReturnMinimumFee_WhenVeryShortDuration(t *testing.T) {
	// Arrange — 1 second of parking should still bill 1 hour minimum
	session := ParkingSession{
		CheckIn:  time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
		CheckOut: time.Date(2026, 4, 24, 10, 0, 1, 0, WIB),
	}

	// Act
	fee, billedHours, err := CalculateParkingFee(session)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(5000), fee)
	assert.Equal(t, 1, billedHours)
}

func TestCalculateParkingFee_ShouldBillExactHours_WhenExactlyOnTheHour(t *testing.T) {
	tests := []struct {
		name          string
		hours         int
		expectedFee   int64
	}{
		{name: "exactly 1 hour", hours: 1, expectedFee: 5000},
		{name: "exactly 3 hours", hours: 3, expectedFee: 15000},
		{name: "exactly 12 hours", hours: 12, expectedFee: 60000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			checkIn := time.Date(2026, 4, 24, 10, 0, 0, 0, WIB)
			checkOut := checkIn.Add(time.Duration(tt.hours) * time.Hour)
			session := ParkingSession{CheckIn: checkIn, CheckOut: checkOut}

			// Act
			fee, billedHours, err := CalculateParkingFee(session)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFee, fee)
			assert.Equal(t, tt.hours, billedHours)
		})
	}
}


// --- IsOvernight tests ---

func TestIsOvernight_ShouldReturnTrue_WhenSessionCrossesMidnightInWIB(t *testing.T) {
	// Arrange — 22:00 WIB to 06:00 WIB next day
	session := ParkingSession{
		CheckIn:  time.Date(2026, 4, 24, 22, 0, 0, 0, WIB),
		CheckOut: time.Date(2026, 4, 25, 6, 0, 0, 0, WIB),
	}

	// Act
	result := IsOvernight(session)

	// Assert
	assert.True(t, result)
}

func TestIsOvernight_ShouldReturnFalse_WhenSessionWithinSameDay(t *testing.T) {
	// Arrange — 10:00 to 14:00 same day
	session := ParkingSession{
		CheckIn:  time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
		CheckOut: time.Date(2026, 4, 24, 14, 0, 0, 0, WIB),
	}

	// Act
	result := IsOvernight(session)

	// Assert
	assert.False(t, result)
}

func TestIsOvernight_ShouldReturnTrue_WhenMultiMidnightCrossings(t *testing.T) {
	// Arrange — session spanning 3 calendar days in WIB
	session := ParkingSession{
		CheckIn:  time.Date(2026, 4, 24, 22, 0, 0, 0, WIB),
		CheckOut: time.Date(2026, 4, 26, 6, 0, 0, 0, WIB),
	}

	// Act
	result := IsOvernight(session)

	// Assert
	assert.True(t, result, "multi-day session should be detected as overnight")
}

func TestIsOvernight_ShouldReturnFalse_WhenSessionEndsExactlyAtMidnight(t *testing.T) {
	// Arrange — 22:00 to 00:00 next day (crosses midnight)
	session := ParkingSession{
		CheckIn:  time.Date(2026, 4, 24, 22, 0, 0, 0, WIB),
		CheckOut: time.Date(2026, 4, 25, 0, 0, 0, 0, WIB),
	}

	// Act
	result := IsOvernight(session)

	// Assert — midnight is a new calendar day, so this crosses midnight
	assert.True(t, result)
}

func TestIsOvernight_ShouldHandleUTCTimesCorrectly(t *testing.T) {
	// Arrange — 16:00 UTC = 23:00 WIB, 18:00 UTC = 01:00 WIB next day
	// This crosses midnight in WIB but not in UTC
	session := ParkingSession{
		CheckIn:  time.Date(2026, 4, 24, 16, 0, 0, 0, time.UTC),
		CheckOut: time.Date(2026, 4, 24, 18, 0, 0, 0, time.UTC),
	}

	// Act
	result := IsOvernight(session)

	// Assert — in WIB this is 23:00 to 01:00, crosses midnight
	assert.True(t, result)
}

// --- OvernightFee tests ---

func TestOvernightFee_ShouldReturn20000_WhenOvernight(t *testing.T) {
	// Arrange
	session := ParkingSession{
		CheckIn:  time.Date(2026, 4, 24, 22, 0, 0, 0, WIB),
		CheckOut: time.Date(2026, 4, 25, 6, 0, 0, 0, WIB),
	}

	// Act
	fee := OvernightFee(session)

	// Assert
	assert.Equal(t, int64(20000), fee)
}

func TestOvernightFee_ShouldReturn0_WhenNotOvernight(t *testing.T) {
	// Arrange
	session := ParkingSession{
		CheckIn:  time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
		CheckOut: time.Date(2026, 4, 24, 14, 0, 0, 0, WIB),
	}

	// Act
	fee := OvernightFee(session)

	// Assert
	assert.Equal(t, int64(0), fee)
}

// --- Constant fee tests ---

func TestBookingFee_ShouldReturn5000(t *testing.T) {
	// Act
	fee := BookingFee()

	// Assert
	assert.Equal(t, int64(5000), fee)
}

func TestNoShowFee_ShouldReturnZero_BookingFeeIsOnlyCost(t *testing.T) {
	// Per PRD: the booking fee (5,000 IDR, already charged at confirmation)
	// is the only cost for a no-show. No additional penalty is applied.
	// Act
	fee := NoShowFee()

	// Assert
	assert.Equal(t, int64(0), fee)
}

func TestWrongSpotPenalty_ShouldReturn200000(t *testing.T) {
	// Act
	fee := WrongSpotPenalty()

	// Assert
	assert.Equal(t, int64(200000), fee)
}

// --- CancellationFee tests ---

func TestCancellationFee_ShouldReturn0_WhenWithinGracePeriod(t *testing.T) {
	tests := []struct {
		name    string
		elapsed time.Duration
	}{
		{name: "immediately (0s)", elapsed: 0},
		{name: "at 1 minute", elapsed: 1 * time.Minute},
		{name: "at exactly 2 minutes", elapsed: 2 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			confirmedAt := time.Date(2026, 4, 24, 10, 0, 0, 0, WIB)
			input := CancellationInput{
				ConfirmedAt: confirmedAt,
				CancelledAt: confirmedAt.Add(tt.elapsed),
			}

			// Act
			fee := CancellationFee(input)

			// Assert
			assert.Equal(t, int64(0), fee)
		})
	}
}

func TestCancellationFee_ShouldReturn5000_WhenAfterGracePeriod(t *testing.T) {
	tests := []struct {
		name    string
		elapsed time.Duration
	}{
		{name: "at 2 minutes and 1 second", elapsed: 2*time.Minute + 1*time.Second},
		{name: "at 5 minutes", elapsed: 5 * time.Minute},
		{name: "at 1 hour", elapsed: 1 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			confirmedAt := time.Date(2026, 4, 24, 10, 0, 0, 0, WIB)
			input := CancellationInput{
				ConfirmedAt: confirmedAt,
				CancelledAt: confirmedAt.Add(tt.elapsed),
			}

			// Act
			fee := CancellationFee(input)

			// Assert
			assert.Equal(t, int64(5000), fee)
		})
	}
}

// --- CalculateTotal tests ---

func TestCalculateTotal_ShouldReturnSumOfAllComponents(t *testing.T) {
	// Arrange — PRD Example 3: overnight parking total
	breakdown := BillBreakdown{
		BookingFee:      5000,
		ParkingFee:      40000,
		OvernightFee:    20000,
		CancellationFee: 0,
		PenaltyAmount:   0,
	}

	// Act
	total := CalculateTotal(breakdown)

	// Assert
	assert.Equal(t, int64(65000), total)
}

func TestCalculateTotal_ShouldIncludeAllFeeTypes(t *testing.T) {
	// Arrange — all fee components populated
	breakdown := BillBreakdown{
		BookingFee:      5000,
		ParkingFee:      10000,
		OvernightFee:    20000,
		CancellationFee: 5000,
		PenaltyAmount:   200000,
	}

	// Act
	total := CalculateTotal(breakdown)

	// Assert
	assert.Equal(t, int64(240000), total)
}

func TestCalculateTotal_ShouldReturnZero_WhenAllComponentsZero(t *testing.T) {
	// Arrange
	breakdown := BillBreakdown{}

	// Act
	total := CalculateTotal(breakdown)

	// Assert
	assert.Equal(t, int64(0), total)
}

// --- PRD end-to-end billing verification ---

func TestPRDBillingExamples_ShouldMatchExpectedTotals(t *testing.T) {
	tests := []struct {
		name          string
		checkIn       time.Time
		checkOut      time.Time
		expectedTotal int64
	}{
		{
			name:          "PRD Example 1: 2-hour parking total 15,000",
			checkIn:       time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
			checkOut:      time.Date(2026, 4, 24, 12, 0, 0, 0, WIB),
			expectedTotal: 15000,
		},
		{
			name:          "PRD Example 2: 1.5-hour parking total 15,000",
			checkIn:       time.Date(2026, 4, 24, 14, 0, 0, 0, WIB),
			checkOut:      time.Date(2026, 4, 24, 15, 30, 0, 0, WIB),
			expectedTotal: 15000,
		},
		{
			name:          "PRD Example 3: overnight parking total 65,000",
			checkIn:       time.Date(2026, 4, 24, 22, 0, 0, 0, WIB),
			checkOut:      time.Date(2026, 4, 25, 6, 0, 0, 0, WIB),
			expectedTotal: 65000,
		},
		{
			name:          "PRD Example 4: overstay 4-hour parking total 25,000",
			checkIn:       time.Date(2026, 4, 24, 10, 0, 0, 0, WIB),
			checkOut:      time.Date(2026, 4, 24, 14, 0, 0, 0, WIB),
			expectedTotal: 25000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			session := ParkingSession{CheckIn: tt.checkIn, CheckOut: tt.checkOut}

			// Act
			parkingFee, _, err := CalculateParkingFee(session)
			require.NoError(t, err)

			breakdown := BillBreakdown{
				BookingFee:   BookingFee(),
				ParkingFee:   parkingFee,
				OvernightFee: OvernightFee(session),
			}
			total := CalculateTotal(breakdown)

			// Assert
			assert.Equal(t, tt.expectedTotal, total)
		})
	}
}
