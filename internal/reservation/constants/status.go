// Package constants defines shared constants for the reservation domain module.
// This package has ZERO internal imports to prevent circular dependencies.
package constants

// ReservationStatus represents the status of a reservation.
type ReservationStatus string

// Reservation status constants.
const (
	StatusPending        ReservationStatus = "pending"
	StatusWaitingPayment ReservationStatus = "waiting_payment"
	StatusConfirmed      ReservationStatus = "confirmed"
	StatusCheckedIn      ReservationStatus = "checked_in"
	StatusCheckedOut     ReservationStatus = "checked_out"
	StatusCompleted      ReservationStatus = "completed"
	StatusExpired        ReservationStatus = "expired"
	StatusCancelled      ReservationStatus = "cancelled"
	StatusFailed         ReservationStatus = "failed"
)

// AssignmentMode represents how a parking spot is assigned.
type AssignmentMode string

// Assignment mode constants.
const (
	AssignmentSystemAssigned AssignmentMode = "system_assigned"
	AssignmentUserSelected   AssignmentMode = "user_selected"
)

// SpotStatus represents the status of a parking spot.
type SpotStatus string

// Spot status constants.
const (
	SpotStatusAvailable SpotStatus = "available"
	SpotStatusReserved  SpotStatus = "reserved"
	SpotStatusOccupied  SpotStatus = "occupied"
)

// PaymentMethod represents a payment method type.

// PaymentEventStatus represents the status field in payment NATS events.
type PaymentEventStatus string

// Payment event status constants (published by payment service).
const (
	PaymentEventSuccess PaymentEventStatus = "success"
	PaymentEventFailed  PaymentEventStatus = "failed"
)

// Payment method constants.
type PaymentMethod string

const (
	PaymentMethodQRIS PaymentMethod = "qris"
)
