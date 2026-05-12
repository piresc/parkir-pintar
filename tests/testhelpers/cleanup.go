// Package testhelpers provides shared utilities for both E2E test layers.
//
// Best practices applied (from Go testify testing standards):
// - Use fmt.Errorf with %w for error wrapping so callers can inspect root cause
// - Use parameterized table names via fmt.Sprintf (table names are internal constants, not user input)
// - Keep helpers focused on a single responsibility
// - Document the recommended truncation order for FK-safe cleanup
package testhelpers

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Recommended truncation order respecting foreign-key dependencies.
// Child tables are truncated before parent tables. The parking_spots table
// is intentionally excluded — its 400-row seed data must be preserved
// across all tests.
//
//   1. presence_logs  (FK → reservations)
//   2. penalties      (FK → reservations)
//   3. payments       (FK → billing_records)
//   4. billing_records (FK → reservations)
//   5. reservations   (FK → drivers, parking_spots)
//   6. drivers        (no child FK deps remaining after above)
//
// parking_spots — NEVER truncated (seed data preserved).

// TruncateTables truncates each of the specified tables individually using
// TRUNCATE TABLE ... CASCADE to handle foreign-key constraints. Each table
// is truncated in the order provided, so callers should pass tables in
// dependency order (children before parents).
//
// Validates: Requirement 1.5 — test isolation via table cleanup.
func TruncateTables(ctx context.Context, db *sqlx.DB, tables ...string) error {
	for _, table := range tables {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	return nil
}

// ResetSpots resets all parking spots back to "available" status.
// Call this after truncating reservations to restore spot inventory
// for tests that depend on spot availability.
func ResetSpots(ctx context.Context, db *sqlx.DB) error {
	_, err := db.ExecContext(ctx, "UPDATE parking_spots SET status = 'available'")
	if err != nil {
		return fmt.Errorf("reset parking_spots: %w", err)
	}
	return nil
}
