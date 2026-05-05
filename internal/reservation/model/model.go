// Package model defines domain structs and state machine for the reservation module.
package model

import (
	"fmt"
	"slices"
	"time"
)

// Reservation status constants.
const (
	StatusPending    = "pending"
	StatusConfirmed  = "confirmed"
	StatusCheckedIn  = "checked_in"
	StatusCheckedOut = "checked_out"
	StatusExpired    = "expired"
	StatusCancelled  = "cancelled"
)

// Assignment mode constants.
const (
	AssignmentSystemAssigned = "system_assigned"
	AssignmentUserSelected   = "user_selected"
)

// Reservation represents a parking reservation domain entity.
type Reservation struct {
	ID             string     `json:"id" db:"id"`
	DriverID       string     `json:"driver_id" db:"driver_id"`
	SpotID         string     `json:"spot_id" db:"spot_id"`
	VehicleType    string     `json:"vehicle_type" db:"vehicle_type"`
	AssignmentMode string     `json:"assignment_mode" db:"assignment_mode"`
	Status         string     `json:"status" db:"status"`
	IdempotencyKey string     `json:"idempotency_key" db:"idempotency_key"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitzero" db:"confirmed_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitzero" db:"expires_at"`
	CheckedInAt    *time.Time `json:"checked_in_at,omitzero" db:"checked_in_at"`
	CheckedOutAt   *time.Time `json:"checked_out_at,omitzero" db:"checked_out_at"`
	CancelledAt    *time.Time `json:"cancelled_at,omitzero" db:"cancelled_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// ParkingSpot represents a parking spot domain entity.
type ParkingSpot struct {
	ID          string    `json:"id" db:"id"`
	FloorNumber int       `json:"floor_number" db:"floor_number"`
	SpotNumber  int       `json:"spot_number" db:"spot_number"`
	VehicleType string    `json:"vehicle_type" db:"vehicle_type"`
	SpotCode    string    `json:"spot_code" db:"spot_code"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Driver represents a driver domain entity.
type Driver struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Phone        string    `json:"phone" db:"phone"`
	Email        string    `json:"email" db:"email"`
	VehicleType  string    `json:"vehicle_type" db:"vehicle_type"`
	VehiclePlate string    `json:"vehicle_plate" db:"vehicle_plate"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// allowedTransitions defines the valid reservation state transitions.
// Terminal states (checked_out, expired, cancelled) have no outgoing transitions.
var allowedTransitions = map[string][]string{
	StatusPending:   {StatusConfirmed},
	StatusConfirmed: {StatusCheckedIn, StatusExpired, StatusCancelled},
	StatusCheckedIn: {StatusCheckedOut},
}

// ValidateTransition checks if a reservation status transition is allowed.
// Returns nil if the transition is valid, or an error describing why it is not.
func ValidateTransition(from, to string) error {
	targets, ok := allowedTransitions[from]
	if !ok {
		return fmt.Errorf("%w: no transitions from terminal state %q", ErrInvalidTransition, from)
	}

	if slices.Contains(targets, to) {
		return nil
	}
	return fmt.Errorf("%w: invalid transition from %q to %q", ErrInvalidTransition, from, to)
}

// CreateReservationRequest is the payload for creating a new reservation.
type CreateReservationRequest struct {
	DriverID       string `json:"driver_id"`
	VehicleType    string `json:"vehicle_type"`
	AssignmentMode string `json:"assignment_mode"`
	SpotID         string `json:"spot_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}

// CancelReservationRequest is the payload for cancelling a reservation.
type CancelReservationRequest struct {
	ReservationID string `json:"reservation_id"`
}

// CheckInRequest is the payload for checking in to a reservation.
type CheckInRequest struct {
	ReservationID string `json:"reservation_id"`
}

// CheckOutRequest is the payload for checking out of a reservation.
type CheckOutRequest struct {
	ReservationID string `json:"reservation_id"`
}

// CheckOutResponse contains the checkout result with billing details.
type CheckOutResponse struct {
	Reservation *Reservation `json:"reservation"`
	TotalAmount int64        `json:"total_amount"`
	BillingID   string       `json:"billing_id"`
}

// ExpireReservationRequest is the payload for expiring a reservation.
type ExpireReservationRequest struct {
	ReservationID string `json:"reservation_id"`
}
