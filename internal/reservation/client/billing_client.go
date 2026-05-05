// Package client provides gRPC client adapters that let the reservation service
// call downstream billing and payment microservices.
package client

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	billingmodel "parkir-pintar/internal/billing/model"
	billingv1 "parkir-pintar/proto/billing/v1"
)

// BillingClient adapts a billingv1.BillingServiceClient to the
// reservation.BillingClient interface.
type BillingClient struct {
	client billingv1.BillingServiceClient
}

// NewBillingClient creates a new BillingClient adapter.
func NewBillingClient(client billingv1.BillingServiceClient) *BillingClient {
	return &BillingClient{client: client}
}

// StartBilling calls the billing service to create a billing record with the booking fee.
func (c *BillingClient) StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) error {
	_, err := c.client.StartBilling(ctx, &billingv1.StartBillingRequest{
		ReservationId:  reservationID,
		BookingFee:     bookingFee,
		IdempotencyKey: idempotencyKey,
	})
	return err
}

// CalculateFee calls the billing service to compute parking fees.
func (c *BillingClient) CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error) {
	resp, err := c.client.CalculateFee(ctx, &billingv1.CalculateFeeRequest{
		ReservationId: reservationID,
		CheckInAt:     timestamppb.New(checkInAt),
		CheckOutAt:    timestamppb.New(checkOutAt),
	})
	if err != nil {
		return nil, err
	}
	return protoToBillingRecord(resp), nil
}

// GenerateInvoice calls the billing service to finalise a billing record into an invoice.
func (c *BillingClient) GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	resp, err := c.client.GenerateInvoice(ctx, &billingv1.GenerateInvoiceRequest{
		ReservationId:  reservationID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return protoToBillingRecord(resp), nil
}

// ApplyPenalty calls the billing service to record a penalty.
func (c *BillingClient) ApplyPenalty(ctx context.Context, reservationID string, penaltyType string, amount int64, description string) error {
	_, err := c.client.ApplyPenalty(ctx, &billingv1.ApplyPenaltyRequest{
		ReservationId: reservationID,
		PenaltyType:   penaltyType,
		Amount:        amount,
		Description:   description,
	})
	return err
}

// protoToBillingRecord maps a protobuf BillingResponse to the domain BillingRecord.
func protoToBillingRecord(r *billingv1.BillingResponse) *billingmodel.BillingRecord {
	if r == nil {
		return nil
	}
	return &billingmodel.BillingRecord{
		ID:              r.GetId(),
		ReservationID:   r.GetReservationId(),
		BookingFee:      r.GetBookingFee(),
		ParkingFee:      r.GetParkingFee(),
		OvernightFee:    r.GetOvernightFee(),
		CancellationFee: r.GetCancellationFee(),
		PenaltyAmount:   r.GetPenaltyAmount(),
		TotalAmount:     r.GetTotalAmount(),
		DurationMinutes: int(r.GetDurationMinutes()),
		BilledHours:     int(r.GetBilledHours()),
		IsOvernight:     r.GetIsOvernight(),
		IdempotencyKey:  r.GetIdempotencyKey(),
		Status:          r.GetStatus(),
	}
}
