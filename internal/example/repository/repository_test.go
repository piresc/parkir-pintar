// Package repository provides the data access layer for the example domain.
//
// Best practices applied (from coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - go-sqlmock for SQL mocking without real DB
// - sqlx.NewDb wrapping sqlmock for sqlx compatibility
// - Cleanup via defer to ensure expectations are met
// - Table-driven tests avoided here for clarity; each case is explicit
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

	"parkir-pintar/internal/example/model"
)

// setupSQLMock creates a sqlmock DB and wraps it with sqlx for repository tests.
func setupSQLMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	cleanup := func() {
		_ = db.Close()
		assert.NoError(t, mock.ExpectationsWereMet())
	}
	return sqlxDB, mock, cleanup
}

func TestGetByID_ShouldReturnExample_WhenRecordExists(t *testing.T) {
	// Arrange
	db, mock, cleanup := setupSQLMock(t)
	defer cleanup()

	repo := New(db, nil)
	now := time.Now()
	expectedID := "550e8400-e29b-41d4-a716-446655440000"

	rows := sqlmock.NewRows([]string{"id", "name", "description", "status", "created_at", "updated_at"}).
		AddRow(expectedID, "Test Example", "A test description", "active", now, now)

	mock.ExpectQuery(`SELECT \* FROM examples WHERE id = \$1`).
		WithArgs(expectedID).
		WillReturnRows(rows)

	// Act
	result, err := repo.GetByID(context.Background(), expectedID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedID, result.ID)
	assert.Equal(t, "Test Example", result.Name)
	assert.Equal(t, "active", result.Status)
}

func TestGetByID_ShouldReturnError_WhenRecordNotFound(t *testing.T) {
	// Arrange
	db, mock, cleanup := setupSQLMock(t)
	defer cleanup()

	repo := New(db, nil)

	mock.ExpectQuery(`SELECT \* FROM examples WHERE id = \$1`).
		WithArgs("nonexistent-id").
		WillReturnError(sql.ErrNoRows)

	// Act
	result, err := repo.GetByID(context.Background(), "nonexistent-id")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "example not found")
}

func TestList_ShouldReturnPaginatedResults_WhenRecordsExist(t *testing.T) {
	// Arrange
	db, mock, cleanup := setupSQLMock(t)
	defer cleanup()

	repo := New(db, nil)
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "description", "status", "created_at", "updated_at"}).
		AddRow("id-1", "First", "desc 1", "active", now, now).
		AddRow("id-2", "Second", "desc 2", "inactive", now, now)

	mock.ExpectQuery(`SELECT \* FROM examples ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	// Act
	results, err := repo.List(context.Background(), 10, 0)

	// Assert
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "First", results[0].Name)
	assert.Equal(t, "Second", results[1].Name)
}

func TestCreate_ShouldInsertRecord_WhenValidExampleProvided(t *testing.T) {
	// Arrange
	db, mock, cleanup := setupSQLMock(t)
	defer cleanup()

	repo := New(db, nil)
	now := time.Now()
	example := &model.Example{
		ID:          "new-id",
		Name:        "New Example",
		Description: "New description",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mock.ExpectExec(`INSERT INTO examples`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Act
	err := repo.Create(context.Background(), example)

	// Assert
	assert.NoError(t, err)
}
