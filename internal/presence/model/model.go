// Package model defines domain structs and geofence detection functions
// for the presence module.
package model

import (
	"math"
	"time"
)

// earthRadiusMeters is the mean radius of the Earth in meters, used for
// Haversine distance calculations.
const earthRadiusMeters = 6_371_000.0

// PresenceLog represents a persisted location record for a reservation.
type PresenceLog struct {
	ID            string    `json:"id" db:"id"`
	ReservationID string    `json:"reservation_id" db:"reservation_id"`
	Latitude      float64   `json:"latitude" db:"latitude"`
	Longitude     float64   `json:"longitude" db:"longitude"`
	Accuracy      float64   `json:"accuracy" db:"accuracy"`
	RecordedAt    time.Time `json:"recorded_at" db:"recorded_at"`
}

// LocationUpdate represents a real-time location update from a driver.
type LocationUpdate struct {
	ReservationID string    `json:"reservation_id"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Accuracy      float64   `json:"accuracy"`
	Timestamp     time.Time `json:"timestamp"`
}

// ArrivalResult holds the result of a geofence arrival detection check.
type ArrivalResult struct {
	Arrived       bool      `json:"arrived"`
	ReservationID string    `json:"reservation_id"`
	DetectedAt    time.Time `json:"detected_at"`
}

// WrongSpotResult holds the result of a wrong-spot detection check.
type WrongSpotResult struct {
	IsWrongSpot    bool    `json:"is_wrong_spot"`
	DistanceMeters float64 `json:"distance_meters"`
}

// DetectArrival checks if a driver's location is within the parking area
// geofence. Returns true if the Haversine distance between the driver's
// position and the geofence center is less than or equal to radiusMeters.
func DetectArrival(lat, lng, centerLat, centerLng, radiusMeters float64) bool {
	return HaversineDistance(lat, lng, centerLat, centerLng) <= radiusMeters
}

// HaversineDistance calculates the great-circle distance in meters between
// two GPS coordinates using the Haversine formula with Earth radius 6,371,000m.
//
// Preconditions:
//   - All inputs are valid latitude/longitude in degrees
//
// Postconditions:
//   - Returns distance in meters (>= 0)
//   - Returns 0 for identical coordinates
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := toRadians(lat2 - lat1)
	dLng := toRadians(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

// toRadians converts degrees to radians.
func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
