// Package usecase implements the business logic layer for the billing domain
// module. It orchestrates billing record creation, fee calculation, invoice
// generation, penalty application, and overnight fee handling, coordinating
// with the repository.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/repository"
	"parkir-pintar/pkg/pricing"
)

// Usecase defines the business logic interface for billing operations.
//
//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/billing/usecase Usecase
type Usecase interface {
	StartBilling(ctx context.Context, req *model.StartBillingRequest) (*model.BillingRecord, error)
	CalculateFee(ctx context.Context, req *model.CalculateFeeRequest) (*model.BillingRecord, error)
	GenerateInvoice(ctx context.Context, req *model.GenerateInvoiceRequest) (*model.BillingRecord, error)
	ApplyOvernightFee(ctx context.Context, req *model.ApplyOvernightFeeRequest) (*model.BillingRecord, error)
}

// billingUsecase is the concrete implementation of Usecase.
type billingUsecase struct {
	repo repository.Repository
}

// NewUsecase creates a new billing Usecase with all required dependencies.
func NewUsecase(repo repository.Repository) Usecase {
	return &billingUsecase{
		repo: repo,
	}
}

// StartBilling creates a billing record with the booking fee when a reservation
// is confirmed. It performs an idempotency check via the idempotency key.
func (uc *billingUsecase) StartBilling(ctx context.Context, req *model.StartBillingRequest) (*model.BillingRecord, error) {
	existing, err := uc.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("start billing idempotency check: %w", err)
	}

	existingByReservation, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err == nil && existingByReservation != nil {
		return existingByReservation, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("start billing reservation check: %w", err)
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
	record.TotalAmount = pricing.CalculateTotal(record.BookingFee, record.ParkingFee, record.OvernightFee)

	if err := uc.repo.CreateBillingRecord(ctx, record); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			existing, fetchErr := uc.repo.GetByReservationID(ctx, req.ReservationID)
			if fetchErr != nil {
				return nil, fmt.Errorf("start billing fetch after conflict: %w", fetchErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("start billing: %w", err)
	}

	return record, nil
}

// CalculateFee computes the parking fee based on actual session duration using
// the pricing engine and updates the billing record.
func (uc *billingUsecase) CalculateFee(ctx context.Context, req *model.CalculateFeeRequest) (*model.BillingRecord, error) {
	record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("calculate fee get record: %w", err)
	}

	if record.Status != model.BillingStatusPending {
		return nil, fmt.Errorf("cannot calculate fee for billing record in status %q: expected %q", record.Status, model.BillingStatusPending)
	}

	feeResult := pricing.CalculateSessionFee(req.CheckInAt, req.CheckOutAt)

	record.ParkingFee = feeResult.ParkingFee
	record.OvernightFee = feeResult.OvernightFee
	record.DurationMinutes = feeResult.DurationMinutes
	record.BilledHours = feeResult.BilledHours
	record.IsOvernight = feeResult.IsOvernight
	record.TotalAmount = pricing.CalculateTotal(record.BookingFee, record.ParkingFee, record.OvernightFee)
	record.Status = model.BillingStatusCalculated
	record.UpdatedAt = time.Now()

	if err := uc.repo.UpdateBillingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("calculate fee update: %w", err)
	}

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
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("generate invoice idempotency check: %w", err)
	}

	record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("generate invoice get record: %w", err)
	}

	if record.Status != model.BillingStatusCalculated {
		return nil, fmt.Errorf("cannot invoice billing record in status %q: expected %q", record.Status, model.BillingStatusCalculated)
	}

	record.Status = model.BillingStatusInvoiced
	record.UpdatedAt = time.Now()

	if err := uc.repo.UpdateBillingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("generate invoice update: %w", err)
	}

	return record, nil
}

// ApplyOvernightFee updates the overnight fee and total on the billing record.
func (uc *billingUsecase) ApplyOvernightFee(ctx context.Context, req *model.ApplyOvernightFeeRequest) (*model.BillingRecord, error) {
	record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("apply overnight fee get record: %w", err)
	}

	// Idempotent: if already marked overnight, return as-is
	if record.IsOvernight {
		return record, nil
	}

	overnightFee := req.Amount
	if overnightFee <= 0 {
		overnightFee = pricing.OvernightPerNight
	}

	record.OvernightFee = overnightFee
	record.IsOvernight = true
	record.TotalAmount = pricing.CalculateTotal(record.BookingFee, record.ParkingFee, record.OvernightFee)
	record.UpdatedAt = time.Now()

	if err := uc.repo.UpdateBillingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("apply overnight fee update: %w", err)
	}

	return record, nil
}
