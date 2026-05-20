// Package errors defines domain-specific sentinel errors for the presence service.
package errors

import "errors"

// Presence domain errors.
var (
	ErrSensorUnavailable = errors.New("sensor unavailable")
	ErrSpotNotOccupied   = errors.New("spot not occupied: driver may be at wrong spot")
	ErrInvalidSpot       = errors.New("invalid spot coordinates")
	ErrVerificationFailed = errors.New("presence verification failed")
)
