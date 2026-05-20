// module using sqlx with parameterized queries for SQL injection prevention.
// - Use keyed fields in struct literals to prevent breakages during refactors
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/payment/model"
	"parkir-pintar/pkg/database"
)

var ErrNotFound = errors.New("payment not found")

var ErrConflict = errors.New("conflict: duplicate record")

var ErrStatusMismatch = errors.New("status mismatch: concurrent modification")

type Repository interface {
	CreatePayment(ctx context.Context, payment *model.Payment) error
	GetByIdempotencyKey(ctx context.Context, key string) (*model.Payment, error)
	UpdatePayment(ctx context.Context, payment *model.Payment) error
	UpdatePaymentWithStatusCheck(ctx context.Context, payment *model.Payment, expectedStatus string) error
	GetByID(ctx context.Context, id string) (*model.Payment, error)
	GetByBillingID(ctx context.Context, billingID string) (*model.Payment, error)
}

type sqlxRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

func (r *sqlxRepository) CreatePayment(ctx context.Context, payment *model.Payment) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO payments (id, billing_id, amount, payment_method, payment_gateway,
		 transaction_ref, idempotency_key, status, paid_at, created_at, updated_at)
		 VALUES (:id, :billing_id, :amount, :payment_method, :payment_gateway,
		 :transaction_ref, :idempotency_key, :status, :paid_at, :created_at, :updated_at)`,
		payment,
	)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return ErrConflict
		}
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (r *sqlxRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.GetContext(ctx, &payment, "SELECT * FROM payments WHERE idempotency_key = $1", key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: idempotency_key=%s", ErrNotFound, key)
		}
		return nil, fmt.Errorf("get payment by idempotency_key: %w", err)
	}
	return &payment, nil
}

func (r *sqlxRepository) UpdatePayment(ctx context.Context, payment *model.Payment) error {
	result, err := r.db.NamedExecContext(ctx,
		`UPDATE payments SET status = :status, transaction_ref = :transaction_ref,
		 paid_at = :paid_at, idempotency_key = :idempotency_key, updated_at = :updated_at
		 WHERE id = :id`,
		payment,
	)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update payment rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update payment: %w: id=%s", ErrNotFound, payment.ID)
	}
	return nil
}

// if the payment's current status matches expectedStatus. This prevents double-refund
// and other race conditions.
func (r *sqlxRepository) UpdatePaymentWithStatusCheck(ctx context.Context, payment *model.Payment, expectedStatus string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE payments SET status = $1, transaction_ref = $2,
		 paid_at = $3, idempotency_key = $4, updated_at = $5
		 WHERE id = $6 AND status = $7`,
		payment.Status, payment.TransactionRef, payment.PaidAt,
		payment.IdempotencyKey, payment.UpdatedAt, payment.ID, expectedStatus,
	)
	if err != nil {
		return fmt.Errorf("update payment with status check: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update payment with status check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrStatusMismatch
	}
	return nil
}

func (r *sqlxRepository) GetByID(ctx context.Context, id string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.GetContext(ctx, &payment, "SELECT * FROM payments WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("get payment by id: %w", err)
	}
	return &payment, nil
}

func (r *sqlxRepository) GetByBillingID(ctx context.Context, billingID string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.GetContext(ctx, &payment, "SELECT * FROM payments WHERE billing_id = $1", billingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: billing_id=%s", ErrNotFound, billingID)
		}
		return nil, fmt.Errorf("get payment by billing_id: %w", err)
	}
	return &payment, nil
}
