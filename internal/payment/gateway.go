package payment

import (
	"context"

	"parkir-pintar/internal/payment/gateway"
)

//go:generate mockgen -destination=mocks/mock_payment_gateway.go -package=mocks parkir-pintar/internal/payment Gateway
type Gateway interface {
	Charge(ctx context.Context, amount int64, method string) (transactionRef string, err error)
	Refund(ctx context.Context, transactionRef string) error
	GetStatus(ctx context.Context, transactionRef string) (string, error)
}

//go:generate mockgen -destination=mocks/mock_event_publisher.go -package=mocks parkir-pintar/internal/payment EventPublisher
type EventPublisher interface {
	PublishPaymentSuccess(ctx context.Context, event gateway.PaymentResultEvent) error
	PublishPaymentFailed(ctx context.Context, event gateway.PaymentResultEvent) error
}
