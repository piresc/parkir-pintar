package asynq

// Task type constants.
const (
	TypeReservationExpire  = "task:reservation:expire"
	TypePaymentHoldTimeout = "task:payment:hold_timeout"
)

// ReservationExpiryPayload is the payload for reservation expiry tasks.
type ReservationExpiryPayload struct {
	ReservationID string `json:"reservation_id"`
}

// PaymentHoldTimeoutPayload is the payload for payment hold timeout tasks.
type PaymentHoldTimeoutPayload struct {
	ReservationID string `json:"reservation_id"`
	PaymentID     string `json:"payment_id"`
}
