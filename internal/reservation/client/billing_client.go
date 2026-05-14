package client

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/pkg/apperror"
	"parkir-pintar/pkg/circuitbreaker"
	billingv1 "parkir-pintar/proto/billing/v1"
)

type BillingClient struct {
	client billingv1.BillingServiceClient
	cb     *circuitbreaker.CircuitBreaker
}

func NewBillingClient(client billingv1.BillingServiceClient) *BillingClient {
	return &BillingClient{
		client: client,
		cb: circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold:  5,
			OpenTimeout:       30 * time.Second,
			HalfOpenMaxProbes: 1,
		}),
	}
}

func (c *BillingClient) StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	var result *billingmodel.BillingRecord
	err := c.cb.Execute(func() error {
		var err error
		result, err = c.startBillingInner(ctx, reservationID, bookingFee, idempotencyKey)
		return err
	})
	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		return nil, apperror.New("SERVICE_UNAVAILABLE", "billing service temporarily unavailable", 503)
	}
	return result, err
}

func (c *BillingClient) startBillingInner(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	resp, err := c.client.StartBilling(ctx, &billingv1.StartBillingRequest{
		ReservationId:  reservationID,
		BookingFee:     bookingFee,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return protoToBillingRecord(resp), nil
}

func (c *BillingClient) CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error) {
	var result *billingmodel.BillingRecord
	err := c.cb.Execute(func() error {
		var err error
		result, err = c.calculateFeeInner(ctx, reservationID, checkInAt, checkOutAt)
		return err
	})
	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		return nil, apperror.New("SERVICE_UNAVAILABLE", "billing service temporarily unavailable", 503)
	}
	return result, err
}

func (c *BillingClient) calculateFeeInner(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error) {
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

func (c *BillingClient) GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	var result *billingmodel.BillingRecord
	err := c.cb.Execute(func() error {
		var err error
		result, err = c.generateInvoiceInner(ctx, reservationID, idempotencyKey)
		return err
	})
	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		return nil, apperror.New("SERVICE_UNAVAILABLE", "billing service temporarily unavailable", 503)
	}
	return result, err
}

func (c *BillingClient) generateInvoiceInner(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	resp, err := c.client.GenerateInvoice(ctx, &billingv1.GenerateInvoiceRequest{
		ReservationId:  reservationID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return protoToBillingRecord(resp), nil
}

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
