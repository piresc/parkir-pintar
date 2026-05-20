package usecase

import (
	"parkir-pintar/internal/analytics"
	"parkir-pintar/internal/analytics/repository"
)

type analyticsUsecase struct {
	repo repository.Repository
}

func NewUsecase(repo repository.Repository) analytics.Usecase {
	return &analyticsUsecase{repo: repo}
}
