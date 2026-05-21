package events

import "time"

type SpotUpdatedEvent struct {
	SpotID      string    `json:"spot_id"`
	FloorNumber int       `json:"floor_number"`
	SpotNumber  int       `json:"spot_number"`
	VehicleType string    `json:"vehicle_type"`
	SpotCode    string    `json:"spot_code"`
	Status      string    `json:"status"` // available, reserved, occupied
	UpdatedAt   time.Time `json:"updated_at"`
}

type PaymentResultEvent struct {
	PaymentID     string    `json:"payment_id"`
	ReservationID string    `json:"reservation_id"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`           // success, failed
	Reason        string    `json:"reason,omitempty"` // failure reason
	Timestamp     time.Time `json:"timestamp"`
}

type ReservationEvent struct {
	ReservationID string    `json:"reservation_id"`
	DriverID      string    `json:"driver_id"`
	SpotID        string    `json:"spot_id"`
	VehicleType   string    `json:"vehicle_type"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}
