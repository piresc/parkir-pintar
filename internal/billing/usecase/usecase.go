// Package usecase implements the business logic layer for the billing domain
// module. It orchestrates billing record creation, fee calculation, invoice
// generation, penalty application, and overnight fee handling, coordinating
// with the repository and NATS (event publishing).
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/repository"
)

// NATSClient defines the interface for NATS JetStream event publishing.
//
//go:generate mockgen -destination=../mocks/mock_nats_client.go -package=mocks parkir-pintar/internal/billing/usecase NATSClient
type NATSClient interface {
	Publish(subject string, data []byte) error
}

// Usecase defines the business logic interface for billing operations.
//
//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/billing/usecase Usecase
type Usecase interface {
	StartBilling(ctx context.Context, req *model.StartBillingRequest) (*model.BillingRecord, error)
	CalculateFee(ctx context.Context, req *model.CalculateFeeRequest) (*model.BillingRecord, error)
	GenerateInvoice(ctx context.Context, req *model.GenerateInvoiceRequest) (*model.BillingRecord, error)
	ApplyPenalty(ctx context.Context, req *model.ApplyPenaltyRequest) (*model.BillingRecord, error)
	ApplyOvernightFee(ctx context.Context, req *model.ApplyOvernightFeeRequest) (*model.BillingRecord, error)
}

// billingUsecase is the concrete implementation of Usecase.
type billingUsecase struct {
	repo repository.Repository
	nats NATSClient
}

// NewUsecase creates a new billing Usecase with all required dependencies.
func NewUsecase(repo repository.Repository, nats NATSClient) Usecase {
	return &billingUsecase{
		repo: repo,
		nats: nats,
	}
}

// StartBilling creates a billing record with the booking fee when a reservation
// is confirmed. It performs an idempotency check via the idempotency key.
func (uc *billingUsecase) StartBilling(ctx context.Context, req *model.StartBillingRequest) (*model.BillingRecord, error) {
	existing, err := uc.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	existingByReservation, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err == nil && existingByReservation != nil {
		return existingByReservation, nil
	}

	now := time.Now()
	record := &model.BillingRecord{
		ID:             uuid.New().String(),
		ReservationID:  req.ReservationID,
		BookingFee:     req.BookingFee,
		IdempotencyKey: req.IdempotencyKey,
		Status:         model.BillingStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	record.TotalAmount = model.CalculateBillingTotal(record)

	if err := uc.repo.CreateBillingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("start billing: %w", err)
	}

	return record, nil
}

// CalculateFee computes the parking fee based on actual session duration using
// the pricing engine, updates the billing record, and publishes a billing.calculated event.
func (uc *billingUsecase) CalculateFee(ctx context.Context, req *model.CalculateFeeRequest) (*model.BillingRecord, error) {
	record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("calculate fee get record: %w", err)
	}

	feeResult := model.CalculateParkingFee(req.CheckInAt, req.CheckOutAt)

	record.ParkingFee = feeResult.ParkingFee
	record.OvernightFee = feeResult.OvernightFee
	record.DurationMinutes = feeResult.DurationMinutes
	record.BilledHours = feeResult.BilledHours
	record.IsOvernight = feeResult.IsOvernight
	record.TotalAmount = model.CalculateBillingTotal(record)
	record.Status = model.BillingStatusCalculated
	record.UpdatedAt = time.Now()

	if err := uc.repo.UpdateBillingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("calculate fee update: %w", err)
	}

	// Publish billing.calculated event (non-critical)
	uc.publishEvent("billing.calculated", record)

	return record, nil
}

// GenerateInvoice generates an idempotent invoice for a reservation. If the
// idempotency key already exists, the existing record is returned.
func (uc *billingUsecase) GenerateInvoice(ctx context.Context, req *model.GenerateInvoiceRequest) (*model.BillingRecord, error) {
	// Idempotency check via idempotency_key
	existing, err := uc.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("generate invoice get record: %w", err)
	}

	record.Status = model.BillingStatusInvoiced
	record.UpdatedAt = time.Now()

	if err := uc.repo.UpdateBillingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("generate invoice update: %w", err)
	}

	// Publish billing.invoiced event (non-critical)
	uc.publishEvent("billing.invoiced", record)

	return record, nil
}

// ApplyPenalty inserts a penalty record and atomically updates the billing
// record's penalty_amount and total_amount using a single SQL UPDATE to prevent
// lost updates under concurrent requests.
func (uc *billingUsecase) ApplyPenalty(ctx context.Context, req *model.ApplyPenaltyRequest) (*model.BillingRecord, error) {
	now := time.Now()
	penalty := &model.Penalty{
		ID:            uuid.New().String(),
		ReservationID: req.ReservationID,
		PenaltyType:   req.PenaltyType,
		Amount:        req.Amount,
		Description:   req.Description,
		CreatedAt:     now,
	}

	if err := uc.repo.CreatePenalty(ctx, penalty); err != nil {
		return nil, fmt.Errorf("apply penalty create: %w", err)
	}

	record, err := uc.repo.AddPenaltyAmount(ctx, req.ReservationID, req.Amount)
	if err != nil {
		return nil, fmt.Errorf("apply penalty update: %w", err)
	}

	// Publish billing event (non-critical)
	uc.publishEvent("billing.calculated", record)

	return record, nil
}

// ApplyOvernightFee updates the overnight fee and total on the billing record.
func (uc *billingUsecase) ApplyOvernightFee(ctx context.Context, req *model.ApplyOvernightFeeRequest) (*model.BillingRecord, error) {
	record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("apply overnight fee get record: %w", err)
	}

	record.OvernightFee = model.OvernightFlatFee
	record.IsOvernight = true
	record.TotalAmount = model.CalculateBillingTotal(record)
	record.UpdatedAt = time.Now()

	if err := uc.repo.UpdateBillingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("apply overnight fee update: %w", err)
	}

	return record, nil
}

// publishEvent serializes the billing record and publishes it to NATS.
// Errors are logged but do not fail the operation.
func (uc *billingUsecase) publishEvent(subject string, record *model.BillingRecord) {
	data, err := json.Marshal(record)
	if err != nil {
		slog.Error("failed to marshal billing event", slog.String("subject", subject), slog.Any("error", err))
		return
	}
	if err := uc.nats.Publish(subject, data); err != nil {
		slog.Error("failed to publish billing event", slog.String("subject", subject), slog.Any("error", err))
	}
}
