package pricing

import (
	"time"
)

var wib = time.FixedZone("WIB", 7*60*60)

type FeeResult struct {
	ParkingFee      int64
	OvernightFee    int64
	NightsCrossed   int
	DurationMinutes int
	BilledHours     int
	IsOvernight     bool
}

func CalculateSessionFee(checkIn, checkOut time.Time) FeeResult {
	duration := checkOut.Sub(checkIn)
	durationMinutes := int(duration.Minutes())

	billedHours := int(duration.Hours())
	if duration > time.Duration(billedHours)*time.Hour {
		billedHours++
	}
	billedHours = max(billedHours, 1)

	parkingFee := int64(billedHours) * HourlyRate

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

func CalculateTotal(bookingFee, parkingFee, overnightFee int64) int64 {
	return bookingFee + parkingFee + overnightFee
}
