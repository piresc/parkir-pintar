// Package testhelpers provides shared utilities for both E2E test layers.
//
// Best practices applied (from Go testify testing standards):
// - Use fmt.Errorf with %w for error wrapping so callers can inspect root cause
// - Use parameterized queries ($1, $2, …) to prevent SQL injection
// - Keep helpers focused on a single responsibility
// - Use keyed struct fields for clarity
package testhelpers

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// phoneCounter is an atomically incremented counter used to generate unique
// phone numbers across concurrent test executions, avoiding UNIQUE constraint
// violations on the drivers.phone column.
var phoneCounter uint64

// InsertTestDriver inserts a driver row into the drivers table with a generated
// UUID, a unique phone number, and sensible defaults for name and email.
// It returns the driver's UUID string.
//
// vehicleType is accepted for API compatibility but not stored on the driver.
func InsertTestDriver(ctx context.Context, db *sqlx.DB, _ string) (string, error) {
	id := uuid.New().String()
	seq := atomic.AddUint64(&phoneCounter, 1)
	phone := fmt.Sprintf("+628%09d", seq)
	name := fmt.Sprintf("Test Driver %d", seq)
	email := fmt.Sprintf("driver%d@test.parkir-pintar.local", seq)

	_, err := db.ExecContext(ctx,
		`INSERT INTO drivers (id, name, phone, email)
		 VALUES ($1, $2, $3, $4)`,
		id, name, phone, email,
	)
	if err != nil {
		return "", fmt.Errorf("insert test driver: %w", err)
	}

	return id, nil
}
