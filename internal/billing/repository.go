package billing

import (
	"context"

	"parkir-pintar/internal/billing/model"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks parkir-pintar/internal/billing Repository
type Repository interface {
	CreateBillingRecord(ctx context.Context, record *model.BillingRecord) error
	GetByReservationID(ctx context.Context, reservationID string) (*model.BillingRecord, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*model.BillingRecord, error)
	UpdateBillingRecord(ctx context.Context, record *model.BillingRecord) error
}
