// Package pricing provides centralized fee calculation functions for the
// ParkirPintar smart parking marketplace. All monetary values are in IDR
// as int64. Overnight determination uses WIB (Asia/Jakarta, UTC+7).
//
// Functions are pure (no side effects) and safe for concurrent use.
package pricing

import (
	"errors"
	"math"
	"time"
)

// WIB is the Asia/Jakarta timezone (UTC+7) used for overnight fee determination.
var WIB *time.Location

func init() {
	var err error
	WIB, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		// Fallback to fixed UTC+7 if timezone database is unavailable.
		WIB = time.FixedZone("WIB", 7*60*60)
	}
}

// Sentinel errors for pricing operations.
var (
	ErrInvalidSession      = errors.New("check-out must be after check-in")
	ErrInvalidCancellation = errors.New("cancelled-at must be after confirmed-at")
)

// ParkingSession represents a parking period with check-in and check-out times.
type ParkingSession struct {
	CheckIn  time.Time
	CheckOut time.Time
}

// CancellationInput represents the timestamps for a reservation cancellation.
type CancellationInput struct {
	ConfirmedAt time.Time
	CancelledAt time.Time
}

// BillBreakdown holds all fee components for a parking transaction.
type BillBreakdown struct {
	BookingFee      int64 // IDR
	ParkingFee      int64 // IDR
	OvernightFee    int64 // IDR
	CancellationFee int64 // IDR
	PenaltyAmount   int64 // IDR
	TotalAmount     int64 // IDR
	BilledHours     int
	IsOvernight     bool
}

// Constant fee amounts in IDR.
const (
	hourlyRate          int64 = 5000
	overnightFeeAmount  int64 = 20000
	bookingFeeAmount    int64 = 5000
	cancellationPenalty int64 = 5000
	// noShowFeeAmount is zero because the PRD states the booking fee (5,000 IDR,
	// already charged at confirmation) is the only cost for a no-show.
	// The driver forfeits the booking fee — no additional penalty is applied.
	noShowFeeAmount int64 = 0
	wrongSpotAmount int64 = 200000
	cancellationGrace     = 2 * time.Minute
)

// CalculateParkingFee computes the parking fee for a session.
// The fee is ceil(duration_in_hours) × 5,000 IDR with a minimum of 5,000 IDR.
// Returns the fee, the number of billed hours, and an error if check-out is
// not after check-in.
func CalculateParkingFee(session ParkingSession) (fee int64, billedHours int, err error) {
	if !session.CheckOut.After(session.CheckIn) {
		return 0, 0, ErrInvalidSession
	}

	duration := session.CheckOut.Sub(session.CheckIn)
	hours := math.Ceil(duration.Hours())
	if hours < 1 {
		hours = 1
	}

	billedHours = int(hours)
	fee = int64(hours) * hourlyRate

	return fee, billedHours, nil
}

// IsOvernight reports whether the parking session crosses midnight in WIB (UTC+7).
func IsOvernight(session ParkingSession) bool {
	inWIB := session.CheckIn.In(WIB)
	outWIB := session.CheckOut.In(WIB)

	// Different calendar dates in WIB means the session crossed midnight.
	inYear, inMonth, inDay := inWIB.Date()
	outYear, outMonth, outDay := outWIB.Date()

	return inYear != outYear || inMonth != outMonth || inDay != outDay
}

// OvernightFee returns 20,000 IDR if the session crosses midnight in WIB,
// 0 IDR otherwise.
func OvernightFee(session ParkingSession) int64 {
	if IsOvernight(session) {
		return overnightFeeAmount
	}
	return 0
}

// BookingFee returns the flat booking fee of 5,000 IDR.
func BookingFee() int64 {
	return bookingFeeAmount
}

// CancellationFee returns 0 IDR if the cancellation is within 2 minutes of
// confirmation, or 5,000 IDR otherwise.
func CancellationFee(input CancellationInput) int64 {
	elapsed := input.CancelledAt.Sub(input.ConfirmedAt)
	if elapsed <= cancellationGrace {
		return 0
	}
	return cancellationPenalty
}

// NoShowFee returns 0 IDR. Per PRD, the booking fee (5,000 IDR) already
// charged at confirmation is the only cost for a no-show — no additional
// penalty is applied when a reservation expires.
func NoShowFee() int64 {
	return noShowFeeAmount
}

// WrongSpotPenalty returns the wrong-spot penalty of 200,000 IDR.
func WrongSpotPenalty() int64 {
	return wrongSpotAmount
}

// CalculateTotal computes the total amount as the sum of all fee components
// in the breakdown: BookingFee + ParkingFee + OvernightFee + CancellationFee + PenaltyAmount.
func CalculateTotal(breakdown BillBreakdown) int64 {
	return breakdown.BookingFee +
		breakdown.ParkingFee +
		breakdown.OvernightFee +
		breakdown.CancellationFee +
		breakdown.PenaltyAmount
}
