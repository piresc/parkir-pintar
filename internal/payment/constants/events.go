package constants

import "time"

// Published event subjects.
const (
	SubjectPaymentSuccess = "payment.reservation.success"
	SubjectPaymentFailed  = "payment.reservation.failed"
)

// NATS stream config for publishing.
const (
	StreamPaymentReservation = "PAYMENT_RESERVATION"
	SubjectPatternPayment    = "payment.reservation.*"
)

// PaymentResultEvent is published when a payment completes.
type PaymentResultEvent struct {
	PaymentID     string    `json:"payment_id"`
	ReservationID string    `json:"reservation_id"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
	Reason        string    `json:"reason,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}
