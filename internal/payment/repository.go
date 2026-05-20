package payment

import (
	"context"

	"parkir-pintar/internal/payment/model"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks parkir-pintar/internal/payment Repository
type Repository interface {
	CreatePayment(ctx context.Context, payment *model.Payment) error
	GetByIdempotencyKey(ctx context.Context, key string) (*model.Payment, error)
	UpdatePayment(ctx context.Context, payment *model.Payment) error
	UpdatePaymentWithStatusCheck(ctx context.Context, payment *model.Payment, expectedStatus string) error
	GetByID(ctx context.Context, id string) (*model.Payment, error)
	GetByBillingID(ctx context.Context, billingID string) (*model.Payment, error)
}
