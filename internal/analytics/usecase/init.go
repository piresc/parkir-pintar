package usecase

import (
	"parkir-pintar/internal/analytics/repository"
)

type analyticsUsecase struct {
	repo repository.Repository
}

func NewUsecase(repo repository.Repository) Usecase {
	return &analyticsUsecase{repo: repo}
}
