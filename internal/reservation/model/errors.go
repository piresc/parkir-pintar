package model

import "errors"

// Sentinel errors for the reservation domain.
// Use errors.Is() to check for these in handlers and tests.
var (
	ErrNotFound          = errors.New("reservation not found")
	ErrConflict          = errors.New("reservation conflict")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrSpotUnavailable   = errors.New("spot unavailable")
)
