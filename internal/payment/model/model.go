package model

import (
	"errors"
	"time"

	"parkir-pintar/internal/payment/constants"
)

// Re-export typed constants for backward compatibility with existing consumers.
const (
	PaymentStatusPending  = constants.PaymentStatusPending
	PaymentStatusSuccess  = constants.PaymentStatusSuccess
	PaymentStatusFailed   = constants.PaymentStatusFailed
	PaymentStatusRefunded = constants.PaymentStatusRefunded
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrPaymentConflict = errors.New("payment conflict")
	ErrPaymentFailed   = errors.New("payment failed")
)

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

type ProcessPaymentRequest struct {
	BillingID      string `json:"billing_id"`
	Amount         int64  `json:"amount"`
	PaymentMethod  string `json:"payment_method"`
	IdempotencyKey string `json:"idempotency_key"`
}

type ProcessQRISRequest struct {
	BillingID      string `json:"billing_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

type RefundPaymentRequest struct {
	PaymentID      string `json:"payment_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type GetPaymentStatusRequest struct {
	PaymentID string `json:"payment_id"`
}
