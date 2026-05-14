// Package model defines domain structs and request types for the billing module.
package model

import "time"

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
	Amount        int64  `json:"amount"` // configurable overnight fee amount; falls back to pricing.OvernightPerNight if 0
}
