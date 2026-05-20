// Package errors defines domain-specific sentinel errors for the search service.
package errors

import "errors"

// Search domain errors.
var (
	ErrSpotNotFound      = errors.New("spot not found")
	ErrFloorNotFound     = errors.New("floor not found")
	ErrNoAvailability    = errors.New("no availability for requested vehicle type")
	ErrCacheUnavailable  = errors.New("cache unavailable")
	ErrInvalidVehicleType = errors.New("invalid vehicle type")
)
