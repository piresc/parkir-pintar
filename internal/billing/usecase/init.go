package usecase

import (
	"parkir-pintar/internal/billing"
	"parkir-pintar/internal/billing/repository"
)

type billingUsecase struct {
	repo repository.Repository
}

func NewUsecase(repo repository.Repository) billing.Usecase {
	return &billingUsecase{
		repo: repo,
	}
}
