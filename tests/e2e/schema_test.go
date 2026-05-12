// Package e2e_test — database schema validation tests.
//
// Best practices applied (from Go testify testing standards):
// - Use t.Helper() in assertion helpers so failures report the caller's line
// - Use require for assertions that must pass to continue (fail-fast)
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
// - Keep tests isolated, repeatable, and focused on a single responsibility
// - Do not mock the database — these are integration tests against real PostgreSQL
package e2e_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchema_ShouldHaveAllTables verifies that all 7 tables defined in
// PRD Section 16 exist in the database after migration.
//
// Validates: Requirement 2.1
func TestSchema_ShouldHaveAllTables(t *testing.T) {
	// Arrange
	ctx := context.Background()
	expectedTables := []string{
		"reservation.drivers",
		"reservation.parking_spots",
		"reservation.reservations",
		"billing.billing_records",
		"billing.penalties",
		"payment.payments",
		"presence.presence_logs",
		"search.spot_read_model",
	}

	// Act
	var tables []string
	err := env.db.SelectContext(ctx, &tables,
		`SELECT table_schema || '.' || table_name AS table_name
		 FROM information_schema.tables
		 WHERE table_schema IN ('reservation', 'billing', 'payment', 'presence', 'search')
		   AND table_type = 'BASE TABLE'
		 ORDER BY table_name`)

	// Assert
	require.NoError(t, err, "failed to query information_schema.tables")
	for _, expected := range expectedTables {
		assert.Contains(t, tables, expected, "missing table: %s", expected)
	}
}

// TestSchema_ShouldHave400ParkingSpots verifies that parking_spots contains
// exactly 400 rows (150 car + 250 motorcycle) after migration seed.
//
// Validates: Requirement 2.2
func TestSchema_ShouldHave400ParkingSpots(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// Act
	var total int
	err := env.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM parking_spots").Scan(&total)

	// Assert
	require.NoError(t, err, "failed to count parking_spots")
	assert.Equal(t, 400, total, "parking_spots should have exactly 400 rows")

	// Verify breakdown by vehicle type
	var carCount, motoCount int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM parking_spots WHERE vehicle_type = 'car'").Scan(&carCount)
	require.NoError(t, err)
	assert.Equal(t, 150, carCount, "should have 150 car spots")

	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM parking_spots WHERE vehicle_type = 'motorcycle'").Scan(&motoCount)
	require.NoError(t, err)
	assert.Equal(t, 250, motoCount, "should have 250 motorcycle spots")
}


// TestSchema_ShouldHaveCorrectFloorDistribution verifies that parking spots
// are distributed as 5 floors × 30 car + 5 floors × 50 motorcycle.
//
// Validates: Requirement 2.3
func TestSchema_ShouldHaveCorrectFloorDistribution(t *testing.T) {
	// Arrange
	ctx := context.Background()

	type floorCount struct {
		FloorNumber int    `db:"floor_number"`
		VehicleType string `db:"vehicle_type"`
		Count       int    `db:"count"`
	}

	// Act
	var counts []floorCount
	err := env.db.SelectContext(ctx, &counts,
		`SELECT floor_number, vehicle_type, COUNT(*) AS count
		 FROM parking_spots
		 GROUP BY floor_number, vehicle_type
		 ORDER BY floor_number, vehicle_type`)

	// Assert
	require.NoError(t, err, "failed to query floor distribution")

	for _, fc := range counts {
		switch fc.VehicleType {
		case "car":
			assert.Equal(t, 30, fc.Count,
				"floor %d should have 30 car spots, got %d", fc.FloorNumber, fc.Count)
		case "motorcycle":
			assert.Equal(t, 50, fc.Count,
				"floor %d should have 50 motorcycle spots, got %d", fc.FloorNumber, fc.Count)
		default:
			t.Errorf("unexpected vehicle_type %q on floor %d", fc.VehicleType, fc.FloorNumber)
		}
	}

	// Verify we have exactly 5 floors for each type
	carFloors := 0
	motoFloors := 0
	for _, fc := range counts {
		if fc.VehicleType == "car" {
			carFloors++
		} else if fc.VehicleType == "motorcycle" {
			motoFloors++
		}
	}
	assert.Equal(t, 5, carFloors, "should have 5 floors with car spots")
	assert.Equal(t, 5, motoFloors, "should have 5 floors with motorcycle spots")
}

// TestSchema_ShouldHaveUniqueSpotCodes verifies that every spot_code is unique
// and follows the format F{1-5}-{C|M}-{001-050}.
//
// Validates: Requirement 2.4
func TestSchema_ShouldHaveUniqueSpotCodes(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// Act
	var codes []string
	err := env.db.SelectContext(ctx, &codes,
		"SELECT spot_code FROM parking_spots ORDER BY spot_code")

	// Assert
	require.NoError(t, err, "failed to query spot_codes")
	require.Len(t, codes, 400, "should have 400 spot codes")

	// Verify uniqueness
	seen := make(map[string]bool, len(codes))
	for _, code := range codes {
		assert.False(t, seen[code], "duplicate spot_code: %s", code)
		seen[code] = true
	}

	// Verify format: F{1-5}-{C|M}-{001-050}
	for _, code := range codes {
		assert.Regexp(t, `^F[1-5]-[CM]-\d{3}$`, code,
			"spot_code %q does not match format F{1-5}-{C|M}-{001-050}", code)
	}
}

// TestSchema_ShouldHaveActiveSpotIndex verifies that the partial unique index
// idx_reservations_active_spot exists in the database.
//
// Validates: Requirement 2.5
func TestSchema_ShouldHaveActiveSpotIndex(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// Act
	var indexDef string
	err := env.db.QueryRowContext(ctx,
		`SELECT indexdef FROM pg_indexes
		 WHERE indexname = 'idx_reservations_active_spot'`).Scan(&indexDef)

	// Assert
	require.NoError(t, err, "idx_reservations_active_spot should exist in pg_indexes")
	assert.Contains(t, indexDef, "UNIQUE", "index should be a UNIQUE index")
	assert.Contains(t, indexDef, "spot_id", "index should be on spot_id column")
}

// TestSchema_ShouldHaveAllRequiredIndexes verifies that all 7 indexes from
// PRD Section 16.3 exist in the database.
//
// Validates: Requirement 2.6
func TestSchema_ShouldHaveAllRequiredIndexes(t *testing.T) {
	// Arrange
	ctx := context.Background()
	expectedIndexes := []string{
		"idx_reservations_active_spot",
		"idx_reservations_driver",
		"idx_parking_spots_availability",
		"idx_reservations_expiry",
		"idx_reservations_stale_payment",
		"idx_billing_reservation",
		"idx_payments_billing",
		"idx_presence_reservation_time",
		"idx_search_spot_availability",
		"idx_search_spot_floor",
	}

	// Act
	var indexes []string
	err := env.db.SelectContext(ctx, &indexes,
		`SELECT indexname FROM pg_indexes
		 WHERE schemaname IN ('reservation', 'billing', 'payment', 'presence', 'search')
		 ORDER BY indexname`)

	// Assert
	require.NoError(t, err, "failed to query pg_indexes")
	for _, expected := range expectedIndexes {
		assert.Contains(t, indexes, expected, "missing index: %s", expected)
	}
}
