// Package errors defines domain-specific sentinel errors for the payment service.
package errors

import "errors"

// Payment domain errors.
var (
	ErrNotFound       = errors.New("payment not found")
	ErrConflict       = errors.New("payment conflict: duplicate")
	ErrStatusMismatch = errors.New("payment status mismatch: concurrent modification")
	ErrCannotRefund   = errors.New("cannot refund: payment not in success status")
	ErrGatewayFailed  = errors.New("payment gateway failed")
	ErrCancelled      = errors.New("payment processing cancelled")
	ErrRefundFailed   = errors.New("refund failed: all retries exhausted")
)
