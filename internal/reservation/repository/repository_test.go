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

	"parkir-pintar/internal/reservation/model"
)

func setupTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return sqlx.NewDb(db, "sqlmock"), mock
}

func TestFindByIdempotencyKey_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM reservations WHERE idempotency_key = $1")).
		WithArgs("key-123").
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.FindByIdempotencyKey(context.Background(), "key-123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByIdempotencyKey_Found(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "spot_id", "vehicle_type", "assignment_mode",
		"status", "idempotency_key", "confirmed_at", "expires_at",
		"checked_in_at", "checked_out_at", "cancelled_at", "created_at", "updated_at",
	}).AddRow(
		"res-1", "driver-1", "spot-1", "car", "system_assigned",
		"confirmed", "key-123", now, now.Add(time.Hour),
		nil, nil, nil, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM reservations WHERE idempotency_key = $1")).
		WithArgs("key-123").
		WillReturnRows(rows)

	repo := NewRepository(db)
	result, err := repo.FindByIdempotencyKey(context.Background(), "key-123")

	assert.NoError(t, err)
	assert.Equal(t, "res-1", result.ID)
	assert.Equal(t, "key-123", result.IdempotencyKey)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindAvailableSpot_Found(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "floor_number", "spot_number", "vehicle_type", "spot_code",
		"status", "created_at", "updated_at",
	}).AddRow(
		"spot-1", 1, 1, "car", "F1-C-001",
		"available", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM parking_spots WHERE vehicle_type = $1 AND status = 'available' ORDER BY floor_number, spot_number LIMIT 1 FOR UPDATE SKIP LOCKED`,
	)).WithArgs("car").WillReturnRows(rows)

	repo := NewRepository(db)
	result, err := repo.FindAvailableSpot(context.Background(), "car")

	assert.NoError(t, err)
	assert.Equal(t, "spot-1", result.ID)
	assert.Equal(t, "car", result.VehicleType)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindAvailableSpot_NoSpots(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM parking_spots WHERE vehicle_type = $1 AND status = 'available' ORDER BY floor_number, spot_number LIMIT 1 FOR UPDATE SKIP LOCKED`,
	)).WithArgs("motorcycle").WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.FindAvailableSpot(context.Background(), "motorcycle")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSpotStatusTx(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE parking_spots SET status = $1, updated_at = NOW() WHERE id = $2",
	)).WithArgs("reserved", "spot-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewRepository(db)
	err := repo.WithTransaction(context.Background(), func(tx *sqlx.Tx) error {
		return repo.UpdateSpotStatusTx(context.Background(), tx, "spot-1", "reserved")
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByIDForUpdate(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "spot_id", "vehicle_type", "assignment_mode",
		"status", "idempotency_key", "confirmed_at", "expires_at",
		"checked_in_at", "checked_out_at", "cancelled_at", "created_at", "updated_at",
	}).AddRow(
		"res-1", "driver-1", "spot-1", "car", "system_assigned",
		"confirmed", "key-123", now, now.Add(time.Hour),
		nil, nil, nil, now, now,
	)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM reservations WHERE id = $1 FOR UPDATE",
	)).WithArgs("res-1").WillReturnRows(rows)
	mock.ExpectCommit()

	repo := NewRepository(db)
	var result *model.Reservation
	err := repo.WithTransaction(context.Background(), func(tx *sqlx.Tx) error {
		var txErr error
		result, txErr = repo.GetByIDForUpdate(context.Background(), tx, "res-1")
		return txErr
	})

	assert.NoError(t, err)
	assert.Equal(t, "res-1", result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindExpiredReservations_Empty(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "spot_id", "vehicle_type", "assignment_mode",
		"status", "idempotency_key", "confirmed_at", "expires_at",
		"checked_in_at", "checked_out_at", "cancelled_at", "created_at", "updated_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM reservations WHERE status = 'confirmed' AND expires_at < NOW()",
	)).WillReturnRows(rows)

	repo := NewRepository(db)
	result, err := repo.FindExpiredReservations(context.Background())

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM reservations WHERE id = $1",
	)).WithArgs("nonexistent").WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.GetByID(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
