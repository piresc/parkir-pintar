// Package pricing implements the ParkirPintar fee calculation engine.
//
// Pricing rules (from PRD):
//   - Booking fee: 5,000 IDR (non-refundable, charged at confirmation)
//   - Hourly rate: 5,000 IDR per started hour (minimum 1 hour)
//   - Overnight fee: 20,000 IDR per midnight crossed in WIB (UTC+7)
//   - Cancellation: free within 2 minutes, 5,000 IDR after
//   - No-show: forfeits booking fee only (no additional charge)
//   - Wrong-spot penalty: 200,000 IDR (not in scope for detection)
package pricing

import "time"

// Pricing constants — all monetary values in IDR.
const (
	BookingFee        int64 = 5_000
	HourlyRate        int64 = 5_000
	OvernightPerNight int64 = 20_000
	WrongSpotPenalty  int64 = 200_000
	CancelFee         int64 = 5_000
	NoShowFee         int64 = 0

	CancelFreeWindow = 2 * time.Minute
)

// wib is the WIB timezone (UTC+7) used for midnight detection.
var wib = time.FixedZone("WIB", 7*60*60)

// FeeResult holds the computed parking fee breakdown for a session.
type FeeResult struct {
	ParkingFee      int64
	OvernightFee    int64
	NightsCrossed   int
	DurationMinutes int
	BilledHours     int
	IsOvernight     bool
}

// CalculateSessionFee computes parking + overnight fees for a completed session.
//
// Rules:
//   - ParkingFee = ceil(duration_hours) × HourlyRate (minimum 1 hour)
//   - OvernightFee = OvernightPerNight × number_of_midnights_crossed_in_WIB
//   - A midnight is crossed when the WIB calendar date changes between checkIn and checkOut
func CalculateSessionFee(checkIn, checkOut time.Time) FeeResult {
	duration := checkOut.Sub(checkIn)
	durationMinutes := int(duration.Minutes())

	// Ceiling-based hour calculation (minimum 1 hour)
	billedHours := int(duration.Hours())
	if duration > time.Duration(billedHours)*time.Hour {
		billedHours++
	}
	billedHours = max(billedHours, 1)

	parkingFee := int64(billedHours) * HourlyRate

	// Count midnights crossed in WIB
	nightsCrossed := countMidnightsCrossed(checkIn, checkOut)
	overnightFee := OvernightPerNight * int64(nightsCrossed)

	return FeeResult{
		ParkingFee:      parkingFee,
		OvernightFee:    overnightFee,
		NightsCrossed:   nightsCrossed,
		DurationMinutes: durationMinutes,
		BilledHours:     billedHours,
		IsOvernight:     nightsCrossed > 0,
	}
}

// countMidnightsCrossed returns the number of midnight boundaries crossed
// between start and end in WIB timezone.
//
// Example: check-in 23:00 Day1, check-out 01:00 Day3 → 2 midnights crossed.
func countMidnightsCrossed(start, end time.Time) int {
	s := start.In(wib)
	e := end.In(wib)

	sDay := time.Date(s.Year(), s.Month(), s.Day(), 0, 0, 0, 0, wib)
	eDay := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, wib)

	days := int(eDay.Sub(sDay).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// CalculateCancellationFee determines the cancellation fee based on elapsed
// time since confirmation.
//
// Returns 0 if cancelled within CancelFreeWindow (2 minutes), else CancelFee.
func CalculateCancellationFee(confirmedAt, cancelledAt time.Time) int64 {
	if cancelledAt.Sub(confirmedAt) <= CancelFreeWindow {
		return 0
	}
	return CancelFee
}

// CalculateTotal computes the total billing amount from individual components.
func CalculateTotal(bookingFee, parkingFee, overnightFee, cancellationFee, penaltyAmount int64) int64 {
	return bookingFee + parkingFee + overnightFee + cancellationFee + penaltyAmount
}
