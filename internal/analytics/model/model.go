package model

import "time"

type PeakHourStats struct {
	Hour            int     `json:"hour" db:"hour"`
	DayOfWeek       int     `json:"day_of_week" db:"day_of_week"`
	AvgOccupancy    float64 `json:"avg_occupancy" db:"avg_occupancy"`
	AvgReservations int     `json:"avg_reservations" db:"avg_reservations"`
	PeakScore       float64 `json:"peak_score" db:"peak_score"`
}

type ResourcePrediction struct {
	Timestamp            time.Time `json:"timestamp"`
	PredictedOccupancy   float64   `json:"predicted_occupancy"`
	RecommendedInstances int       `json:"recommended_instances"`
	Confidence           float64   `json:"confidence"`
}

type UsagePattern struct {
	Period         string  `json:"period"`
	AvgUtilization float64 `json:"avg_utilization"`
	PeakHours      []int   `json:"peak_hours"`
	IdleHours      []int   `json:"idle_hours"`
}

type DailyOccupancy struct {
	Date          time.Time `json:"date" db:"date"`
	AvgOccupancy  float64   `json:"avg_occupancy" db:"avg_occupancy"`
	TotalSpots    int       `json:"total_spots" db:"total_spots"`
	OccupiedSpots int       `json:"occupied_spots" db:"occupied_spots"`
}

type ReservationEvent struct {
	ReservationID string    `json:"reservation_id" db:"reservation_id"`
	DriverID      string    `json:"driver_id" db:"driver_id"`
	SpotID        string    `json:"spot_id" db:"spot_id"`
	VehicleType   string    `json:"vehicle_type" db:"vehicle_type"`
	Status        string    `json:"status" db:"status"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
}
