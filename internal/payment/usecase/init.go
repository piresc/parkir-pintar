// - Never ignore errors; always handle them explicitly
package usecase

import (
	"parkir-pintar/internal/payment"
	"parkir-pintar/internal/payment/gateway"
	"parkir-pintar/internal/payment/repository"
)

type paymentUsecase struct {
	repo           repository.Repository
	gw             gateway.PaymentGateway
	eventPublisher EventPublisher
}

func NewUsecase(repo repository.Repository, gw gateway.PaymentGateway, pub EventPublisher) payment.Usecase {
	return &paymentUsecase{
		repo:           repo,
		gw:             gw,
		eventPublisher: pub,
	}
}
