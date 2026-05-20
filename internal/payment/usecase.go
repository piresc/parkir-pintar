package payment

import (
	"context"

	"parkir-pintar/internal/payment/model"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks parkir-pintar/internal/payment Usecase
type Usecase interface {
	ProcessPayment(ctx context.Context, req *model.ProcessPaymentRequest) (*model.Payment, error)
	ProcessQRIS(ctx context.Context, req *model.ProcessQRISRequest) (*model.Payment, error)
	RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.Payment, error)
	GetPaymentStatus(ctx context.Context, req *model.GetPaymentStatusRequest) (*model.Payment, error)
}
