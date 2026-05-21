package events

// NATS subject constants for inter-service communication.
const (
	// Reservation → Search
	SubjectReservationSearchSpotUpdated = "reservation.search.spot-updated"

	// Reservation → Analytics
	SubjectReservationAnalyticsCreated   = "reservation.analytics.created"
	SubjectReservationAnalyticsConfirmed = "reservation.analytics.confirmed"
	SubjectReservationAnalyticsCheckedIn = "reservation.analytics.checked-in"
	SubjectReservationAnalyticsCompleted = "reservation.analytics.completed"
	SubjectReservationAnalyticsCancelled = "reservation.analytics.cancelled"
	SubjectReservationAnalyticsExpired   = "reservation.analytics.expired"
	SubjectReservationAnalyticsFailed    = "reservation.analytics.failed"

	// Payment → Reservation
	SubjectPaymentReservationSuccess = "payment.reservation.success"
	SubjectPaymentReservationFailed  = "payment.reservation.failed"
)
