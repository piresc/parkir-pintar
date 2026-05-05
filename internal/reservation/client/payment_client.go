package client

import (
	"context"

	paymentv1 "parkir-pintar/proto/payment/v1"
)

// PaymentClient adapts a paymentv1.PaymentServiceClient to the
// reservation.PaymentClient interface.
type PaymentClient struct {
	client paymentv1.PaymentServiceClient
}

// NewPaymentClient creates a new PaymentClient adapter.
func NewPaymentClient(client paymentv1.PaymentServiceClient) *PaymentClient {
	return &PaymentClient{client: client}
}

// ProcessPayment calls the payment service to process a payment for a billing record.
func (c *PaymentClient) ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error) {
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
