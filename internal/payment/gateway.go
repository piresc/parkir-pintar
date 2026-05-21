package payment

import (
	"context"

	"parkir-pintar/internal/payment/gateway"
)

//go:generate mockgen -destination=mocks/mock_event_publisher.go -package=mocks parkir-pintar/internal/payment EventPublisher
type EventPublisher interface {
	PublishPaymentSuccess(ctx context.Context, event gateway.PaymentResultEvent) error
	PublishPaymentFailed(ctx context.Context, event gateway.PaymentResultEvent) error
}
