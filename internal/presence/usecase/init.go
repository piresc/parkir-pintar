package usecase

import (
	"parkir-pintar/internal/presence/repository"
)

type presenceUsecase struct {
	sensor repository.SensorGateway
}

func NewUsecase(sensor repository.SensorGateway) Usecase {
	return &presenceUsecase{
		sensor: sensor,
	}
}
