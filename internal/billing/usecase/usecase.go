package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	billingerrors "parkir-pintar/internal/billing/errors"
	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/repository"
	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/pkg/idempotency"
	"parkir-pintar/pkg/pricing"
)

//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/billing/usecase Usecase
type Usecase interface {
	StartBilling(ctx context.Context, req *model.StartBillingRequest) (*model.BillingRecord, error)
	CalculateFee(ctx context.Context, req *model.CalculateFeeRequest) (*model.BillingRecord, error)
	GenerateInvoice(ctx context.Context, req *model.GenerateInvoiceRequest) (*model.BillingRecord, error)
	ApplyOvernightFee(ctx context.Context, req *model.ApplyOvernightFeeRequest) (*model.BillingRecord, error)
}

func (uc *billingUsecase) StartBilling(ctx context.Context, req *model.StartBillingRequest) (*model.BillingRecord, error) {
	res, err := idempotency.Check(ctx, req.IdempotencyKey, uc.repo.GetByIdempotencyKey, repository.ErrNotFound, "start billing")
	if err != nil {
		return nil, err
	}
	if res.Found {
		return res.Record, nil
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

func (uc *billingUsecase) CalculateFee(ctx context.Context, req *model.CalculateFeeRequest) (*model.BillingRecord, error) {
	const maxRetries = 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
		if err != nil {
			return nil, fmt.Errorf("calculate fee get record: %w", err)
		}

		if record.Status != model.BillingStatusPending {
			return nil, fmt.Errorf("%w: current status %q, expected %q", billingerrors.ErrCannotCalculate, record.Status, model.BillingStatusPending)
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
			if errors.Is(err, repository.ErrConcurrentModification) && attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("calculate fee update: %w", err)
		}

		return record, nil
	}
	return nil, fmt.Errorf("calculate fee: %w", repository.ErrConcurrentModification)
}

func (uc *billingUsecase) GenerateInvoice(ctx context.Context, req *model.GenerateInvoiceRequest) (*model.BillingRecord, error) {
	record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("generate invoice get record: %w", err)
	}

	if record.Status == model.BillingStatusInvoiced {
		return record, nil
	}

	if record.Status != model.BillingStatusCalculated {
		return nil, fmt.Errorf("%w: current status %q, expected %q or %q", billingerrors.ErrCannotInvoice, record.Status, model.BillingStatusCalculated, model.BillingStatusInvoiced)
	}

	record.Status = model.BillingStatusInvoiced
	record.UpdatedAt = time.Now()

	if err := uc.repo.UpdateBillingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("generate invoice update: %w", err)
	}

	return record, nil
}

func (uc *billingUsecase) ApplyOvernightFee(ctx context.Context, req *model.ApplyOvernightFeeRequest) (*model.BillingRecord, error) {
	const maxRetries = 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
		if err != nil {
			return nil, fmt.Errorf("apply overnight fee get record: %w", err)
		}

		if record.IsOvernight {
			return record, nil
		}

		overnightFee := req.Amount
		if overnightFee <= 0 {
			overnightFee = constants.OvernightPerNight
		}

		record.OvernightFee = overnightFee
		record.IsOvernight = true
		record.TotalAmount = pricing.CalculateTotal(record.BookingFee, record.ParkingFee, record.OvernightFee)
		record.UpdatedAt = time.Now()

		if err := uc.repo.UpdateBillingRecord(ctx, record); err != nil {
			if errors.Is(err, repository.ErrConcurrentModification) && attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("apply overnight fee update: %w", err)
		}

		return record, nil
	}
	return nil, fmt.Errorf("apply overnight fee: %w", repository.ErrConcurrentModification)
}
