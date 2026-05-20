package model

type FloorAvailability struct {
	FloorNumber   int `json:"floor_number" db:"floor_number"`
	AvailableCar  int `json:"available_car" db:"available_car"`
	AvailableMoto int `json:"available_moto" db:"available_moto"`
	TotalCar      int `json:"total_car" db:"total_car"`
	TotalMoto     int `json:"total_moto" db:"total_moto"`
}

type AvailabilitySummary struct {
	TotalAvailable int `json:"total_available"`
	TotalCapacity  int `json:"total_capacity"`
}

type SpotDetails struct {
	ID          string `json:"id" db:"id"`
	SpotCode    string `json:"spot_code" db:"spot_code"`
	FloorNumber int    `json:"floor_number" db:"floor_number"`
	SpotNumber  int    `json:"spot_number" db:"spot_number"`
	VehicleType string `json:"vehicle_type" db:"vehicle_type"`
	Status      string `json:"status" db:"status"`
}

type GetAvailabilityRequest struct {
	VehicleType string `json:"vehicle_type"`
}

type GetFloorMapRequest struct {
	FloorNumber int `json:"floor_number"`
}

type GetSpotDetailsRequest struct {
	SpotID string `json:"spot_id"`
}
