package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/analytics/model"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "sqlmock"), mock
}

func TestGetHourlyStats_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"hour", "day_of_week", "avg_occupancy", "avg_reservations", "peak_score"}).
		AddRow(8, 1, 0.85, 20, 4.5).
		AddRow(9, 1, 0.90, 25, 5.0).
		AddRow(14, 2, 0.40, 8, 1.5)

	mock.ExpectQuery(`SELECT`).WithArgs(startDate, endDate).WillReturnRows(rows)

	result, err := repo.GetHourlyStats(context.Background(), startDate, endDate)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, 8, result[0].Hour)
	assert.Equal(t, 1, result[0].DayOfWeek)
	assert.Equal(t, 0.85, result[0].AvgOccupancy)
	assert.Equal(t, 20, result[0].AvgReservations)
	assert.Equal(t, 4.5, result[0].PeakScore)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetHourlyStats_Empty(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"hour", "day_of_week", "avg_occupancy", "avg_reservations", "peak_score"})
	mock.ExpectQuery(`SELECT`).WithArgs(startDate, endDate).WillReturnRows(rows)

	result, err := repo.GetHourlyStats(context.Background(), startDate, endDate)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetHourlyStats_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT`).WithArgs(startDate, endDate).WillReturnError(sql.ErrConnDone)

	result, err := repo.GetHourlyStats(context.Background(), startDate, endDate)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get hourly stats")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDailyOccupancy_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	date1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"date", "occupied_spots", "total_spots", "avg_occupancy"}).
		AddRow(date1, 40, 100, 0.4).
		AddRow(date2, 60, 100, 0.6)

	mock.ExpectQuery(`WITH daily AS`).WithArgs(30).WillReturnRows(rows)

	result, err := repo.GetDailyOccupancy(context.Background(), 30)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, date1, result[0].Date)
	assert.Equal(t, 40, result[0].OccupiedSpots)
	assert.Equal(t, 100, result[0].TotalSpots)
	assert.Equal(t, 0.4, result[0].AvgOccupancy)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDailyOccupancy_Empty(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"date", "occupied_spots", "total_spots", "avg_occupancy"})
	mock.ExpectQuery(`WITH daily AS`).WithArgs(7).WillReturnRows(rows)

	result, err := repo.GetDailyOccupancy(context.Background(), 7)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDailyOccupancy_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	mock.ExpectQuery(`WITH daily AS`).WithArgs(30).WillReturnError(sql.ErrConnDone)

	result, err := repo.GetDailyOccupancy(context.Background(), 30)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get daily occupancy")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordEvent_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	event := model.ReservationEvent{
		ReservationID: "res-123",
		DriverID:      "driver-1",
		SpotID:        "spot-1",
		VehicleType:   "car",
		Status:        "confirmed",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	mock.ExpectExec(`INSERT INTO reservation.reservation_events`).
		WithArgs(event.ReservationID, event.DriverID, event.SpotID, event.VehicleType, event.Status, event.Timestamp).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.RecordEvent(context.Background(), event)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordEvent_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	event := model.ReservationEvent{
		ReservationID: "res-123",
		DriverID:      "driver-1",
		SpotID:        "spot-1",
		VehicleType:   "car",
		Status:        "confirmed",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	mock.ExpectExec(`INSERT INTO reservation.reservation_events`).
		WithArgs(event.ReservationID, event.DriverID, event.SpotID, event.VehicleType, event.Status, event.Timestamp).
		WillReturnError(sql.ErrConnDone)

	err := repo.RecordEvent(context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "record event")
	assert.NoError(t, mock.ExpectationsWereMet())
}
