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
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/internal/reservation/repository"
	"parkir-pintar/pkg/apperror"
	"parkir-pintar/pkg/redislock"
)

// BillingClient defines the interface for billing service operations.
//
//go:generate mockgen -destination=../mocks/mock_billing_client.go -package=mocks parkir-pintar/internal/reservation/usecase BillingClient
type BillingClient interface {
	StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*billingmodel.BillingRecord, error)
	CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error)
	GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error)
	ApplyPenalty(ctx context.Context, reservationID string, penaltyType string, amount int64, description string) error
}

// PaymentClient defines the interface for payment service operations.
//
//go:generate mockgen -destination=../mocks/mock_payment_client.go -package=mocks parkir-pintar/internal/reservation/usecase PaymentClient
type PaymentClient interface {
	ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error)
}

// Lock represents an acquired distributed lock.
type Lock interface {
	Release(ctx context.Context) error
}

// Locker manages distributed lock acquisition.
type Locker interface {
	Acquire(ctx context.Context, key string) (Lock, error)
}

type lockerAdapter struct {
	inner *redislock.Locker
}

func (a *lockerAdapter) Acquire(ctx context.Context, key string) (Lock, error) {
	l, err := a.inner.Acquire(ctx, key)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// NewLockerAdapter wraps a *redislock.Locker to satisfy the Locker interface.
func NewLockerAdapter(l *redislock.Locker) Locker {
	return &lockerAdapter{inner: l}
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
	GetReservation(ctx context.Context, id string) (*model.Reservation, error)
	CancelReservation(ctx context.Context, req *model.CancelReservationRequest) (*model.Reservation, error)
	CheckIn(ctx context.Context, req *model.CheckInRequest) (*model.Reservation, error)
	CheckOut(ctx context.Context, req *model.CheckOutRequest) (*model.CheckOutResponse, error)
	ConfirmReservation(ctx context.Context, req *model.ConfirmReservationRequest) (*model.Reservation, error)
	CompleteCheckout(ctx context.Context, req *model.CompleteCheckoutRequest) (*model.CheckOutResponse, error)
	ExpireReservation(ctx context.Context, req *model.ExpireReservationRequest) error
	FailReservation(ctx context.Context, req *model.FailReservationRequest) error
	ListByDriver(ctx context.Context, driverID string, status string) ([]*model.Reservation, error)
}

// reservationUsecase is the concrete implementation of Usecase.
type reservationUsecase struct {
	repo          repository.Repository
	locker        Locker
	nats          NATSClient
	billingClient BillingClient
	paymentClient PaymentClient
}

// NewUsecase creates a new reservation Usecase with all required dependencies.
func NewUsecase(
	repo repository.Repository,
	locker Locker,
	nats NATSClient,
	billingClient BillingClient,
	paymentClient PaymentClient,
) Usecase {
	return &reservationUsecase{
		repo:          repo,
		locker:        locker,
		nats:          nats,
		billingClient: billingClient,
		paymentClient: paymentClient,
	}
}

// GetReservation retrieves a reservation by ID.
func (uc *reservationUsecase) GetReservation(ctx context.Context, id string) (*model.Reservation, error) {
	return uc.repo.GetByID(ctx, id)
}

// CreateReservation handles idempotent spot reservation with distributed locking.
//
// Flow:
//  1. Idempotency check via FindByIdempotencyKey
//  2. Spot assignment (system_assigned or user_selected)
//  3. Redis distributed lock (SETNX with configurable TTL)
//  4. Double-check spot availability under lock
//  5. Create reservation + update spot in DB transaction (status=waiting_payment)
//  6. Create billing record (booking fee 5,000 IDR)
//  7. Return reservation in waiting_payment state
func (uc *reservationUsecase) CreateReservation(ctx context.Context, req *model.CreateReservationRequest) (*model.Reservation, error) {
	// Step 1: Idempotency check
	existing, err := uc.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Check if driver already has an active reservation
	active, _ := uc.repo.ListByDriverID(ctx, req.DriverID, "")
	for _, r := range active {
		if r.Status == model.StatusWaitingPayment || r.Status == model.StatusConfirmed || r.Status == model.StatusCheckedIn {
			return nil, apperror.New("CONFLICT", "driver already has an active reservation", 409)
		}
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

	// Step 3: Acquire distributed lock (TTL configured in locker)
	lockKey := fmt.Sprintf("spot:%s", spotID)
	lock, err := uc.locker.Acquire(ctx, lockKey)
	if err != nil {
		if errors.Is(err, redislock.ErrLockUnavailable) {
			return nil, apperror.New("CONFLICT", "spot is being reserved by another driver", 409)
		}
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	defer func() {
		if unlockErr := lock.Release(ctx); unlockErr != nil {
			slog.Error("failed to release spot lock", slog.String("lock_key", lockKey), slog.Any("error", unlockErr))
		}
	}()

	// Step 4: Double-check spot availability and vehicle-type compatibility under lock
	spot, err := uc.repo.GetSpotForUpdate(ctx, spotID)
	if err != nil || spot.Status != "available" {
		return nil, apperror.New("CONFLICT", "spot no longer available", 409)
	}
	if req.AssignmentMode == model.AssignmentUserSelected && spot.VehicleType != req.VehicleType {
		return nil, apperror.BadRequest("spot vehicle type does not match requested vehicle type")
	}

	// Step 5: Create reservation in transaction with status=waiting_payment
	now := time.Now()
	reservation := &model.Reservation{
		ID:             uuid.New().String(),
		DriverID:       req.DriverID,
		SpotID:         spotID,
		VehicleType:    req.VehicleType,
		AssignmentMode: req.AssignmentMode,
		Status:         model.StatusWaitingPayment,
		IdempotencyKey: req.IdempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		if err := uc.repo.CreateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, spotID, "reserved")
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			existing, findErr := uc.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
			if findErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("create reservation: %w", err)
	}

	uc.publishSpotUpdated(ctx, spot)

	// Step 6: Create billing record with booking fee
	billingIdempotencyKey := fmt.Sprintf("billing-%s", reservation.ID)
	if _, err := uc.billingClient.StartBilling(ctx, reservation.ID, billingmodel.BookingFee, billingIdempotencyKey); err != nil {
		slog.Error("failed to start billing", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		uc.failReservationInternal(ctx, reservation)
		return nil, apperror.New("PAYMENT_FAILED", "unable to create billing record", 402)
	}

	return reservation, nil
}

// ConfirmReservation processes the booking fee payment for a waiting_payment
// reservation and transitions it to confirmed on success.
func (uc *reservationUsecase) ConfirmReservation(ctx context.Context, req *model.ConfirmReservationRequest) (*model.Reservation, error) {
	var reservation *model.Reservation

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		var err error
		reservation, err = uc.repo.GetByIDForUpdate(ctx, tx, req.ReservationID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return apperror.NotFound("reservation not found")
			}
			return fmt.Errorf("confirm reservation get: %w", err)
		}

		if reservation.Status != model.StatusWaitingPayment {
			return apperror.BadRequest("reservation is not pending payment")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// Re-call StartBilling (idempotent) to obtain billing record
	billingIdempotencyKey := fmt.Sprintf("billing-%s", reservation.ID)
	billingRecord, err := uc.billingClient.StartBilling(ctx, reservation.ID, billingmodel.BookingFee, billingIdempotencyKey)
	if err != nil {
		slog.Error("failed to get billing record", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		return nil, apperror.New("PAYMENT_FAILED", "unable to retrieve billing record", 402)
	}

	// Process payment for booking fee
	paymentIdempotencyKey := fmt.Sprintf("booking-payment-%s", reservation.ID)
	_, payErr := uc.paymentClient.ProcessPayment(ctx, billingRecord.ID, billingmodel.BookingFee, "qris", paymentIdempotencyKey)

	if payErr != nil {
		slog.Error("booking fee payment failed",
			slog.String("reservation_id", reservation.ID),
			slog.Any("error", payErr))
		uc.failReservationInternal(ctx, reservation)
		return nil, apperror.New("PAYMENT_FAILED", "booking fee payment failed", 402)
	}

	// Payment succeeded — confirm the reservation
	confirmedAt := time.Now()
	expiresAt := confirmedAt.Add(1 * time.Hour)
	reservation.Status = model.StatusConfirmed
	reservation.ConfirmedAt = &confirmedAt
	reservation.ExpiresAt = &expiresAt
	reservation.UpdatedAt = confirmedAt

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return uc.repo.UpdateReservationTx(ctx, tx, reservation)
	}); err != nil {
		slog.Error("failed to confirm reservation after payment",
			slog.String("reservation_id", reservation.ID),
			slog.Any("error", err))
		return nil, fmt.Errorf("confirm reservation: %w", err)
	}

	// Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.confirmed", data); err != nil {
			slog.Error("failed to publish reservation.confirmed", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	return reservation, nil
}

// failReservationInternal transitions a waiting_payment reservation to failed,
// releases the spot, and publishes a payment_failed event. Used when payment
// fails during ConfirmReservation or CreateReservation.
func (uc *reservationUsecase) failReservationInternal(ctx context.Context, reservation *model.Reservation) {
	now := time.Now()
	reservation.Status = model.StatusFailed
	reservation.UpdatedAt = now

	if txErr := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, "available")
	}); txErr != nil {
		slog.Error("failed to release spot on payment failure",
			slog.String("reservation_id", reservation.ID),
			slog.Any("error", txErr))
		return
	}

	uc.publishSpotStatusChange(ctx, reservation.SpotID, "available")

	// Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.payment_failed", data); err != nil {
			slog.Error("failed to publish reservation.payment_failed", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}
}

// CancelReservation cancels a confirmed or waiting_payment reservation, calculates
// the cancellation fee if applicable, releases the spot, and publishes a
// cancellation event. Uses SELECT FOR UPDATE to prevent TOCTOU races.
//
// When cancelled from waiting_payment state, no cancellation fee is charged
// (the booking fee was never successfully collected).
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

	uc.publishSpotStatusChange(ctx, reservation.SpotID, "available")

	// Apply cancellation fee only if reservation was confirmed (not waiting_payment)
	if reservation.ConfirmedAt != nil {
		cancellationFee := billingmodel.CalculateCancellationFee(*reservation.ConfirmedAt, *reservation.CancelledAt)
		if cancellationFee > 0 {
			if err := uc.billingClient.ApplyPenalty(ctx, reservation.ID, "cancellation", cancellationFee, "cancellation fee"); err != nil {
				slog.Error("failed to apply cancellation fee", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
			}
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

	uc.publishSpotStatusChange(ctx, reservation.SpotID, "occupied")

	// Notify billing to activate (non-critical, outside transaction)
	billingIdempotencyKey := fmt.Sprintf("checkin-billing-%s", reservation.ID)
	if _, err := uc.billingClient.StartBilling(ctx, reservation.ID, 0, billingIdempotencyKey); err != nil {
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

// CheckOut transitions a checked_in reservation to checked_out, calculates the
// fee via billing, generates an invoice, but does NOT process payment or release
// the spot. Payment and spot release happen in CompleteCheckout.
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

		return uc.repo.UpdateReservationTx(ctx, tx, reservation)
	}); err != nil {
		return nil, err
	}

	// Phase 2: Calculate fee and generate invoice (outside the row lock)
	_, err := uc.billingClient.CalculateFee(ctx, reservation.ID, *reservation.CheckedInAt, *reservation.CheckedOutAt)
	if err != nil {
		return nil, fmt.Errorf("check-out calculate fee: %w", err)
	}

	invoiceIdempotencyKey := fmt.Sprintf("invoice-%s", reservation.ID)
	billingRecord, err := uc.billingClient.GenerateInvoice(ctx, reservation.ID, invoiceIdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("check-out generate invoice: %w", err)
	}

	return &model.CheckOutResponse{
		Reservation:   reservation,
		TotalAmount:   billingRecord.TotalAmount,
		BillingID:     billingRecord.ID,
		PaymentID:     "", // payment not yet processed
		BookingFee:    billingRecord.BookingFee,
		ParkingFee:    billingRecord.ParkingFee,
		OvernightFee:  billingRecord.OvernightFee,
		PenaltyAmount: billingRecord.PenaltyAmount,
	}, nil
}

// CompleteCheckout processes the payment for a checked_out reservation and
// releases the spot back to inventory.
func (uc *reservationUsecase) CompleteCheckout(ctx context.Context, req *model.CompleteCheckoutRequest) (*model.CheckOutResponse, error) {
	var reservation *model.Reservation

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		var err error
		reservation, err = uc.repo.GetByIDForUpdate(ctx, tx, req.ReservationID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return apperror.NotFound("reservation not found")
			}
			return fmt.Errorf("complete checkout get: %w", err)
		}

		if reservation.Status != model.StatusCheckedOut {
			return apperror.BadRequest("reservation is not checked out")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// Re-generate invoice (idempotent) to obtain billing record
	invoiceIdempotencyKey := fmt.Sprintf("invoice-%s", reservation.ID)
	billingRecord, err := uc.billingClient.GenerateInvoice(ctx, reservation.ID, invoiceIdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("complete checkout get invoice: %w", err)
	}

	// Process payment for the total amount
	paymentIdempotencyKey := fmt.Sprintf("payment-%s", reservation.ID)
	paymentID, err := uc.paymentClient.ProcessPayment(ctx, billingRecord.ID, billingRecord.TotalAmount, "qris", paymentIdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("complete checkout process payment: %w", err)
	}

	// Release spot
	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, "available")
	}); err != nil {
		slog.Error("failed to release spot after checkout payment",
			slog.String("reservation_id", reservation.ID),
			slog.Any("error", err))
		return nil, fmt.Errorf("complete checkout release spot: %w", err)
	}

	uc.publishSpotStatusChange(ctx, reservation.SpotID, "available")

	// Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.checked_out", data); err != nil {
			slog.Error("failed to publish reservation.checked_out", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	return &model.CheckOutResponse{
		Reservation:   reservation,
		TotalAmount:   billingRecord.TotalAmount,
		BillingID:     billingRecord.ID,
		PaymentID:     paymentID,
		BookingFee:    billingRecord.BookingFee,
		ParkingFee:    billingRecord.ParkingFee,
		OvernightFee:  billingRecord.OvernightFee,
		PenaltyAmount: billingRecord.PenaltyAmount,
	}, nil
}

func (uc *reservationUsecase) publishSpotUpdated(ctx context.Context, spot *model.ParkingSpot) {
	data, err := json.Marshal(map[string]interface{}{
		"id":           spot.ID,
		"floor_number": spot.FloorNumber,
		"spot_number":  spot.SpotNumber,
		"vehicle_type": spot.VehicleType,
		"spot_code":    spot.SpotCode,
		"status":       spot.Status,
	})
	if err != nil {
		slog.Warn("reservation: failed to marshal spot.updated event", slog.Any("error", err))
		return
	}
	if err := uc.nats.Publish("spot.updated", data); err != nil {
		slog.Warn("reservation: failed to publish spot.updated event", slog.Any("error", err))
	}
}

func (uc *reservationUsecase) publishSpotStatusChange(ctx context.Context, spotID string, status string) {
	spot, err := uc.repo.GetSpotForUpdate(ctx, spotID)
	if err != nil {
		slog.Warn("reservation: failed to fetch spot for status change event",
			slog.String("spot_id", spotID), slog.Any("error", err))
		return
	}
	uc.publishSpotUpdated(ctx, spot)
}

// ExpireReservation transitions a confirmed reservation to expired, releases
// the spot, and publishes an expiry event.
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

	uc.publishSpotStatusChange(ctx, reservation.SpotID, "available")

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

// FailReservation transitions a waiting_payment reservation to failed, releases
// the spot, and publishes a payment_failed event. Called by the payment timeout
// worker for reservations that exceeded the payment window.
// Uses SELECT FOR UPDATE to prevent TOCTOU races.
func (uc *reservationUsecase) FailReservation(ctx context.Context, req *model.FailReservationRequest) error {
	var reservation *model.Reservation

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		var err error
		reservation, err = uc.repo.GetByIDForUpdate(ctx, tx, req.ReservationID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return apperror.NotFound("reservation not found")
			}
			return fmt.Errorf("fail reservation get: %w", err)
		}

		if err := model.ValidateTransition(reservation.Status, model.StatusFailed); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = model.StatusFailed
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, "available")
	}); err != nil {
		return err
	}

	uc.publishSpotStatusChange(ctx, reservation.SpotID, "available")

	// Publish event (non-critical)
	if data, err := json.Marshal(reservation); err == nil {
		if err := uc.nats.Publish("reservation.payment_failed", data); err != nil {
			slog.Error("failed to publish reservation.payment_failed", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		}
	}

	return nil
}

// ListByDriver retrieves reservations for a driver with optional status filter.
func (uc *reservationUsecase) ListByDriver(ctx context.Context, driverID string, status string) ([]*model.Reservation, error) {
	return uc.repo.ListByDriverID(ctx, driverID, status)
}
