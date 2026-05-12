package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/payment/model"
)

func setupPaymentTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return sqlx.NewDb(db, "sqlmock"), mock
}

func TestGetByIdempotencyKey_NotFound(t *testing.T) {
	db, mock := setupPaymentTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM payments WHERE idempotency_key = $1")).
		WithArgs("key-pay-1").
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.GetByIdempotencyKey(context.Background(), "key-pay-1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	db, mock := setupPaymentTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM payments WHERE id = $1")).
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.GetByID(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByBillingID_Found(t *testing.T) {
	db, mock := setupPaymentTestDB(t)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "billing_id", "amount", "payment_method", "payment_gateway",
		"transaction_ref", "idempotency_key", "status", "paid_at", "created_at", "updated_at",
	}).AddRow(
		"pay-1", "bill-1", 5000, "qris", "midtrans",
		"txn-123", "key-pay-1", "success", now, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM payments WHERE billing_id = $1")).
		WithArgs("bill-1").
		WillReturnRows(rows)

	repo := NewRepository(db)
	result, err := repo.GetByBillingID(context.Background(), "bill-1")

	assert.NoError(t, err)
	assert.Equal(t, "pay-1", result.ID)
	assert.Equal(t, int64(5000), result.Amount)
	assert.Equal(t, "success", result.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePayment(t *testing.T) {
	db, mock := setupPaymentTestDB(t)
	defer db.Close()

	now := time.Now()
	payment := &model.Payment{
		ID:             "pay-1",
		BillingID:      "bill-1",
		Amount:         5000,
		PaymentMethod:  "qris",
		PaymentGateway: "midtrans",
		TransactionRef: "txn-123",
		IdempotencyKey: "key-pay-1",
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO payments`,
	)).WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewRepository(db)
	err := repo.CreatePayment(context.Background(), payment)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByBillingID_NotFound(t *testing.T) {
	db, mock := setupPaymentTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM payments WHERE billing_id = $1")).
		WithArgs("bill-nonexistent").
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.GetByBillingID(context.Background(), "bill-nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
