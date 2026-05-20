package constants

// Pricing constants — all monetary values in IDR.
const (
	// BookingFee is the non-refundable fee charged at confirmation.
	BookingFee int64 = 5_000

	// HourlyRate is the per-started-hour parking rate (minimum 1 hour).
	HourlyRate int64 = 5_000

	// OvernightPerNight is the fee per midnight crossed in WIB (UTC+7).
	OvernightPerNight int64 = 20_000
)
