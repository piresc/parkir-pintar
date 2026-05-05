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
// UUID, a unique phone number, and sensible defaults for name, email, and
// vehicle_plate. It returns the driver's UUID string.
//
// vehicleType must be "car" or "motorcycle" (enforced by the DB CHECK constraint).
//
// Validates: Requirements 1.3, 5.1 — test driver creation for E2E scenarios.
func InsertTestDriver(ctx context.Context, db *sqlx.DB, vehicleType string) (string, error) {
	id := uuid.New().String()
	seq := atomic.AddUint64(&phoneCounter, 1)
	phone := fmt.Sprintf("+628%09d", seq)
	name := fmt.Sprintf("Test Driver %d", seq)
	email := fmt.Sprintf("driver%d@test.parkir-pintar.local", seq)
	plate := fmt.Sprintf("B%04dTST", seq%10000)

	_, err := db.ExecContext(ctx,
		`INSERT INTO drivers (id, name, phone, email, vehicle_type, vehicle_plate)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, name, phone, email, vehicleType, plate,
	)
	if err != nil {
		return "", fmt.Errorf("insert test driver: %w", err)
	}

	return id, nil
}
