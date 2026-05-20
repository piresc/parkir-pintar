// Package constants defines shared constants for the reservation domain module.
// This package has ZERO internal imports to prevent circular dependencies.
package constants

// Reservation status constants.
const (
	StatusPending        = "pending"
	StatusWaitingPayment = "waiting_payment"
	StatusConfirmed      = "confirmed"
	StatusCheckedIn      = "checked_in"
	StatusCheckedOut     = "checked_out"
	StatusCompleted      = "completed"
	StatusExpired        = "expired"
	StatusCancelled      = "cancelled"
	StatusFailed         = "failed"
)

// Assignment mode constants.
const (
	AssignmentSystemAssigned = "system_assigned"
	AssignmentUserSelected   = "user_selected"
)

// Spot status constants.
const (
	SpotStatusAvailable = "available"
	SpotStatusReserved  = "reserved"
	SpotStatusOccupied  = "occupied"
)

// Payment method constants.
const (
	PaymentMethodQRIS = "qris"
)
