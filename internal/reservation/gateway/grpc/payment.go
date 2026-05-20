package grpcgw

import (
	"context"
	"errors"
	"time"

	"parkir-pintar/pkg/apperror"
	"parkir-pintar/pkg/circuitbreaker"
	paymentv1 "parkir-pintar/proto/payment/v1"
)

type PaymentClient struct {
	client paymentv1.PaymentServiceClient
	cb     *circuitbreaker.CircuitBreaker
}

func NewPaymentClient(client paymentv1.PaymentServiceClient) *PaymentClient {
	return &PaymentClient{
		client: client,
		cb: circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold:  5,
			OpenTimeout:       30 * time.Second,
			HalfOpenMaxProbes: 1,
		}),
	}
}

func (c *PaymentClient) ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error) {
	var result string
	err := c.cb.Execute(func() error {
		var err error
		result, err = c.processPaymentInner(ctx, billingID, amount, paymentMethod, idempotencyKey)
		return err
	})
	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		return "", apperror.ServiceUnavailable("payment service temporarily unavailable")
	}
	return result, err
}

func (c *PaymentClient) processPaymentInner(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error) {
	resp, err := c.client.ProcessPayment(ctx, &paymentv1.ProcessPaymentRequest{
		BillingId:      billingID,
		Amount:         amount,
		PaymentMethod:  paymentMethod,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return "", err
	}
	return resp.GetId(), nil
}

func (c *PaymentClient) RefundPayment(ctx context.Context, paymentID string) error {
	err := c.cb.Execute(func() error {
		_, err := c.client.RefundPayment(ctx, &paymentv1.RefundPaymentRequest{
			PaymentId: paymentID,
		})
		return err
	})
	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		return apperror.ServiceUnavailable("payment service temporarily unavailable")
	}
	return err
}
