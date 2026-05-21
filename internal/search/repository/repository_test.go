package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/search"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "sqlmock"), mock
}

// --- Repository (read queries) ---

func TestGetAvailabilityByVehicleType_WithFilter(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"floor_number", "available_car", "available_moto", "total_car", "total_moto"}).
		AddRow(1, 5, 3, 10, 8).
		AddRow(2, 2, 4, 6, 7)

	mock.ExpectQuery(`SELECT`).WithArgs("car").WillReturnRows(rows)

	result, err := repo.GetAvailabilityByVehicleType(context.Background(), "car")

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].FloorNumber)
	assert.Equal(t, 5, result[0].AvailableCar)
	assert.Equal(t, 2, result[1].FloorNumber)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAvailabilityByVehicleType_WithoutFilter(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"floor_number", "available_car", "available_moto", "total_car", "total_moto"}).
		AddRow(1, 5, 3, 10, 8)

	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

	result, err := repo.GetAvailabilityByVehicleType(context.Background(), "")

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, result[0].FloorNumber)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAvailabilityByVehicleType_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	mock.ExpectQuery(`SELECT`).WithArgs("car").WillReturnError(sql.ErrConnDone)

	result, err := repo.GetAvailabilityByVehicleType(context.Background(), "car")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get availability")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFloorSpots_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"id", "spot_code", "vehicle_type", "status", "floor_number", "spot_number"}).
		AddRow("spot-1", "A-01", "car", "available", 1, 1).
		AddRow("spot-2", "A-02", "motorcycle", "occupied", 1, 2)

	mock.ExpectQuery(`SELECT .+ FROM spot_read_model`).WithArgs(1).WillReturnRows(rows)

	result, err := repo.GetFloorSpots(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "spot-1", result[0].ID)
	assert.Equal(t, "A-01", result[0].SpotCode)
	assert.Equal(t, "car", result[0].VehicleType)
	assert.Equal(t, "available", result[0].Status)
	assert.Equal(t, 1, result[0].FloorNumber)
	assert.Equal(t, 1, result[0].SpotNumber)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFloorSpots_Empty(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"id", "spot_code", "vehicle_type", "status", "floor_number", "spot_number"})
	mock.ExpectQuery(`SELECT .+ FROM spot_read_model`).WithArgs(99).WillReturnRows(rows)

	result, err := repo.GetFloorSpots(context.Background(), 99)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFloorSpots_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	mock.ExpectQuery(`SELECT .+ FROM spot_read_model`).WithArgs(1).WillReturnError(sql.ErrConnDone)

	result, err := repo.GetFloorSpots(context.Background(), 1)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get floor spots")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSpotByID_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"id", "spot_code", "vehicle_type", "status", "floor_number", "spot_number"}).
		AddRow("spot-1", "A-01", "car", "available", 1, 1)

	mock.ExpectQuery(`SELECT .+ FROM spot_read_model`).WithArgs("spot-1").WillReturnRows(rows)

	result, err := repo.GetSpotByID(context.Background(), "spot-1")

	require.NoError(t, err)
	assert.Equal(t, "spot-1", result.ID)
	assert.Equal(t, "A-01", result.SpotCode)
	assert.Equal(t, "car", result.VehicleType)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSpotByID_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	mock.ExpectQuery(`SELECT .+ FROM spot_read_model`).WithArgs("nonexistent").WillReturnError(sql.ErrNoRows)

	result, err := repo.GetSpotByID(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSpotByID_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepository(db)

	mock.ExpectQuery(`SELECT .+ FROM spot_read_model`).WithArgs("spot-1").WillReturnError(sql.ErrConnDone)

	result, err := repo.GetSpotByID(context.Background(), "spot-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get spot by id")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- ReadModelRepository ---

func TestUpsertSpot_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewReadModelRepository(db)

	spot := search.SpotData{
		ID:          "spot-1",
		FloorNumber: 1,
		SpotNumber:  1,
		VehicleType: "car",
		SpotCode:    "A-01",
		Status:      "available",
	}

	mock.ExpectExec(`INSERT INTO spot_read_model`).
		WithArgs(spot.ID, spot.FloorNumber, spot.SpotNumber, spot.VehicleType, spot.SpotCode, spot.Status).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpsertSpot(context.Background(), spot)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSpot_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewReadModelRepository(db)

	spot := search.SpotData{
		ID:          "spot-1",
		FloorNumber: 1,
		SpotNumber:  1,
		VehicleType: "car",
		SpotCode:    "A-01",
		Status:      "available",
	}

	mock.ExpectExec(`INSERT INTO spot_read_model`).
		WithArgs(spot.ID, spot.FloorNumber, spot.SpotNumber, spot.VehicleType, spot.SpotCode, spot.Status).
		WillReturnError(sql.ErrConnDone)

	err := repo.UpsertSpot(context.Background(), spot)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert spot read model")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteSpot_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewReadModelRepository(db)

	mock.ExpectExec(`DELETE FROM spot_read_model WHERE id`).
		WithArgs("spot-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteSpot(context.Background(), "spot-1")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteSpot_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewReadModelRepository(db)

	mock.ExpectExec(`DELETE FROM spot_read_model WHERE id`).
		WithArgs("spot-1").
		WillReturnError(sql.ErrConnDone)

	err := repo.DeleteSpot(context.Background(), "spot-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete spot read model")
	assert.NoError(t, mock.ExpectationsWereMet())
}
