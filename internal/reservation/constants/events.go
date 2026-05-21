package constants

import "time"

// Published event subjects.
const (
	SubjectSearchSpotUpdated  = "reservation.search.spot-updated"
	SubjectAnalyticsCreated   = "reservation.analytics.created"
	SubjectAnalyticsConfirmed = "reservation.analytics.confirmed"
	SubjectAnalyticsCheckedIn = "reservation.analytics.checked-in"
	SubjectAnalyticsCompleted = "reservation.analytics.completed"
	SubjectAnalyticsCancelled = "reservation.analytics.cancelled"
	SubjectAnalyticsExpired   = "reservation.analytics.expired"
	SubjectAnalyticsFailed    = "reservation.analytics.failed"
)

// Consumed event subjects.
const (
	SubjectPaymentSuccess = "payment.reservation.success"
	SubjectPaymentFailed  = "payment.reservation.failed"
)

// NATS stream/consumer config for this service.
const (
	StreamPaymentReservation   = "PAYMENT_RESERVATION"
	ConsumerReservationPayment = "reservation-payment-consumer"
	SubjectPatternPayment      = "payment.reservation.*"
)

// SpotUpdatedEvent is published when a spot's status changes.
type SpotUpdatedEvent struct {
	SpotID      string    `json:"spot_id"`
	FloorNumber int       `json:"floor_number"`
	SpotNumber  int       `json:"spot_number"`
	VehicleType string    `json:"vehicle_type"`
	SpotCode    string    `json:"spot_code"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReservationEvent is published for analytics tracking.
type ReservationEvent struct {
	ReservationID string    `json:"reservation_id"`
	DriverID      string    `json:"driver_id"`
	SpotID        string    `json:"spot_id"`
	VehicleType   string    `json:"vehicle_type"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}

// PaymentResultEvent is consumed from the payment service.
type PaymentResultEvent struct {
	PaymentID     string    `json:"payment_id"`
	ReservationID string    `json:"reservation_id"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
	Reason        string    `json:"reason,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}
