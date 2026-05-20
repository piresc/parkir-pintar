package usecase

import (
	"parkir-pintar/internal/presence"
	"parkir-pintar/internal/presence/repository"
)

type presenceUsecase struct {
	sensor repository.SensorGateway
}

func NewUsecase(sensor repository.SensorGateway) presence.Usecase {
	return &presenceUsecase{
		sensor: sensor,
	}
}
