// Package model defines domain structs, status constants, sentinel errors,
// and request types for the payment module.
package model

import (
	"errors"
	"time"
)

// Payment status constants.
const (
	PaymentStatusPending  = "pending"
	PaymentStatusSuccess  = "success"
	PaymentStatusFailed   = "failed"
	PaymentStatusRefunded = "refunded"
)

// Sentinel errors for the payment domain.
var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrPaymentConflict = errors.New("payment conflict")
	ErrPaymentFailed   = errors.New("payment failed")
)

// Payment represents a payment domain entity.
type Payment struct {
	ID             string     `json:"id" db:"id"`
	BillingID      string     `json:"billing_id" db:"billing_id"`
	Amount         int64      `json:"amount" db:"amount"`
	PaymentMethod  string     `json:"payment_method" db:"payment_method"`
	PaymentGateway string     `json:"payment_gateway" db:"payment_gateway"`
	TransactionRef string     `json:"transaction_ref" db:"transaction_ref"`
	IdempotencyKey string     `json:"idempotency_key" db:"idempotency_key"`
	Status         string     `json:"status" db:"status"`
	PaidAt         *time.Time `json:"paid_at,omitzero" db:"paid_at"`
	CreatedAt      time.Time  `json:"created_at,omitzero" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at,omitzero" db:"updated_at"`
}

// ProcessPaymentRequest is the payload for processing a payment.
type ProcessPaymentRequest struct {
	BillingID      string `json:"billing_id"`
	Amount         int64  `json:"amount"`
	PaymentMethod  string `json:"payment_method"`
	IdempotencyKey string `json:"idempotency_key"`
}

// ProcessQRISRequest is the payload for processing a QRIS payment.
type ProcessQRISRequest struct {
	BillingID      string `json:"billing_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

// RefundPaymentRequest is the payload for refunding a payment.
type RefundPaymentRequest struct {
	PaymentID      string `json:"payment_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

// GetPaymentStatusRequest is the payload for querying payment status.
type GetPaymentStatusRequest struct {
	PaymentID string `json:"payment_id"`
}
