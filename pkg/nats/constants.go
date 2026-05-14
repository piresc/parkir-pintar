// Package nats provides a JetStream client for inter-service messaging.
package nats

// Subjects
const (
	// Reservation -> Search
	SubjectReservationSearchSpotUpdated = "reservation.search.spot-updated"

	// Reservation -> Analytics
	SubjectReservationAnalyticsCreated   = "reservation.analytics.created"
	SubjectReservationAnalyticsConfirmed = "reservation.analytics.confirmed"
	SubjectReservationAnalyticsCheckedIn = "reservation.analytics.checked-in"
	SubjectReservationAnalyticsCompleted = "reservation.analytics.completed"
	SubjectReservationAnalyticsCancelled = "reservation.analytics.cancelled"
	SubjectReservationAnalyticsExpired   = "reservation.analytics.expired"
	SubjectReservationAnalyticsFailed    = "reservation.analytics.failed"

	// Payment -> Reservation
	SubjectPaymentReservationSuccess = "payment.reservation.success"
	SubjectPaymentReservationFailed  = "payment.reservation.failed"
)

// Stream names (one per producer->consumer pair)
const (
	StreamReservationSearch    = "RESERVATION_SEARCH"
	StreamReservationAnalytics = "RESERVATION_ANALYTICS"
	StreamPaymentReservation   = "PAYMENT_RESERVATION"
)

// Consumer names (exclusive, one consumer per stream)
const (
	ConsumerSearchSpot         = "search-spot-consumer"
	ConsumerAnalytics          = "analytics-consumer"
	ConsumerReservationPayment = "reservation-payment-consumer"
)
