// Package usecase implements the business logic layer for the reservation domain
// module. It orchestrates the reservation lifecycle: create, cancel, check-in,
// check-out, and expiry, coordinating with the repository, Redis (distributed
// locks), and external billing/payment clients.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/gateway"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/internal/reservation/repository"
	"parkir-pintar/pkg/apperror"
	"parkir-pintar/pkg/database"
	pkgnats "parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redislock"
)

// BillingClient defines the interface for billing service operations.
//
//go:generate mockgen -destination=../mocks/mock_billing_client.go -package=mocks parkir-pintar/internal/reservation/usecase BillingClient
type BillingClient interface {
	StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*billingmodel.BillingRecord, error)
	CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error)
	GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error)
}

// PaymentClient defines the interface for payment service operations.
//
//go:generate mockgen -destination=../mocks/mock_payment_client.go -package=mocks parkir-pintar/internal/reservation/usecase PaymentClient
type PaymentClient interface {
	ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error)
	RefundPayment(ctx context.Context, paymentID string) error
}

// PresenceClient defines the interface for presence verification operations.
// This is optional — if nil, presence verification is skipped (graceful degradation).
//
//go:generate mockgen -destination=../mocks/mock_presence_client.go -package=mocks parkir-pintar/internal/reservation/usecase PresenceClient
type PresenceClient interface {
	VerifyPresence(ctx context.Context, driverID string, reservationID string, floorNumber int, spotNumber int) (*PresenceResult, error)
}

// PresenceResult holds the result of a presence verification check.
type PresenceResult struct {
	Verified bool
	Message  string
}

// Lock represents an acquired distributed lock.
type Lock interface {
	Release(ctx context.Context) error
}

// Locker manages distributed lock acquisition.
type Locker interface {
	Acquire(ctx context.Context, key string) (Lock, error)
}

// TaskEnqueuer enqueues asynchronous tasks (e.g. reservation expiry).
// Implementations must be safe to call concurrently.
//
//go:generate mockgen -destination=../mocks/mock_task_enqueuer.go -package=mocks parkir-pintar/internal/reservation/usecase TaskEnqueuer
type TaskEnqueuer interface {
	EnqueueReservationExpiry(ctx context.Context, reservationID string, delay time.Duration) (string, error)
	EnqueuePaymentHoldTimeout(ctx context.Context, reservationID string, paymentID string, delay time.Duration) (string, error)
	CancelTask(ctx context.Context, taskID string) error
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

// Usecase defines the business logic interface for the reservation lifecycle.
//
//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/reservation/usecase Usecase
type Usecase interface {
	CreateReservation(ctx context.Context, req *model.CreateReservationRequest) (*model.Reservation, error)
	GetReservation(ctx context.Context, id string, callerID string) (*model.Reservation, error)
	CancelReservation(ctx context.Context, req *model.CancelReservationRequest) (*model.Reservation, error)
	CheckIn(ctx context.Context, req *model.CheckInRequest) (*model.CheckInResponse, error)
	CheckOut(ctx context.Context, req *model.CheckOutRequest) (*model.CheckOutResponse, error)
	ConfirmReservation(ctx context.Context, req *model.ConfirmReservationRequest) (*model.Reservation, error)
	CompleteCheckout(ctx context.Context, req *model.CompleteCheckoutRequest) (*model.CheckOutResponse, error)
	ExpireReservation(ctx context.Context, req *model.ExpireReservationRequest) error
	FailReservation(ctx context.Context, req *model.FailReservationRequest) error
	ListByDriver(ctx context.Context, driverID string, status string) ([]*model.Reservation, error)
}

// reservationUsecase is the concrete implementation of Usecase.
type reservationUsecase struct {
	repo           repository.Repository
	locker         Locker
	billingClient  BillingClient
	paymentClient  PaymentClient
	presenceClient PresenceClient
	taskEnqueuer   TaskEnqueuer
	eventPublisher gateway.EventPublisher
	expiryTimeout  time.Duration
	paymentTimeout time.Duration
}

// NewUsecase creates a new reservation Usecase with all required dependencies.
// taskEnqueuer, eventPublisher, and presenceClient are optional (nil-safe); when nil,
// no async tasks are enqueued, no events are published, and presence verification
// is skipped respectively.
func NewUsecase(
	repo repository.Repository,
	locker Locker,
	billingClient BillingClient,
	paymentClient PaymentClient,
	presenceClient PresenceClient,
	taskEnqueuer TaskEnqueuer,
	eventPublisher gateway.EventPublisher,
	expiryTimeoutMinutes int,
	paymentTimeoutMinutes int,
) Usecase {
	timeout := time.Duration(expiryTimeoutMinutes) * time.Minute
	if timeout <= 0 {
		timeout = constants.DefaultExpiryTimeout
	}
	paymentTimeout := time.Duration(paymentTimeoutMinutes) * time.Minute
	if paymentTimeout <= 0 {
		paymentTimeout = constants.DefaultPaymentTimeout
	}
	return &reservationUsecase{
		repo:           repo,
		locker:         locker,
		billingClient:  billingClient,
		paymentClient:  paymentClient,
		presenceClient: presenceClient,
		taskEnqueuer:   taskEnqueuer,
		eventPublisher: eventPublisher,
		expiryTimeout:  timeout,
		paymentTimeout: paymentTimeout,
	}
}

// GetReservation retrieves a reservation by ID and enforces ownership when callerID is provided.
func (uc *reservationUsecase) GetReservation(ctx context.Context, id string, callerID string) (*model.Reservation, error) {
	reservation, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, apperror.NotFound("reservation not found")
		}
		return nil, fmt.Errorf("get reservation: %w", err)
	}
	if callerID != "" && reservation.DriverID != callerID {
		return nil, apperror.New("FORBIDDEN", "reservation belongs to another driver", 403)
	}
	return reservation, nil
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

	// Step 2: Find available spot
	var spotID string
	switch req.AssignmentMode {
	case constants.AssignmentSystemAssigned:
		spot, err := uc.repo.FindAvailableSpot(ctx, req.VehicleType)
		if err != nil {
			return nil, apperror.New("CONFLICT", "no available spots for vehicle type", 409)
		}
		spotID = spot.ID
	case constants.AssignmentUserSelected:
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
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if unlockErr := lock.Release(releaseCtx); unlockErr != nil {
			slog.Error("failed to release spot lock", slog.String("lock_key", lockKey), slog.Any("error", unlockErr))
		}
	}()

	// Step 4+5: Verify spot AND create reservation in single transaction
	now := time.Now()
	reservation := &model.Reservation{
		ID:             uuid.New().String(),
		DriverID:       req.DriverID,
		SpotID:         spotID,
		VehicleType:    req.VehicleType,
		AssignmentMode: req.AssignmentMode,
		Status:         constants.StatusWaitingPayment,
		IdempotencyKey: req.IdempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// Double-check spot availability under DB row lock (within transaction)
		spot, err := uc.repo.GetSpotForUpdateTx(ctx, tx, spotID)
		if err != nil || spot.Status != constants.SpotStatusAvailable {
			return apperror.New("CONFLICT", "spot no longer available", 409)
		}
		if req.AssignmentMode == constants.AssignmentUserSelected && spot.VehicleType != req.VehicleType {
			return apperror.BadRequest("spot vehicle type does not match requested vehicle type")
		}

		if err := uc.repo.CreateReservationTx(ctx, tx, reservation); err != nil {
			if database.IsUniqueViolationOn(err, "idx_reservations_one_active_per_driver") {
				return apperror.New("CONFLICT", "driver already has an active reservation", 409)
			}
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, spotID, constants.SpotStatusReserved)
	}); err != nil {
		if database.IsUniqueViolation(err) {
			existing, findErr := uc.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
			if findErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("create reservation: %w", err)
	}

	// Step 6: Create billing record with booking fee
	billingIdempotencyKey := fmt.Sprintf("billing-%s", reservation.ID)
	if _, err := uc.billingClient.StartBilling(ctx, reservation.ID, constants.BookingFee, billingIdempotencyKey); err != nil {
		slog.Error("failed to start billing", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		uc.failReservationInternal(ctx, reservation)
		return nil, apperror.New("PAYMENT_FAILED", "unable to create billing record", 402)
	}

	// Step 7: Enqueue expiry task (non-critical, best-effort)
	if uc.taskEnqueuer != nil {
		if _, err := uc.taskEnqueuer.EnqueueReservationExpiry(ctx, reservation.ID, uc.expiryTimeout); err != nil {
			slog.Error("failed to enqueue reservation expiry task",
				slog.String("reservation_id", reservation.ID),
				slog.Any("error", err))
		}
	}

	// Step 7b: Enqueue payment hold timeout (fails reservation if payment not completed in time)
	if uc.taskEnqueuer != nil {
		if _, err := uc.taskEnqueuer.EnqueuePaymentHoldTimeout(ctx, reservation.ID, "", uc.paymentTimeout); err != nil {
			slog.Error("failed to enqueue payment hold timeout",
				slog.String("reservation_id", reservation.ID),
				slog.Any("error", err))
		}
	}

	// Step 8: Publish events (best-effort)
	uc.publishSpotUpdated(ctx, reservation.SpotID, constants.SpotStatusReserved)
	uc.publishAnalyticsEvent(ctx, pkgnats.SubjectReservationAnalyticsCreated, reservation, "created")

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

		if req.CallerID != "" && reservation.DriverID != req.CallerID {
			return apperror.New("FORBIDDEN", "reservation belongs to another driver", 403)
		}

		if reservation.Status != constants.StatusWaitingPayment {
			return apperror.BadRequest("reservation is not pending payment")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// Re-call StartBilling (idempotent) to obtain billing record
	billingIdempotencyKey := fmt.Sprintf("billing-%s", reservation.ID)
	billingRecord, err := uc.billingClient.StartBilling(ctx, reservation.ID, constants.BookingFee, billingIdempotencyKey)
	if err != nil {
		slog.Error("failed to get billing record", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
		return nil, apperror.New("PAYMENT_FAILED", "unable to retrieve billing record", 402)
	}

	// Process payment for booking fee
	paymentIdempotencyKey := fmt.Sprintf("booking-payment-%s", reservation.ID)
	paymentID, payErr := uc.paymentClient.ProcessPayment(ctx, billingRecord.ID, constants.BookingFee, constants.PaymentMethodQRIS, paymentIdempotencyKey)

	if payErr != nil {
		slog.Error("booking fee payment failed",
			slog.String("reservation_id", reservation.ID),
			slog.Any("error", payErr))
		uc.failReservationInternal(ctx, reservation)
		return nil, apperror.New("PAYMENT_FAILED", "booking fee payment failed", 402)
	}

	// Payment succeeded — confirm the reservation
	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// Re-fetch with lock to prevent TOCTOU race
		current, err := uc.repo.GetByIDForUpdate(ctx, tx, reservation.ID)
		if err != nil {
			return fmt.Errorf("confirm reservation re-fetch: %w", err)
		}
		// Verify status hasn't changed since our first check
		if current.Status != constants.StatusWaitingPayment {
			// Compensate: refund the payment we just processed
			// Best-effort refund — log error if it fails for manual intervention
			if refundErr := uc.paymentClient.RefundPayment(ctx, paymentID); refundErr != nil {
				slog.Error("failed to refund orphaned payment",
					slog.String("payment_id", paymentID),
					slog.String("reservation_id", reservation.ID),
					slog.Any("error", refundErr))
			}
			return apperror.New("CONFLICT", "reservation status changed concurrently", 409)
		}

		confirmedAt := time.Now()
		expiresAt := confirmedAt.Add(constants.DefaultConfirmationExpiry)
		reservation.Status = constants.StatusConfirmed
		reservation.ConfirmedAt = &confirmedAt
		reservation.ExpiresAt = &expiresAt
		reservation.UpdatedAt = confirmedAt

		return uc.repo.UpdateReservationTx(ctx, tx, reservation)
	}); err != nil {
		slog.Error("failed to confirm reservation after payment",
			slog.String("reservation_id", reservation.ID),
			slog.Any("error", err))
		return nil, fmt.Errorf("confirm reservation: %w", err)
	}

	// Publish analytics event (best-effort)
	uc.publishAnalyticsEvent(ctx, pkgnats.SubjectReservationAnalyticsConfirmed, reservation, "confirmed")

	// Cancel payment hold timeout — driver paid in time
	if uc.taskEnqueuer != nil {
		// Best-effort: if cancel fails, the handler will see status=confirmed and skip
		if err := uc.taskEnqueuer.CancelTask(ctx, fmt.Sprintf("payment-hold:%s", reservation.ID)); err != nil {
			slog.Warn("failed to cancel payment hold timeout (non-critical)",
				slog.String("reservation_id", reservation.ID),
				slog.Any("error", err))
		}
	}

	return reservation, nil
}

// failReservationInternal transitions a waiting_payment reservation to failed,
// releases the spot. Used when payment fails during ConfirmReservation or CreateReservation.
func (uc *reservationUsecase) failReservationInternal(ctx context.Context, reservation *model.Reservation) {
	if txErr := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// Re-fetch with lock to prevent TOCTOU
		locked, err := uc.repo.GetByIDForUpdate(ctx, tx, reservation.ID)
		if err != nil {
			return fmt.Errorf("fail reservation lock: %w", err)
		}
		// Only transition from waiting_payment
		if locked.Status != constants.StatusWaitingPayment {
			return nil // Already transitioned by another path — no-op
		}

		now := time.Now()
		locked.Status = constants.StatusFailed
		locked.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, locked); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, locked.SpotID, constants.SpotStatusAvailable)
	}); txErr != nil {
		slog.Error("failed to release spot on payment failure",
			slog.String("reservation_id", reservation.ID),
			slog.Any("error", txErr))
	}
}

// CancelReservation cancels a confirmed or waiting_payment reservation, calculates
// the cancellation fee if applicable, releases the spot.
// Uses SELECT FOR UPDATE to prevent TOCTOU races.
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

		if req.CallerID != "" && reservation.DriverID != req.CallerID {
			return apperror.New("FORBIDDEN", "reservation belongs to another driver", 403)
		}

		if err := model.ValidateTransition(reservation.Status, constants.StatusCancelled); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = constants.StatusCancelled
		reservation.CancelledAt = &now
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, constants.SpotStatusAvailable)
	}); err != nil {
		return nil, err
	}

	// Publish events (best-effort)
	uc.publishSpotUpdated(ctx, reservation.SpotID, constants.SpotStatusAvailable)
	uc.publishAnalyticsEvent(ctx, pkgnats.SubjectReservationAnalyticsCancelled, reservation, "cancelled")

	return reservation, nil
}

// CheckIn transitions a confirmed reservation to checked_in, updates the spot
// to occupied, and notifies billing. Optionally verifies driver presence near
// the assigned spot (non-blocking: errors are logged but do not prevent check-in).
// Uses SELECT FOR UPDATE to prevent TOCTOU races.
func (uc *reservationUsecase) CheckIn(ctx context.Context, req *model.CheckInRequest) (*model.CheckInResponse, error) {
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

		if req.CallerID != "" && reservation.DriverID != req.CallerID {
			return apperror.New("FORBIDDEN", "reservation belongs to another driver", 403)
		}

		if err := model.ValidateTransition(reservation.Status, constants.StatusCheckedIn); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = constants.StatusCheckedIn
		reservation.CheckedInAt = &now
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, constants.SpotStatusOccupied)
	}); err != nil {
		return nil, err
	}

	// Notify billing to activate (non-critical, outside transaction)
	billingIdempotencyKey := fmt.Sprintf("checkin-billing-%s", reservation.ID)
	if _, err := uc.billingClient.StartBilling(ctx, reservation.ID, 0, billingIdempotencyKey); err != nil {
		slog.Error("failed to activate billing on check-in", slog.String("reservation_id", reservation.ID), slog.Any("error", err))
	}

	// Presence verification (non-blocking, graceful degradation)
	response := &model.CheckInResponse{Reservation: reservation}
	if uc.presenceClient != nil {
		// Look up the spot to get floor/spot numbers for sensor verification.
		spot, spotErr := uc.repo.GetSpotByID(ctx, reservation.SpotID)
		if spotErr != nil {
			slog.Warn("failed to look up spot for presence verification, skipping",
				slog.String("reservation_id", reservation.ID),
				slog.Any("error", spotErr))
		} else {
			presenceResult, err := uc.presenceClient.VerifyPresence(ctx, reservation.DriverID, reservation.ID, spot.FloorNumber, spot.SpotNumber)
			if err != nil {
				slog.Warn("presence verification failed, skipping",
					slog.String("reservation_id", reservation.ID),
					slog.Any("error", err))
			} else if !presenceResult.Verified {
				response.WrongSpotWarning = true
				slog.Warn("driver checked in at wrong spot",
					slog.String("reservation_id", reservation.ID),
					slog.String("driver_id", reservation.DriverID),
					slog.String("message", presenceResult.Message))
			}
		}
	}

	// Publish events (best-effort)
	uc.publishSpotUpdated(ctx, reservation.SpotID, constants.SpotStatusOccupied)
	uc.publishAnalyticsEvent(ctx, pkgnats.SubjectReservationAnalyticsCheckedIn, reservation, "checked-in")

	return response, nil
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

		if req.CallerID != "" && reservation.DriverID != req.CallerID {
			return apperror.New("FORBIDDEN", "reservation belongs to another driver", 403)
		}

		if err := model.ValidateTransition(reservation.Status, constants.StatusCheckedOut); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = constants.StatusCheckedOut
		reservation.CheckedOutAt = &now
		reservation.UpdatedAt = now

		return uc.repo.UpdateReservationTx(ctx, tx, reservation)
	}); err != nil {
		return nil, err
	}

	// Phase 2: Calculate fee and generate invoice (outside the row lock)
	if reservation.CheckedInAt == nil || reservation.CheckedOutAt == nil {
		return nil, apperror.New("INTERNAL", "reservation missing check-in/check-out timestamps", 500)
	}
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
		Reservation:  reservation,
		TotalAmount:  billingRecord.TotalAmount,
		BillingID:    billingRecord.ID,
		PaymentID:    "", // payment not yet processed
		BookingFee:   billingRecord.BookingFee,
		ParkingFee:   billingRecord.ParkingFee,
		OvernightFee: billingRecord.OvernightFee,
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

		if req.CallerID != "" && reservation.DriverID != req.CallerID {
			return apperror.New("FORBIDDEN", "reservation belongs to another driver", 403)
		}

		if reservation.Status != constants.StatusCheckedOut {
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
	paymentID, err := uc.paymentClient.ProcessPayment(ctx, billingRecord.ID, billingRecord.TotalAmount, constants.PaymentMethodQRIS, paymentIdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("complete checkout process payment: %w", err)
	}

	// Transition to completed and release spot atomically
	if err := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		now := time.Now()
		reservation.Status = constants.StatusCompleted
		reservation.UpdatedAt = now
		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, constants.SpotStatusAvailable)
	}); err != nil {
		slog.Error("failed to complete reservation after payment",
			slog.String("reservation_id", reservation.ID),
			slog.Any("error", err))
		return nil, fmt.Errorf("complete checkout release spot: %w", err)
	}

	// Publish events (best-effort)
	uc.publishSpotUpdated(ctx, reservation.SpotID, constants.SpotStatusAvailable)
	uc.publishAnalyticsEvent(ctx, pkgnats.SubjectReservationAnalyticsCompleted, reservation, "completed")

	return &model.CheckOutResponse{
		Reservation:  reservation,
		TotalAmount:  billingRecord.TotalAmount,
		BillingID:    billingRecord.ID,
		PaymentID:    paymentID,
		BookingFee:   billingRecord.BookingFee,
		ParkingFee:   billingRecord.ParkingFee,
		OvernightFee: billingRecord.OvernightFee,
	}, nil
}

// ExpireReservation transitions a confirmed reservation to expired and releases
// the spot. Uses SELECT FOR UPDATE to prevent TOCTOU races.
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

		if err := model.ValidateTransition(reservation.Status, constants.StatusExpired); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = constants.StatusExpired
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, constants.SpotStatusAvailable)
	}); err != nil {
		return err
	}

	// No additional no-show penalty is applied. Per PRD, the booking fee
	// (5,000 IDR, already charged at confirmation) is the only cost the
	// driver forfeits when a reservation expires without check-in.

	// Publish events (best-effort)
	uc.publishSpotUpdated(ctx, reservation.SpotID, constants.SpotStatusAvailable)
	uc.publishAnalyticsEvent(ctx, pkgnats.SubjectReservationAnalyticsExpired, reservation, "expired")

	return nil
}

// FailReservation transitions a waiting_payment reservation to failed, releases
// the spot. Called by the payment timeout worker for reservations that exceeded
// the payment window. Uses SELECT FOR UPDATE to prevent TOCTOU races.
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

		if err := model.ValidateTransition(reservation.Status, constants.StatusFailed); err != nil {
			return apperror.BadRequest(err.Error())
		}

		now := time.Now()
		reservation.Status = constants.StatusFailed
		reservation.UpdatedAt = now

		if err := uc.repo.UpdateReservationTx(ctx, tx, reservation); err != nil {
			return err
		}
		return uc.repo.UpdateSpotStatusTx(ctx, tx, reservation.SpotID, constants.SpotStatusAvailable)
	}); err != nil {
		return err
	}

	// Publish events (best-effort)
	uc.publishSpotUpdated(ctx, reservation.SpotID, constants.SpotStatusAvailable)
	uc.publishAnalyticsEvent(ctx, pkgnats.SubjectReservationAnalyticsFailed, reservation, "failed")

	return nil
}

// ListByDriver retrieves reservations for a driver with optional status filter.
func (uc *reservationUsecase) ListByDriver(ctx context.Context, driverID string, status string) ([]*model.Reservation, error) {
	return uc.repo.ListByDriverID(ctx, driverID, status)
}

// publishSpotUpdated publishes a spot status change event (best-effort).
func (uc *reservationUsecase) publishSpotUpdated(ctx context.Context, spotID, status string) {
	if uc.eventPublisher == nil {
		return
	}
	spot, err := uc.repo.GetSpotByID(ctx, spotID)
	if err != nil {
		slog.Error("failed to get spot for event publishing",
			slog.String("spot_id", spotID),
			slog.Any("error", err))
		return
	}
	event := gateway.SpotUpdatedEvent{
		SpotID:      spot.ID,
		FloorNumber: spot.FloorNumber,
		SpotNumber:  spot.SpotNumber,
		VehicleType: spot.VehicleType,
		SpotCode:    spot.SpotCode,
		Status:      status,
		UpdatedAt:   time.Now(),
	}
	if err := uc.eventPublisher.PublishSpotUpdated(ctx, event); err != nil {
		slog.Error("failed to publish spot updated event",
			slog.String("spot_id", spotID),
			slog.String("status", status),
			slog.Any("error", err))
	}
}

// publishAnalyticsEvent publishes a reservation analytics event (best-effort).
func (uc *reservationUsecase) publishAnalyticsEvent(ctx context.Context, subject string, reservation *model.Reservation, status string) {
	if uc.eventPublisher == nil {
		return
	}
	event := gateway.ReservationEvent{
		ReservationID: reservation.ID,
		DriverID:      reservation.DriverID,
		SpotID:        reservation.SpotID,
		VehicleType:   reservation.VehicleType,
		Status:        status,
		Timestamp:     time.Now(),
	}
	if err := uc.eventPublisher.PublishReservationEvent(ctx, subject, event); err != nil {
		slog.Error("failed to publish reservation analytics event",
			slog.String("reservation_id", reservation.ID),
			slog.String("subject", subject),
			slog.Any("error", err))
	}
}
