package constants

import "time"

// Default timeout values for the reservation lifecycle.
const (
	// DefaultExpiryTimeout is the default time a confirmed reservation remains
	// valid before expiring (driver must check in within this window).
	DefaultExpiryTimeout = 60 * time.Minute

	// DefaultPaymentTimeout is the default time a waiting_payment reservation
	// remains valid before being failed (driver must complete payment within this window).
	DefaultPaymentTimeout = 10 * time.Minute

	// DefaultConfirmationExpiry is the time window after confirmation during
	// which the driver must check in.
	DefaultConfirmationExpiry = 1 * time.Hour
)
