// Package errors defines domain-specific sentinel errors for the reservation service.
package constants

import "errors"

// Reservation lifecycle errors.
var (
	ErrNotFound          = errors.New("reservation not found")
	ErrConflict          = errors.New("reservation conflict")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrSpotUnavailable   = errors.New("spot unavailable")
	ErrAlreadyActive     = errors.New("driver already has an active reservation")
	ErrSpotLocked        = errors.New("spot is being reserved by another driver")
	ErrForbidden         = errors.New("reservation belongs to another driver")
	ErrPaymentFailed     = errors.New("payment processing failed")
	ErrBillingFailed     = errors.New("billing record creation failed")
	ErrNotPending        = errors.New("reservation is not pending payment")
	ErrNotCheckedOut     = errors.New("reservation is not checked out")
	ErrMissingTimestamps = errors.New("reservation missing check-in/check-out timestamps")
	ErrConcurrentChange  = errors.New("reservation status changed concurrently")
)
