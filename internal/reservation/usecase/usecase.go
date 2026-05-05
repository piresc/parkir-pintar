// Package usecase implements the business logic layer for the reservation domain
// module. It orchestrates the reservation lifecycle: create, cancel, check-in,
// check-out, and expiry, coordinating with the repository, Redis (distributed
// locks), NATS (event publishing), and external billing/payment clients.
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/internal/reservation/repository"
	"parkir-pintar/pkg/apperror"
)

// BillingClient defines the interface for billing service operations.
//
//go:generate mockgen -destination=../mocks/mock_billing_client.go -package=mocks parkir-pintar/internal/reservation/usecase BillingClient
type BillingClient interface {
	StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) error
	CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error)
	GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error)
	ApplyPenalty(ctx context.Context, reservationID string, penaltyType string, amount int64, description string) error
}

// PaymentClient defines the interface for payment service operations.
//
//go:generate mockgen -destination=../mocks/mock_payment_client.go -package=mocks parkir-pintar/internal/reservation/usecase PaymentClient
type PaymentClient interface {
	ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) error
}

// RedisClient defines the interface for Redis operations used by the reservation usecase.
//
//go:generate mockgen -destination=../mocks/mock_redis_client.go -package=mocks parkir-pintar/internal/reservation/usecase RedisClient
type RedisClient interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
}

// NATSClient defines the interface for NATS JetStream event publishing.
//
//go:generate mockgen -destination=../mocks/mock_nats_client.go -package=mocks parkir-pintar/internal/reservation/usecase NATSClient
type NATSClient interface {
	Publish(subject string, data []byte) error
}

// Usecase defines the business logic interface for the reservation lifecycle.
//
//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/reservation/usecase Usecase
type Usecase interface {
	CreateReservation(ctx context.Context, req *model.CreateReservationRequest) (*model.Reservation, error)
	CancelReservation(ctx context.Context, req *model.CancelReservationRequest) (*model.Reservation, error)
	CheckIn(ctx context.Context, req *model.CheckInRequest) (*model.Reservation, error)
	CheckOut(ctx context.Context, req *model.CheckOutRequest) (*model.CheckOutResponse, error)
	ExpireReservation(ctx context.Context, req *model.ExpireReservationRequest) error
}

// reservationUsecase is the concrete implementation of Usecase.
type reservationUsecase struct {
	repo          repository.Repository
	redis         RedisClient
	nats          NATSClient
	billingClient BillingClient
	paymentClient PaymentClient
}

// NewUsecase creates a new reservation Usecase with all required dependencies.
func NewUsecase(
	repo repository.Repository,
	redis RedisClient,
	nats NATSClient,
	billingClient BillingClient,
	paymentClient PaymentClient,
) Usecase {
	return &reservationUsecase{
		repo:          repo,
		redis:         redis,
		nats:          nats,
		billingClient: billingClient,
		paymentClient: paymentClient,
	}
}

// CreateReservation handles idempotent spot reservation with distributed locking.
// It follows Algorithm 1 from the design document:
//  1. Idempotency check via FindByIdempotencyKey
//  2. Spot assignment (system_assigned or user_selected)
//  3. Redis distributed lock (SETNX with 30s TTL)
//  4. Double-check spot availability under lock
//  5. Create reservation + update spot in a DB transaction
//  6. Start billing (booking fee 5,000 IDR)
//  7. Publish "reservation.confirmed" event
func (uc *reservationUsecase) CreateReservation(ctx context.Context, req *model.CreateReservationRequest) (*model.Reservation, error) {
	// Step 1: Idempotency check
	existing, err := uc.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Step 2: Find available spot
	var spotID string
	switch req.AssignmentMode {
	case model.AssignmentSystemAssigned:
		spot, err := uc.repo.FindAvailableSpot(ctx, req.VehicleType)
		if err != nil {
			return nil, apperror.New("CONFLICT", "no available spots for vehicle type", 409)
		}
		spotID = spot.ID
	case model.AssignmentUserSelected:
		spotID = req.SpotID
	default:
		return nil, apperror.BadRequest("invalid assignment mode")
	}

	// Step 3: Acquire distributed lock
	lockKey := fmt.Sprintf("lock:spot:%s", spotID)
	acquired, err := uc.redis.SetNX(ctx, lockKey, "locked", 30*time.Second)
	if err != nil || !acquired {
		return nil, apperror.New("CONFLICT", "spot is being reserved by another driver", 409)
	}
	defer func() {
		if delErr := uc.redis.Delete(ctx, lockKey); delErr != nil {
			slog.Error("failed to release spot lock", slog.String("lock_key", lockKey), slog.Any("error", delErr))
		}
	}()

	// Step 4: Double-check spot availability and vehicle-type compatibility under lock
	spot, err := uc.repo.GetSpotForUpdate(ctx, spotID)
	if err != nil || spot.Status != "available" {
		return nil, apperror.New("CONFLICT", "spot no longer available", 409)
	}
	if spot.VehicleType != req.VehicleType {
		return nil, apperror.BadRequest("spot vehicle type does not match requested vehicle type")
	}

	// Step 5: Create reservation in transaction
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)
	reservation := &model.Reservation{
		ID:             uuid.New().String(),
		DriverID:       req.DriverID,
		SpotID:         spotID,
		VehicleType:    req.VehicleType,
		AssignmentMode: req.AssignmentMode,
		Status:         model.StatusConfirmed,
		IdempotencyKey: req.IdempotencyKey,
		ConfirmedAt:    &now,
		ExpiresAt:      &expiresAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		if err := uc.repo.CreateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, spotID, "reserved")
	}); err != nil {
		// Handle concurrent duplicate idempotency key: if the DB rejects with a
		// unique constraint violation, another request already inserted the row.
		// Retry the idempotency lookup to return the existing reservation.
		errMsg := err.Error()
		if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint") {
			existing, findErr := uc.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
			if findErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("create reservation: %w", err)
	}

	// Step 6: Start billing (non-critical — log errors but don't fail)
	billingIdempotencyKey := fmt.Sprintf("billing-%s", reservation.ID)
	if err := uc.billingClient.StartBilling(ctx, reservation.ID, billingmodel.BookingFee, billingIdempotencyKey); err != nil {
		slog.Error("failed to start billing", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
	}

	// Step 7: Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.confirmed", data); err != nil {
			slog.Error("failed to publish reservation.confirmed", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	return reservation, nil
}

// CancelReservation cancels a confirmed reservation, calculates the cancellation
// fee based on time elapsed since confirmation, releases the spot, and publishes
// a cancellation event. Uses SELECT FOR UPDATE to prevent TOCTOU races.
func (uc *reservationUsecase) CancelReservation(ctx context.Context, req *model.CancelReservationRequest) (*model.Reservation, error) {
	var reservation *model.Reservation

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		var err error
		reservation, err = uc.repo.GetByIDForUpdate(ctx, tx, req.ReservationID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return apperror.NotFound("reservation not found")
			}
			return fmt.Errorf("cancel reservation get: %w", err)
		}

		if err := model.ValidateTransition(reservation.Status, model.StatusCancelled); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = model.StatusCancelled
		reservation.CancelledAt = &now
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, "available")
	}); err != nil {
		return nil, err
	}

	// Apply cancellation fee if applicable (non-critical, outside transaction)
	cancellationFee := billingmodel.CalculateCancellationFee(*reservation.ConfirmedAt, *reservation.CancelledAt)
	if cancellationFee > 0 {
		if err := uc.billingClient.ApplyPenalty(ctx, reservation.ID, "cancellation", cancellationFee, "cancellation fee"); err != nil {
			slog.Error("failed to apply cancellation fee", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	// Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.cancelled", data); err != nil {
			slog.Error("failed to publish reservation.cancelled", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	return reservation, nil
}

// CheckIn transitions a confirmed reservation to checked_in, updates the spot
// to occupied, notifies billing, and publishes a check-in event.
// Uses SELECT FOR UPDATE to prevent TOCTOU races.
func (uc *reservationUsecase) CheckIn(ctx context.Context, req *model.CheckInRequest) (*model.Reservation, error) {
	var reservation *model.Reservation

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		var err error
		reservation, err = uc.repo.GetByIDForUpdate(ctx, tx, req.ReservationID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return apperror.NotFound("reservation not found")
			}
			return fmt.Errorf("check-in get: %w", err)
		}

		if err := model.ValidateTransition(reservation.Status, model.StatusCheckedIn); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = model.StatusCheckedIn
		reservation.CheckedInAt = &now
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, "occupied")
	}); err != nil {
		return nil, err
	}

	// Notify billing to activate (non-critical, outside transaction)
	billingIdempotencyKey := fmt.Sprintf("checkin-billing-%s", reservation.ID)
	if err := uc.billingClient.StartBilling(ctx, reservation.ID, 0, billingIdempotencyKey); err != nil {
		slog.Error("failed to activate billing on check-in", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
	}

	// Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.checked_in", data); err != nil {
			slog.Error("failed to publish reservation.checked_in", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	return reservation, nil
}

// CheckOut transitions a checked_in reservation to checked_out. It calculates
// the fee via billing, generates an invoice, processes payment, releases the
// spot, and publishes a check-out event.
// Uses SELECT FOR UPDATE to prevent TOCTOU races on concurrent checkouts.
func (uc *reservationUsecase) CheckOut(ctx context.Context, req *model.CheckOutRequest) (*model.CheckOutResponse, error) {
	var reservation *model.Reservation

	// Phase 1: Lock the reservation row, validate state, and mark as checked_out atomically
	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		var err error
		reservation, err = uc.repo.GetByIDForUpdate(ctx, tx, req.ReservationID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return apperror.NotFound("reservation not found")
			}
			return fmt.Errorf("check-out get: %w", err)
		}

		if err := model.ValidateTransition(reservation.Status, model.StatusCheckedOut); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = model.StatusCheckedOut
		reservation.CheckedOutAt = &now
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, "available")
	}); err != nil {
		return nil, err
	}

	// Phase 2: Calculate fee, generate invoice, process payment (outside the row lock)
	billingRecord, err := uc.billingClient.CalculateFee(ctx, reservation.ID, *reservation.CheckedInAt, *reservation.CheckedOutAt)
	if err != nil {
		return nil, fmt.Errorf("check-out calculate fee: %w", err)
	}

	invoiceIdempotencyKey := fmt.Sprintf("invoice-%s", reservation.ID)
	billingRecord, err = uc.billingClient.GenerateInvoice(ctx, reservation.ID, invoiceIdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("check-out generate invoice: %w", err)
	}

	paymentIdempotencyKey := fmt.Sprintf("payment-%s", reservation.ID)
	if err := uc.paymentClient.ProcessPayment(ctx, billingRecord.ID, billingRecord.TotalAmount, "qris", paymentIdempotencyKey); err != nil {
		return nil, fmt.Errorf("check-out process payment: %w", err)
	}

	// Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.checked_out", data); err != nil {
			slog.Error("failed to publish reservation.checked_out", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	return &model.CheckOutResponse{
		Reservation: reservation,
		TotalAmount: billingRecord.TotalAmount,
		BillingID:   billingRecord.ID,
	}, nil
}

// ExpireReservation transitions a confirmed reservation to expired, releases
// the spot, applies a no-show penalty (10,000 IDR), and publishes an expiry event.
// Uses SELECT FOR UPDATE to prevent TOCTOU races.
func (uc *reservationUsecase) ExpireReservation(ctx context.Context, req *model.ExpireReservationRequest) error {
	var reservation *model.Reservation

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		var err error
		reservation, err = uc.repo.GetByIDForUpdate(ctx, tx, req.ReservationID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return apperror.NotFound("reservation not found")
			}
			return fmt.Errorf("expire reservation get: %w", err)
		}

		if err := model.ValidateTransition(reservation.Status, model.StatusExpired); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = model.StatusExpired
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, "available")
	}); err != nil {
		return err
	}

	// No additional no-show penalty is applied. Per PRD, the booking fee
	// (5,000 IDR, already charged at confirmation) is the only cost the
	// driver forfeits when a reservation expires without check-in.

	// Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.expired", data); err != nil {
			slog.Error("failed to publish reservation.expired", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	return nil
}
