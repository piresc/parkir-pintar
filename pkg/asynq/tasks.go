package asynq

const (
	TypeReservationExpire  = "task:reservation:expire"
	TypePaymentHoldTimeout = "task:payment:hold_timeout"
)

type ReservationExpiryPayload struct {
	ReservationID string `json:"reservation_id"`
}

type PaymentHoldTimeoutPayload struct {
	ReservationID string `json:"reservation_id"`
	PaymentID     string `json:"payment_id"`
}
