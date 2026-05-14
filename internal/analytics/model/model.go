// Package model defines domain structs for the analytics module including
// peak/idle hour statistics, resource predictions, and usage patterns.
package model

import "time"

// PeakHourStats represents aggregated occupancy and reservation statistics
// for a specific hour and day-of-week combination.
type PeakHourStats struct {
	Hour            int     `json:"hour" db:"hour"`
	DayOfWeek       int     `json:"day_of_week" db:"day_of_week"`
	AvgOccupancy    float64 `json:"avg_occupancy" db:"avg_occupancy"`
	AvgReservations int     `json:"avg_reservations" db:"avg_reservations"`
	PeakScore       float64 `json:"peak_score" db:"peak_score"`
}

// ResourcePrediction represents a predicted resource requirement at a future timestamp.
type ResourcePrediction struct {
	Timestamp            time.Time `json:"timestamp"`
	PredictedOccupancy   float64   `json:"predicted_occupancy"`
	RecommendedInstances int       `json:"recommended_instances"`
	Confidence           float64   `json:"confidence"`
}

// UsagePattern summarizes weekly utilization patterns including peak and idle hours.
type UsagePattern struct {
	Period         string  `json:"period"`
	AvgUtilization float64 `json:"avg_utilization"`
	PeakHours      []int   `json:"peak_hours"`
	IdleHours      []int   `json:"idle_hours"`
}

// DailyOccupancy represents the average spot occupancy for a single day.
type DailyOccupancy struct {
	Date          time.Time `json:"date" db:"date"`
	AvgOccupancy  float64   `json:"avg_occupancy" db:"avg_occupancy"`
	TotalSpots    int       `json:"total_spots" db:"total_spots"`
	OccupiedSpots int       `json:"occupied_spots" db:"occupied_spots"`
}
