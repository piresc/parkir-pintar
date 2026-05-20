package billing

import (
	"context"

	"parkir-pintar/internal/billing/model"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks parkir-pintar/internal/billing Usecase
type Usecase interface {
	StartBilling(ctx context.Context, req *model.StartBillingRequest) (*model.BillingRecord, error)
	CalculateFee(ctx context.Context, req *model.CalculateFeeRequest) (*model.BillingRecord, error)
	GenerateInvoice(ctx context.Context, req *model.GenerateInvoiceRequest) (*model.BillingRecord, error)
	ApplyOvernightFee(ctx context.Context, req *model.ApplyOvernightFeeRequest) (*model.BillingRecord, error)
}
