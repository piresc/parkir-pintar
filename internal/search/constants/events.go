package constants

import "time"

// Consumed event config.
const (
	StreamReservationSearch = "RESERVATION_SEARCH"
	ConsumerSearchSpot      = "search-spot-consumer"
	SubjectPatternSearch    = "reservation.search.*"
)

// SpotUpdatedEvent is consumed from the reservation service.
type SpotUpdatedEvent struct {
	SpotID      string    `json:"spot_id"`
	FloorNumber int       `json:"floor_number"`
	SpotNumber  int       `json:"spot_number"`
	VehicleType string    `json:"vehicle_type"`
	SpotCode    string    `json:"spot_code"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updated_at"`
}
