package nats

const (
	SubjectReservationSearchSpotUpdated = "reservation.search.spot-updated"

	SubjectReservationAnalyticsCreated   = "reservation.analytics.created"
	SubjectReservationAnalyticsConfirmed = "reservation.analytics.confirmed"
	SubjectReservationAnalyticsCheckedIn = "reservation.analytics.checked-in"
	SubjectReservationAnalyticsCompleted = "reservation.analytics.completed"
	SubjectReservationAnalyticsCancelled = "reservation.analytics.cancelled"
	SubjectReservationAnalyticsExpired   = "reservation.analytics.expired"
	SubjectReservationAnalyticsFailed    = "reservation.analytics.failed"

	SubjectPaymentReservationSuccess = "payment.reservation.success"
	SubjectPaymentReservationFailed  = "payment.reservation.failed"
)

const (
	StreamReservationSearch    = "RESERVATION_SEARCH"
	StreamReservationAnalytics = "RESERVATION_ANALYTICS"
	StreamPaymentReservation   = "PAYMENT_RESERVATION"
)

const (
	ConsumerSearchSpot         = "search-spot-consumer"
	ConsumerAnalytics          = "analytics-consumer"
	ConsumerReservationPayment = "reservation-payment-consumer"
)
