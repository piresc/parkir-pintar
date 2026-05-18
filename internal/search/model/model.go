// Package model defines domain structs and request types for the search module.
package model

// FloorAvailability represents the availability breakdown for a single floor.
type FloorAvailability struct {
	FloorNumber   int `json:"floor_number" db:"floor_number"`
	AvailableCar  int `json:"available_car" db:"available_car"`
	AvailableMoto int `json:"available_moto" db:"available_moto"`
	TotalCar      int `json:"total_car" db:"total_car"`
	TotalMoto     int `json:"total_moto" db:"total_moto"`
}

// AvailabilitySummary represents the overall parking availability totals.
type AvailabilitySummary struct {
	TotalAvailable int `json:"total_available"`
	TotalCapacity  int `json:"total_capacity"`
}

// SpotDetails represents the details of a single parking spot.
type SpotDetails struct {
	ID          string   `json:"id" db:"id"`
	SpotCode    string   `json:"spot_code" db:"spot_code"`
	FloorNumber int      `json:"floor_number" db:"floor_number"`
	SpotNumber  int      `json:"spot_number" db:"spot_number"`
	VehicleType string   `json:"vehicle_type" db:"vehicle_type"`
	Status      string   `json:"status" db:"status"`
	Latitude    *float64 `json:"latitude,omitempty" db:"latitude"`
	Longitude   *float64 `json:"longitude,omitempty" db:"longitude"`
}

// GetAvailabilityRequest is the payload for querying parking availability.
type GetAvailabilityRequest struct {
	VehicleType string `json:"vehicle_type"`
}

// GetFloorMapRequest is the payload for querying a specific floor's spot map.
type GetFloorMapRequest struct {
	FloorNumber int `json:"floor_number"`
}

// GetSpotDetailsRequest is the payload for querying a specific spot's details.
type GetSpotDetailsRequest struct {
	SpotID string `json:"spot_id"`
}
