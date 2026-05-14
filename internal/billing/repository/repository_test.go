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

	"parkir-pintar/internal/billing/model"
)

func setupBillingTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return sqlx.NewDb(db, "sqlmock"), mock
}

func TestGetByReservationID_NotFound(t *testing.T) {
	db, mock := setupBillingTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM billing_records WHERE reservation_id = $1")).
		WithArgs("res-1").
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.GetByReservationID(context.Background(), "res-1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByReservationID_Found(t *testing.T) {
	db, mock := setupBillingTestDB(t)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "reservation_id", "booking_fee", "parking_fee", "overnight_fee",
		"cancellation_fee", "penalty_amount", "total_amount", "duration_minutes",
		"billed_hours", "is_overnight", "idempotency_key", "status", "created_at", "updated_at",
	}).AddRow(
		"bill-1", "res-1", 5000, 10000, 20000,
		0, 0, 35000, 120,
		2, true, "key-123", "invoiced", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM billing_records WHERE reservation_id = $1")).
		WithArgs("res-1").
		WillReturnRows(rows)

	repo := NewRepository(db)
	result, err := repo.GetByReservationID(context.Background(), "res-1")

	assert.NoError(t, err)
	assert.Equal(t, "bill-1", result.ID)
	assert.Equal(t, int64(35000), result.TotalAmount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByIdempotencyKey_NotFound(t *testing.T) {
	db, mock := setupBillingTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM billing_records WHERE idempotency_key = $1")).
		WithArgs("key-456").
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.GetByIdempotencyKey(context.Background(), "key-456")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddPenaltyAmount_NotFound(t *testing.T) {
	db, mock := setupBillingTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		`UPDATE billing_records SET penalty_amount = penalty_amount + $1, total_amount = booking_fee + parking_fee + overnight_fee + cancellation_fee + (penalty_amount + $1), updated_at = NOW() WHERE reservation_id = $2 RETURNING *`,
	)).WithArgs(int64(10000), "res-nonexistent").WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.AddPenaltyAmount(context.Background(), "res-nonexistent", 10000)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateBillingRecord(t *testing.T) {
	db, mock := setupBillingTestDB(t)
	defer db.Close()

	now := time.Now()
	record := &model.BillingRecord{
		ID:              "bill-1",
		ReservationID:   "res-1",
		BookingFee:      5000,
		ParkingFee:      0,
		OvernightFee:    0,
		CancellationFee: 0,
		PenaltyAmount:   0,
		TotalAmount:     5000,
		DurationMinutes: 0,
		BilledHours:     0,
		IsOvernight:     false,
		IdempotencyKey:  "key-bill-1",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO billing_records`,
	)).WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewRepository(db)
	err := repo.CreateBillingRecord(context.Background(), record)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
