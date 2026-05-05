// Package model defines domain structs, pricing constants, and fee calculation
// functions for the billing module.
package model

import "time"

// Pricing constants for the ParkirPintar billing system.
// All monetary values are in IDR (Indonesian Rupiah).
const (
	BookingFee       int64 = 5_000
	HourlyRate       int64 = 5_000
	OvernightFlatFee int64 = 20_000
	WrongSpotPenalty int64 = 200_000
	CancelFreeWindow       = 2 * time.Minute
	CancelFee        int64 = 5_000
	// NoShowFee is zero because the PRD states the booking fee (5,000 IDR,
	// already charged at confirmation) is the only cost for a no-show.
	// The driver forfeits the booking fee — no additional penalty is applied.
	NoShowFee int64 = 0
)

// Billing status constants.
const (
	BillingStatusPending    = "pending"
	BillingStatusCalculated = "calculated"
	BillingStatusInvoiced   = "invoiced"
	BillingStatusPaid       = "paid"
)

// BillingRecord represents a billing record for a single reservation.
type BillingRecord struct {
	ID              string    `json:"id" db:"id"`
	ReservationID   string    `json:"reservation_id" db:"reservation_id"`
	BookingFee      int64     `json:"booking_fee" db:"booking_fee"`
	ParkingFee      int64     `json:"parking_fee" db:"parking_fee"`
	OvernightFee    int64     `json:"overnight_fee" db:"overnight_fee"`
	CancellationFee int64     `json:"cancellation_fee" db:"cancellation_fee"`
	PenaltyAmount   int64     `json:"penalty_amount" db:"penalty_amount"`
	TotalAmount     int64     `json:"total_amount" db:"total_amount"`
	DurationMinutes int       `json:"duration_minutes" db:"duration_minutes"`
	BilledHours     int       `json:"billed_hours" db:"billed_hours"`
	IsOvernight     bool      `json:"is_overnight" db:"is_overnight"`
	IdempotencyKey  string    `json:"idempotency_key" db:"idempotency_key"`
	Status          string    `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at,omitzero" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at,omitzero" db:"updated_at"`
}

// Penalty represents a penalty applied to a reservation.
type Penalty struct {
	ID            string    `json:"id" db:"id"`
	ReservationID string    `json:"reservation_id" db:"reservation_id"`
	PenaltyType   string    `json:"penalty_type" db:"penalty_type"`
	Amount        int64     `json:"amount" db:"amount"`
	Description   string    `json:"description" db:"description"`
	CreatedAt     time.Time `json:"created_at,omitzero" db:"created_at"`
}

// ParkingFeeResult holds the computed parking fee breakdown.
type ParkingFeeResult struct {
	ParkingFee      int64
	OvernightFee    int64
	DurationMinutes int
	BilledHours     int
	IsOvernight     bool
}

// CalculateParkingFee computes the total parking fee for a session.
//
// Preconditions:
//   - checkIn is before checkOut
//   - Both times are valid (non-zero)
//
// Postconditions:
//   - ParkingFee = ceil(duration_in_hours) * HourlyRate
//   - OvernightFee = OvernightFlatFee if session crosses midnight in WIB, else 0
//   - BilledHours >= 1 (minimum 1 hour)
//   - DurationMinutes = actual minutes between checkIn and checkOut
func CalculateParkingFee(checkIn, checkOut time.Time) ParkingFeeResult {
	duration := checkOut.Sub(checkIn)
	durationMinutes := int(duration.Minutes())

	// Ceiling-based hour calculation
	billedHours := int(duration.Hours())
	if duration > time.Duration(billedHours)*time.Hour {
		billedHours++
	}
	billedHours = max(billedHours, 1)

	parkingFee := int64(billedHours) * HourlyRate

	// Overnight detection: session crosses midnight boundary in WIB
	isOvernight := crossesMidnight(checkIn, checkOut)
	var overnightFee int64
	if isOvernight {
		overnightFee = OvernightFlatFee
	}

	return ParkingFeeResult{
		ParkingFee:      parkingFee,
		OvernightFee:    overnightFee,
		DurationMinutes: durationMinutes,
		BilledHours:     billedHours,
		IsOvernight:     isOvernight,
	}
}

// crossesMidnight returns true if the time range [start, end) spans a midnight
// boundary in WIB (UTC+7). It normalizes both times to WIB and compares
// calendar dates.
func crossesMidnight(start, end time.Time) bool {
	loc := time.FixedZone("WIB", 7*60*60)
	s := start.In(loc)
	e := end.In(loc)

	sDate := time.Date(s.Year(), s.Month(), s.Day(), 0, 0, 0, 0, loc)
	eDate := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, loc)
	return eDate.After(sDate)
}

// CalculateCancellationFee determines the cancellation fee based on time
// elapsed since confirmation.
//
// Preconditions:
//   - confirmedAt is a valid non-zero time
//   - cancelledAt is after confirmedAt
//
// Postconditions:
//   - Returns 0 if cancelled within CancelFreeWindow (2 minutes)
//   - Returns CancelFee (5,000 IDR) if cancelled after CancelFreeWindow
func CalculateCancellationFee(confirmedAt, cancelledAt time.Time) int64 {
	elapsed := cancelledAt.Sub(confirmedAt)
	if elapsed <= CancelFreeWindow {
		return 0
	}
	return CancelFee
}

// CalculateBillingTotal computes the total amount for a billing record as the
// sum of all fee fields.
func CalculateBillingTotal(record *BillingRecord) int64 {
	return record.BookingFee + record.ParkingFee + record.OvernightFee +
		record.CancellationFee + record.PenaltyAmount
}

// Request types for billing service operations.

// StartBillingRequest is the payload for starting billing on a reservation.
type StartBillingRequest struct {
	ReservationID  string `json:"reservation_id"`
	BookingFee     int64  `json:"booking_fee"`
	IdempotencyKey string `json:"idempotency_key"`
}

// CalculateFeeRequest is the payload for calculating fees at check-out.
type CalculateFeeRequest struct {
	ReservationID string    `json:"reservation_id"`
	CheckInAt     time.Time `json:"check_in_at"`
	CheckOutAt    time.Time `json:"check_out_at"`
}

// GenerateInvoiceRequest is the payload for generating an invoice.
type GenerateInvoiceRequest struct {
	ReservationID  string `json:"reservation_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

// ApplyPenaltyRequest is the payload for applying a penalty to a reservation.
type ApplyPenaltyRequest struct {
	ReservationID string `json:"reservation_id"`
	PenaltyType   string `json:"penalty_type"`
	Amount        int64  `json:"amount"`
	Description   string `json:"description"`
}

// ApplyOvernightFeeRequest is the payload for applying an overnight fee.
type ApplyOvernightFeeRequest struct {
	ReservationID string `json:"reservation_id"`
}
