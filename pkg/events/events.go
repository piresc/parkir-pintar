// Package events defines shared event structs used across microservices
// for NATS JetStream messaging. These are the canonical definitions —
// publishers and consumers both import from here.
package events

import "time"

// SpotUpdatedEvent is published when a parking spot's status changes.
// Publisher: reservation service. Consumer: search service.
type SpotUpdatedEvent struct {
	SpotID      string    `json:"spot_id"`
	FloorNumber int       `json:"floor_number"`
	SpotNumber  int       `json:"spot_number"`
	VehicleType string    `json:"vehicle_type"`
	SpotCode    string    `json:"spot_code"`
	Status      string    `json:"status"` // available, reserved, occupied
	UpdatedAt   time.Time `json:"updated_at"`
}

// PaymentResultEvent is published when a payment completes or fails.
// Publisher: payment service. Consumer: reservation service.
type PaymentResultEvent struct {
	PaymentID     string    `json:"payment_id"`
	ReservationID string    `json:"reservation_id"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`           // success, failed
	Reason        string    `json:"reason,omitempty"` // failure reason
	Timestamp     time.Time `json:"timestamp"`
}

// ReservationEvent is published on reservation lifecycle transitions.
// Publisher: reservation service. Consumer: analytics service.
type ReservationEvent struct {
	ReservationID string    `json:"reservation_id"`
	DriverID      string    `json:"driver_id"`
	SpotID        string    `json:"spot_id"`
	VehicleType   string    `json:"vehicle_type"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}
