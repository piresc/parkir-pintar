package model

import (
	"time"

	"parkir-pintar/internal/billing/constants"
)

// Re-export typed constants for backward compatibility with existing consumers.
const (
	BillingStatusPending    string = string(constants.BillingStatusPending)
	BillingStatusCalculated string = string(constants.BillingStatusCalculated)
	BillingStatusInvoiced   string = string(constants.BillingStatusInvoiced)
	BillingStatusPaid       string = string(constants.BillingStatusPaid)
)

type BillingRecord struct {
	ID              string    `json:"id" db:"id"`
	ReservationID   string    `json:"reservation_id" db:"reservation_id"`
	BookingFee      int64     `json:"booking_fee" db:"booking_fee"`
	ParkingFee      int64     `json:"parking_fee" db:"parking_fee"`
	OvernightFee    int64     `json:"overnight_fee" db:"overnight_fee"`
	TotalAmount     int64     `json:"total_amount" db:"total_amount"`
	DurationMinutes int       `json:"duration_minutes" db:"duration_minutes"`
	BilledHours     int       `json:"billed_hours" db:"billed_hours"`
	IsOvernight     bool      `json:"is_overnight" db:"is_overnight"`
	IdempotencyKey  string    `json:"idempotency_key" db:"idempotency_key"`
	Status          string    `json:"status" db:"status"`
	Version         int       `json:"version" db:"version"`
	CreatedAt       time.Time `json:"created_at,omitzero" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at,omitzero" db:"updated_at"`
}

type StartBillingRequest struct {
	ReservationID  string `json:"reservation_id"`
	BookingFee     int64  `json:"booking_fee"`
	IdempotencyKey string `json:"idempotency_key"`
}

type CalculateFeeRequest struct {
	ReservationID string    `json:"reservation_id"`
	CheckInAt     time.Time `json:"check_in_at"`
	CheckOutAt    time.Time `json:"check_out_at"`
}

type GenerateInvoiceRequest struct {
	ReservationID  string `json:"reservation_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type ApplyOvernightFeeRequest struct {
	ReservationID string `json:"reservation_id"`
	Amount        int64  `json:"amount"` // configurable overnight fee amount; falls back to pricing.OvernightPerNight if 0
}
