// Package worker provides background workers for the reservation domain module.
package worker

import (
	"context"
	"log/slog"
	"time"

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/internal/reservation/repository"
	"parkir-pintar/internal/reservation/usecase"
)

// RunExpiryWorker periodically scans for expired reservations and processes them.
// It runs until ctx is cancelled.
//
// Postconditions:
//   - All confirmed reservations past expires_at are transitioned to "expired"
//   - Spots are released back to "available"
//   - The booking fee (5,000 IDR, already charged at confirmation) is the only
//     cost the driver forfeits — no additional no-show penalty is applied
//   - reservation.expired events are published
//
// Loop Invariant:
//   - Each iteration processes only reservations with status='confirmed' AND expires_at < NOW()
//   - Previously expired reservations are not re-processed (status is already 'expired')
func RunExpiryWorker(ctx context.Context, interval time.Duration, repo repository.Repository, uc usecase.Usecase) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("expiry worker started", slog.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			slog.Info("expiry worker stopped")
			return
		case <-ticker.C:
			expired, err := repo.FindExpiredReservations(ctx)
			if err != nil {
				slog.Error("expiry worker: find expired", slog.Any("error", err))
				continue
			}
			for _, r := range expired {
				if err := uc.ExpireReservation(ctx, &model.ExpireReservationRequest{
					ReservationID: r.ID,
				}); err != nil {
					slog.Error("expiry worker: expire reservation",
						slog.String("reservation_id", r.ID),
						slog.Any("error", err))
				}
			}
		}
	}
}

// RunPaymentTimeoutWorker periodically scans for waiting_payment reservations
// that have exceeded the payment timeout window and fails them (releases spot).
// It runs until ctx is cancelled.
//
// Postconditions:
//   - All waiting_payment reservations older than timeoutMinutes are
//     transitioned to "failed"
//   - Spots are released back to "available"
//   - reservation.payment_failed events are published
//
// Loop Invariant:
//   - Each iteration processes only reservations with status='waiting_payment'
//     AND created_at < NOW() - timeoutMinutes
func RunPaymentTimeoutWorker(ctx context.Context, interval time.Duration, repo repository.Repository, uc usecase.Usecase, timeoutMinutes int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("payment timeout worker started",
		slog.Duration("interval", interval),
		slog.Int("timeout_minutes", timeoutMinutes))

	for {
		select {
		case <-ctx.Done():
			slog.Info("payment timeout worker stopped")
			return
		case <-ticker.C:
			stale, err := repo.FindStalePaymentReservations(ctx, timeoutMinutes)
			if err != nil {
				slog.Error("payment timeout worker: find stale", slog.Any("error", err))
				continue
			}
			for _, r := range stale {
				if err := uc.FailReservation(ctx, &model.FailReservationRequest{
					ReservationID: r.ID,
				}); err != nil {
					slog.Error("payment timeout worker: fail reservation",
						slog.String("reservation_id", r.ID),
						slog.Any("error", err))
				}
			}
		}
	}
}
