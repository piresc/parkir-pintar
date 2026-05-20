// Package constants defines shared constants for the payment domain module.
package constants

// PaymentStatus represents the status of a payment.
type PaymentStatus string

// Payment status constants.
const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusSuccess  PaymentStatus = "success"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// PaymentMethod represents a payment method type.
type PaymentMethod string

// Payment method constants.
const (
	PaymentMethodQRIS PaymentMethod = "qris"
)
