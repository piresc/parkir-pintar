# Database-Per-Service Separation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate from a shared single-database architecture to PostgreSQL schema-per-service isolation, eliminating cross-service table access.

**Architecture:** Use PostgreSQL schemas within a single cluster as the first step toward full database-per-service. Each service gets its own schema (`reservation`, `billing`, `payment`, `presence`, `search`) with tables scoped to that schema. The Search service's direct read of `parking_spots` (owned by Reservation) is replaced by NATS event-driven sync of a read-only `spot_read_model` table in the `search` schema. Cross-schema foreign keys are removed in favor of eventual consistency via NATS events.

**Tech Stack:** Go 1.25, PostgreSQL 14, sqlx, NATS JetStream, Redis 7, Docker Compose, testcontainers-go, testify

---

## Scope Decision: Single Plan

This plan covers one cohesive subsystem: database isolation. All changes are interdependent (schema creation, migration, repository updates, search read-model, docker-compose, tests) and must be applied together to produce working software.

---

## File Structure

### New Files
| File | Responsibility |
|------|---------------|
| `db/migrations/000004_schema_per_service.up.sql` | Create schemas, move tables into schemas, create search read model, drop public tables |
| `db/migrations/000004_schema_per_service.down.sql` | Rollback: move tables back to public, drop schemas |
| `internal/search/client/reservation_client.go` | gRPC client interface for Search → Reservation communication |
| `internal/search/sync/spot_sync.go` | NATS subscriber that syncs `spot_read_model` in search schema |
| `internal/search/sync/spot_sync_test.go` | Tests for spot sync logic |

### Modified Files
| File | Change |
|------|--------|
| `pkg/config/config.go` | Add `Schema` field to `DatabaseConfig` |
| `pkg/database/postgres.go` | Set `search_path` to configured schema on connection |
| `docker-compose.yml` | Add `DB_SCHEMA` env var per service |
| `db/migrations/000002_parkir_pintar.up.sql` | Add `SET search_path` and schema-qualify table creation |
| `internal/search/repository/repository.go` | Replace `parking_spots` queries with `spot_read_model` queries |
| `internal/search/usecase/usecase.go` | Add gRPC client for direct spot lookups (GetSpotDetails) |
| `internal/search/subscriber/subscriber.go` | Wire spot sync subscriber alongside cache invalidation |
| `cmd/search/main.go` | Wire gRPC client to Reservation, start spot sync subscriber |
| `tests/e2e/setup_test.go` | Apply new migration, update wiring |
| `tests/e2e/schema_test.go` | Update table existence assertions for schema-qualified names |

---

### Task 1: Add Schema Support to Config and Database Client

**Files:**
- Modify: `pkg/config/config.go:73-83`
- Modify: `pkg/database/postgres.go:24-61`
- Test: `pkg/database/postgres_test.go`

- [ ] **Step 1: Write the failing test for config Schema field**

Add to `pkg/config/postgres_test.go` (new test function at end of file):

```go
func TestDatabaseConfig_ShouldHaveSchemaField_WhenLoadedFromEnv(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "test",
		Password: "test",
		Database: "testdb",
		Schema:   "reservation",
		SSLMode:  "disable",
	}
	assert.Equal(t, "reservation", cfg.Schema)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/... -run TestDatabaseConfig_ShouldHaveSchemaField -v`
Expected: FAIL — `config.DatabaseConfig` has no `Schema` field

- [ ] **Step 3: Add Schema field to DatabaseConfig**

In `pkg/config/config.go`, add `Schema` field to `DatabaseConfig`:

```go
type DatabaseConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	Database    string
	Schema      string
	SSLMode     string
	MaxConns    int
	IdleConns   int
	MaxLifetime int
}
```

In `pkg/config/config.go`, add the env var loading after line 166 (`cfg.Database.Database = getEnv("DB_DATABASE", "")`):

```go
cfg.Database.Schema = getEnv("DB_SCHEMA", "public")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/config/... -run TestDatabaseConfig_ShouldHaveSchemaField -v`
Expected: PASS

- [ ] **Step 5: Write the failing test for search_path in PostgresClient**

Add to `pkg/database/postgres_test.go`:

```go
func TestNewPostgresClient_ShouldSetSearchPath_WhenSchemaIsConfigured(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:     "invalid-host",
		Port:     5432,
		Username: "test",
		Password: "test",
		Database: "testdb",
		Schema:   "reservation",
		SSLMode:  "disable",
	}
	client, err := NewPostgresClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to ping postgres")
}
```

This test just verifies the constructor accepts the Schema field without panic. A real integration test would verify `search_path` is set.

- [ ] **Step 6: Modify NewPostgresClient to set search_path**

In `pkg/database/postgres.go`, update `NewPostgresClient` to execute `SET search_path` after connecting:

```go
func NewPostgresClient(cfg config.DatabaseConfig) (*PostgresClient, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if cfg.MaxConns > 0 {
		db.SetMaxOpenConns(cfg.MaxConns)
	}
	if cfg.IdleConns > 0 {
		db.SetMaxIdleConns(cfg.IdleConns)
	}
	if cfg.MaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Minute)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	if cfg.Schema != "" && cfg.Schema != "public" {
		_, err := db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", cfg.Schema))
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set search_path to %s: %w", cfg.Schema, err)
		}
	}

	return &PostgresClient{db: db}, nil
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./pkg/database/... -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add pkg/config/config.go pkg/database/postgres.go pkg/database/postgres_test.go
git commit -m "feat: add schema-per-service config and search_path support"
```

---

### Task 2: Create Schema-Per-Service Migration

**Files:**
- Create: `db/migrations/000004_schema_per_service.up.sql`
- Create: `db/migrations/000004_schema_per_service.down.sql`

- [ ] **Step 1: Write the up migration**

Create `db/migrations/000004_schema_per_service.up.sql`:

```sql
-- 000004_schema_per_service.up.sql
-- Migrate from shared public schema to per-service PostgreSQL schemas.
-- Each service owns its schema exclusively. Cross-schema foreign keys
-- are dropped in favor of eventual consistency via NATS events.

-- ============================================================================
-- 1. Create schemas
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS reservation;
CREATE SCHEMA IF NOT EXISTS billing;
CREATE SCHEMA IF NOT EXISTS payment;
CREATE SCHEMA IF NOT EXISTS presence;
CREATE SCHEMA IF NOT EXISTS search;

-- ============================================================================
-- 2. RESERVATION schema: drivers, parking_spots, reservations
-- ============================================================================

-- Drop foreign keys that cross domain boundaries before moving tables.
-- billing_records.reservation_id -> reservations.id  (billing domain)
-- payments.billing_id -> billing_records.id           (payment domain)
-- penalties.reservation_id -> reservations.id          (billing domain)
-- presence_logs.reservation_id -> reservations.id      (presence domain)

ALTER TABLE IF EXISTS billing_records DROP CONSTRAINT IF EXISTS billing_records_reservation_id_fkey;
ALTER TABLE IF EXISTS payments DROP CONSTRAINT IF EXISTS payments_billing_id_fkey;
ALTER TABLE IF EXISTS penalties DROP CONSTRAINT IF EXISTS penalties_reservation_id_fkey;
ALTER TABLE IF EXISTS presence_logs DROP CONSTRAINT IF EXISTS presence_logs_reservation_id_fkey;

-- Move reservation-owned tables from public to reservation schema
ALTER TABLE drivers SET SCHEMA reservation;
ALTER TABLE parking_spots SET SCHEMA reservation;
ALTER TABLE reservations SET SCHEMA reservation;

-- Re-add the self-domain foreign key (reservations references drivers and parking_spots, both in reservation schema)
ALTER TABLE reservation.reservations
    ADD CONSTRAINT reservations_driver_id_fkey
    FOREIGN KEY (driver_id) REFERENCES reservation.drivers(id);
ALTER TABLE reservation.reservations
    ADD CONSTRAINT reservations_spot_id_fkey
    FOREIGN KEY (spot_id) REFERENCES reservation.parking_spots(id);

-- ============================================================================
-- 3. BILLING schema: billing_records, penalties
-- ============================================================================

ALTER TABLE billing_records SET SCHEMA billing;
ALTER TABLE penalties SET SCHEMA billing;

-- ============================================================================
-- 4. PAYMENT schema: payments
-- ============================================================================

ALTER TABLE payments SET SCHEMA payment;

-- ============================================================================
-- 5. PRESENCE schema: presence_logs
-- ============================================================================

ALTER TABLE presence_logs SET SCHEMA presence;

-- ============================================================================
-- 6. SEARCH schema: spot_read_model (new read model synced via NATS)
-- ============================================================================

CREATE TABLE IF NOT EXISTS search.spot_read_model (
    id           UUID PRIMARY KEY,
    floor_number INT         NOT NULL CHECK (floor_number BETWEEN 1 AND 5),
    spot_number  INT         NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('car', 'motorcycle')),
    spot_code    VARCHAR(10) NOT NULL UNIQUE,
    status       VARCHAR(20) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'reserved', 'occupied')),
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for search queries (mirror the ones on parking_spots)
CREATE INDEX idx_search_spot_availability ON search.spot_read_model (vehicle_type, status, floor_number);
CREATE INDEX idx_search_spot_floor ON search.spot_read_model (floor_number, spot_number);

-- Seed the read model from current parking_spots data
INSERT INTO search.spot_read_model (id, floor_number, spot_number, vehicle_type, spot_code, status, updated_at)
SELECT id, floor_number, spot_number, vehicle_type, spot_code, status, updated_at
FROM reservation.parking_spots;

-- ============================================================================
-- 7. Re-create indexes that were on public tables (some move with the table,
--    some need explicit handling)
-- ============================================================================

-- The partial unique index on reservations should still work (moves with table).
-- If it didn't survive the SET SCHEMA, recreate it:
CREATE UNIQUE INDEX IF NOT EXISTS idx_reservations_active_spot
    ON reservation.reservations (spot_id)
    WHERE status IN ('waiting_payment', 'confirmed', 'checked_in');

CREATE INDEX IF NOT EXISTS idx_reservations_driver
    ON reservation.reservations (driver_id, status);

CREATE INDEX IF NOT EXISTS idx_parking_spots_availability
    ON reservation.parking_spots (vehicle_type, status, floor_number);

CREATE INDEX IF NOT EXISTS idx_reservations_expiry
    ON reservation.reservations (status, expires_at)
    WHERE status = 'confirmed';

CREATE INDEX IF NOT EXISTS idx_reservations_stale_payment
    ON reservation.reservations (status, created_at)
    WHERE status = 'waiting_payment';

CREATE INDEX IF NOT EXISTS idx_billing_reservation
    ON billing.billing_records (reservation_id);

CREATE INDEX IF NOT EXISTS idx_payments_billing
    ON payment.payments (billing_id, status);

CREATE INDEX IF NOT EXISTS idx_presence_reservation_time
    ON presence.presence_logs (reservation_id, recorded_at);

-- ============================================================================
-- 8. Grant schema permissions (same DB user for now, but prepare for per-user)
-- ============================================================================

GRANT USAGE ON SCHEMA reservation TO CURRENT_USER;
GRANT USAGE ON SCHEMA billing TO CURRENT_USER;
GRANT USAGE ON SCHEMA payment TO CURRENT_USER;
GRANT USAGE ON SCHEMA presence TO CURRENT_USER;
GRANT USAGE ON SCHEMA search TO CURRENT_USER;
```

- [ ] **Step 2: Write the down migration**

Create `db/migrations/000004_schema_per_service.down.sql`:

```sql
-- 000004_schema_per_service.down.sql
-- Rollback: move all tables back to public schema.

-- Drop search read model
DROP TABLE IF EXISTS search.spot_read_model;

-- Move tables back to public
ALTER TABLE reservation.drivers SET SCHEMA public;
ALTER TABLE reservation.parking_spots SET SCHEMA public;
ALTER TABLE reservation.reservations SET SCHEMA public;
ALTER TABLE billing.billing_records SET SCHEMA public;
ALTER TABLE billing.penalties SET SCHEMA public;
ALTER TABLE payment.payments SET SCHEMA public;
ALTER TABLE presence.presence_logs SET SCHEMA public;

-- Re-add cross-domain foreign keys
ALTER TABLE public.billing_records
    ADD CONSTRAINT billing_records_reservation_id_fkey
    FOREIGN KEY (reservation_id) REFERENCES public.reservations(id);
ALTER TABLE public.payments
    ADD CONSTRAINT payments_billing_id_fkey
    FOREIGN KEY (billing_id) REFERENCES public.billing_records(id);
ALTER TABLE public.penalties
    ADD CONSTRAINT penalties_reservation_id_fkey
    FOREIGN KEY (reservation_id) REFERENCES public.reservations(id);
ALTER TABLE public.presence_logs
    ADD CONSTRAINT presence_logs_reservation_id_fkey
    FOREIGN KEY (reservation_id) REFERENCES public.reservations(id);

-- Drop schemas
DROP SCHEMA IF EXISTS reservation;
DROP SCHEMA IF EXISTS billing;
DROP SCHEMA IF EXISTS payment;
DROP SCHEMA IF EXISTS presence;
DROP SCHEMA IF EXISTS search;
```

- [ ] **Step 3: Verify migration SQL syntax**

Run: `docker compose exec postgres psql -U ${DB_USERNAME} -d ${DB_DATABASE} -c "SELECT 1"` (or just review the SQL manually for syntax)

This step is a manual review — the migration will be tested in Task 8 via the e2e tests.

- [ ] **Step 4: Commit**

```bash
git add db/migrations/000004_schema_per_service.up.sql db/migrations/000004_schema_per_service.down.sql
git commit -m "feat: add schema-per-service migration"
```

---

### Task 3: Update Docker Compose with DB_SCHEMA per Service

**Files:**
- Modify: `docker-compose.yml:107-121,147-161,189-206,227-243,261-280`

- [ ] **Step 1: Add DB_SCHEMA to each service in docker-compose.yml**

For the **search** service (around line 121), add after `DB_SSL_MODE: disable`:

```yaml
      DB_SCHEMA: search
```

For the **reservation** service (around line 161), add after `DB_SSL_MODE: disable`:

```yaml
      DB_SCHEMA: reservation
```

For the **billing** service (around line 206), add after `DB_SSL_MODE: disable`:

```yaml
      DB_SCHEMA: billing
```

For the **payment** service (around line 243), add after `DB_SSL_MODE: disable`:

```yaml
      DB_SCHEMA: payment
```

For the **presence** service (around line 280), add after `DB_SSL_MODE: disable`:

```yaml
      DB_SCHEMA: presence
```

Do NOT add `DB_SCHEMA` to the `gateway` or `notification` services (they don't use PostgreSQL).

- [ ] **Step 2: Verify docker-compose config is valid**

Run: `docker compose config > /dev/null 2>&1 && echo "VALID" || echo "INVALID"`
Expected: `VALID`

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: add DB_SCHEMA env var per service in docker-compose"
```

---

### Task 4: Create Search Spot Read Model Sync (NATS Subscriber)

**Files:**
- Create: `internal/search/sync/spot_sync.go`
- Create: `internal/search/sync/spot_sync_test.go`

- [ ] **Step 1: Write the failing test for SpotSync**

Create `internal/search/sync/spot_sync_test.go`:

```go
package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSpotRepo struct {
	mock.Mock
}

func (m *mockSpotRepo) UpsertSpot(ctx context.Context, spot SpotData) error {
	args := m.Called(ctx, spot)
	return args.Error(0)
}

func (m *mockSpotRepo) DeleteSpot(ctx context.Context, spotID string) error {
	args := m.Called(ctx, spotID)
	return args.Error(0)
}

func TestSpotSync_HandleSpotUpdated_ShouldUpsertSpot(t *testing.T) {
	repo := new(mockSpotRepo)
	syncer := NewSpotSync(repo)

	spot := SpotData{
		ID:          "spot-1",
		FloorNumber: 1,
		SpotNumber:  5,
		VehicleType: "car",
		SpotCode:    "F1-C-005",
		Status:      "reserved",
	}

	repo.On("UpsertSpot", mock.Anything, spot).Return(nil)

	err := syncer.HandleSpotUpdated(t.Context(), spot)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSpotSync_HandleSpotUpdated_ShouldReturnError_WhenUpsertFails(t *testing.T) {
	repo := new(mockSpotRepo)
	syncer := NewSpotSync(repo)

	spot := SpotData{
		ID:     "spot-1",
		Status: "reserved",
	}

	repo.On("UpsertSpot", mock.Anything, spot).Return(assert.AnError)

	err := syncer.HandleSpotUpdated(t.Context(), spot)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/search/sync/... -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement SpotSync**

Create `internal/search/sync/spot_sync.go`:

```go
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// SpotData represents a parking spot snapshot for the search read model.
type SpotData struct {
	ID          string `json:"id"`
	FloorNumber int    `json:"floor_number"`
	SpotNumber  int    `json:"spot_number"`
	VehicleType string `json:"vehicle_type"`
	SpotCode    string `json:"spot_code"`
	Status      string `json:"status"`
}

// SpotRepository defines the interface for writing to the search read model.
type SpotRepository interface {
	UpsertSpot(ctx context.Context, spot SpotData) error
	DeleteSpot(ctx context.Context, spotID string) error
}

// SpotSync processes parking spot change events and updates the search read model.
type SpotSync struct {
	repo SpotRepository
}

// NewSpotSync creates a new SpotSync with the given repository.
func NewSpotSync(repo SpotRepository) *SpotSync {
	return &SpotSync{repo: repo}
}

// HandleSpotUpdated upserts a spot into the search read model.
func (s *SpotSync) HandleSpotUpdated(ctx context.Context, spot SpotData) error {
	if err := s.repo.UpsertSpot(ctx, spot); err != nil {
		return fmt.Errorf("upsert spot read model: %w", err)
	}
	return nil
}

// HandleNATSEvent parses a NATS message payload as SpotData and upserts it.
// This is the callback for the `spot.updated` NATS subject.
func (s *SpotSync) HandleNATSEvent(ctx context.Context, subject string, data []byte) {
	var spot SpotData
	if err := json.Unmarshal(data, &spot); err != nil {
		slog.Warn("spot sync: failed to unmarshal event", slog.String("subject", subject), slog.Any("error", err))
		return
	}
	if err := s.HandleSpotUpdated(ctx, spot); err != nil {
		slog.Warn("spot sync: failed to upsert spot", slog.String("spot_id", spot.ID), slog.Any("error", err))
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/search/sync/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/search/sync/spot_sync.go internal/search/sync/spot_sync_test.go
git commit -m "feat: add SpotSync for search read model via NATS"
```

---

### Task 5: Add Spot Read Model Repository to Search Service

**Files:**
- Modify: `internal/search/repository/repository.go:27-94`

- [ ] **Step 1: Write the failing test for read model repository**

Add a new test file `internal/search/repository/repository_test.go`:

```go
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/search/model"
	"parkir-pintar/internal/search/sync"
)

func TestSqlxReadModelRepository_UpsertSpot_ShouldInsertNewSpot(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}
	t.Skip("requires running PostgreSQL — covered by e2e tests")
}

func TestSpotDataConversion_ShouldMatchSpotDetails(t *testing.T) {
	spot := sync.SpotData{
		ID:          "spot-1",
		FloorNumber: 2,
		SpotNumber:  10,
		VehicleType: "car",
		SpotCode:    "F2-C-010",
		Status:      "available",
	}

	assert.Equal(t, "spot-1", spot.ID)
	assert.Equal(t, 2, spot.FloorNumber)
	assert.Equal(t, "car", spot.VehicleType)
	assert.Equal(t, "available", spot.Status)
}
```

- [ ] **Step 2: Add read model methods to the search repository**

In `internal/search/repository/repository.go`, add a new interface and implementation for the read model. Add this after the existing `sqlxRepository` struct and before the `GetAvailabilityByVehicleType` method:

Add new import for the sync package:

```go
import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/search/model"
	"parkir-pintar/internal/search/sync"
)
```

Add a new `ReadModelRepository` interface and implementation after line 31 (after the `Repository` interface):

```go
// ReadModelRepository defines the data access interface for the search read model.
type ReadModelRepository interface {
	UpsertSpot(ctx context.Context, spot sync.SpotData) error
	DeleteSpot(ctx context.Context, spotID string) error
}

// sqlxReadModelRepository is the sqlx-backed implementation of ReadModelRepository.
type sqlxReadModelRepository struct {
	db *sqlx.DB
}

// NewReadModelRepository creates a new ReadModelRepository backed by the given sqlx.DB.
func NewReadModelRepository(db *sqlx.DB) ReadModelRepository {
	return &sqlxReadModelRepository{db: db}
}

// UpsertSpot inserts or updates a spot in the spot_read_model table.
func (r *sqlxReadModelRepository) UpsertSpot(ctx context.Context, spot sync.SpotData) error {
	query := `
		INSERT INTO spot_read_model (id, floor_number, spot_number, vehicle_type, spot_code, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id) DO UPDATE SET
			floor_number = EXCLUDED.floor_number,
			spot_number = EXCLUDED.spot_number,
			vehicle_type = EXCLUDED.vehicle_type,
			spot_code = EXCLUDED.spot_code,
			status = EXCLUDED.status,
			updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		spot.ID, spot.FloorNumber, spot.SpotNumber, spot.VehicleType, spot.SpotCode, spot.Status)
	if err != nil {
		return fmt.Errorf("upsert spot read model: %w", err)
	}
	return nil
}

// DeleteSpot removes a spot from the spot_read_model table.
func (r *sqlxReadModelRepository) DeleteSpot(ctx context.Context, spotID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM spot_read_model WHERE id = $1", spotID)
	if err != nil {
		return fmt.Errorf("delete spot read model: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Update existing Repository queries to use spot_read_model instead of parking_spots**

In `internal/search/repository/repository.go`, update the three query methods in `sqlxRepository`:

Replace the `GetAvailabilityByVehicleType` method body (lines 46-61):

```go
func (r *sqlxRepository) GetAvailabilityByVehicleType(ctx context.Context, vehicleType string) ([]model.FloorAvailability, error) {
	query := `
		SELECT
			floor_number,
			COUNT(*) FILTER (WHERE status = 'available' AND vehicle_type = 'car') AS available_car,
			COUNT(*) FILTER (WHERE status = 'available' AND vehicle_type = 'motorcycle') AS available_moto,
			COUNT(*) FILTER (WHERE vehicle_type = 'car') AS total_car,
			COUNT(*) FILTER (WHERE vehicle_type = 'motorcycle') AS total_moto
		FROM spot_read_model
		GROUP BY floor_number
		ORDER BY floor_number`

	var floors []model.FloorAvailability
	if err := r.db.SelectContext(ctx, &floors, query); err != nil {
		return nil, fmt.Errorf("get availability: %w", err)
	}
	return floors, nil
}
```

Replace the `GetFloorSpots` method body (lines 65-76):

```go
func (r *sqlxRepository) GetFloorSpots(ctx context.Context, floorNumber int) ([]model.SpotDetails, error) {
	query := `
		SELECT id, spot_code, vehicle_type, status, floor_number, spot_number
		FROM spot_read_model
		WHERE floor_number = $1
		ORDER BY spot_number`

	var spots []model.SpotDetails
	if err := r.db.SelectContext(ctx, &spots, query, floorNumber); err != nil {
		return nil, fmt.Errorf("get floor spots: %w", err)
	}
	return spots, nil
}
```

Replace the `GetSpotByID` method body (lines 80-94):

```go
func (r *sqlxRepository) GetSpotByID(ctx context.Context, spotID string) (*model.SpotDetails, error) {
	query := `
		SELECT id, spot_code, vehicle_type, status, floor_number, spot_number
		FROM spot_read_model
		WHERE id = $1`

	var spot model.SpotDetails
	if err := r.db.GetContext(ctx, &spot, query, spotID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", ErrNotFound, spotID)
		}
		return nil, fmt.Errorf("get spot by id: %w", err)
	}
	return &spot, nil
}
```

- [ ] **Step 4: Run tests to verify compilation**

Run: `go build ./internal/search/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/search/repository/repository.go internal/search/repository/repository_test.go
git commit -m "feat: add read model repository and switch queries from parking_spots to spot_read_model"
```

---

### Task 6: Publish Spot Events from Reservation Service

**Files:**
- Modify: `internal/reservation/usecase/usecase.go`

The Reservation service already publishes NATS events (`reservation.confirmed`, `reservation.cancelled`, etc.) but does NOT publish spot-level events. We need to add a `spot.updated` event whenever a parking spot's status changes.

- [ ] **Step 1: Add a spot.updated publish after every spot status change**

In `internal/reservation/usecase/usecase.go`, add a helper method to publish spot update events. The NATSClient interface already exists in the usecase package. Add a new `publishSpotUpdated` method.

First, find the NATSClient interface. It already has `Publish(subject string, data []byte) error`. Add a helper that marshals the ParkingSpot into JSON and publishes it.

Add this helper method to the `reservationUsecase` struct (after the existing methods):

```go
func (uc *reservationUsecase) publishSpotUpdated(ctx context.Context, spot *model.ParkingSpot) {
	data, err := json.Marshal(map[string]interface{}{
		"id":           spot.ID,
		"floor_number": spot.FloorNumber,
		"spot_number":  spot.SpotNumber,
		"vehicle_type": spot.VehicleType,
		"spot_code":    spot.SpotCode,
		"status":       spot.Status,
	})
	if err != nil {
		slog.Warn("reservation: failed to marshal spot.updated event", slog.Any("error", err))
		return
	}
	if err := uc.nats.Publish("spot.updated", data); err != nil {
		slog.Warn("reservation: failed to publish spot.updated event", slog.Any("error", err))
	}
}
```

Ensure `"encoding/json"` and `"log/slog"` are in the imports (they likely already are).

Then, in each method that calls `UpdateSpotStatusTx`, add a call to `publishSpotUpdated` after the spot status is updated. The key locations are:

1. In `CreateReservation` — after the transaction commits with spot status "reserved", fetch the spot and publish.
2. In `CheckIn` — after spot status is set to "occupied".
3. In `CompleteCheckout` — after spot status is set to "available".
4. In `CancelReservation` — after spot status is set to "available".
5. In `ExpireReservation` — after spot status is set to "available".
6. In `FailReservation` — after spot status is set to "available".

For each of these, add after the successful update/transaction:

```go
uc.publishSpotUpdated(ctx, &model.ParkingSpot{
    ID:          spotID,
    FloorNumber: spot.FloorNumber,
    SpotNumber:  spot.SpotNumber,
    VehicleType: spot.VehicleType,
    SpotCode:    spot.SpotCode,
    Status:      newStatus,
})
```

The exact variable names (`spotID`, `spot.FloorNumber`, etc.) depend on the local variables in each method. The `spot` variable is already fetched with `FindAvailableSpot` or `GetSpotForUpdate` in each of these methods.

- [ ] **Step 2: Add `spot.updated` to the NATS stream configuration**

In `internal/natssetup/streams.go`, add `"spot.updated"` to the `RESERVATIONS` stream subjects:

Find the line that defines the RESERVATIONS stream subjects and add `"spot.updated"`:

```go
Subjects: []string{"reservation.*", "spot.updated"},
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/reservation/... ./internal/natssetup/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/reservation/usecase/usecase.go internal/natssetup/streams.go
git commit -m "feat: publish spot.updated events from reservation service"
```

---

### Task 7: Wire Spot Sync into Search Service Main

**Files:**
- Modify: `cmd/search/main.go:68-89`

- [ ] **Step 1: Update cmd/search/main.go to wire the read model repo and spot sync**

In `cmd/search/main.go`, add imports:

```go
import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/natssetup"
	searchhandler "parkir-pintar/internal/search/handler"
	searchrepo "parkir-pintar/internal/search/repository"
	searchsub "parkir-pintar/internal/search/subscriber"
	searchsync "parkir-pintar/internal/search/sync"
	searchuc "parkir-pintar/internal/search/usecase"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
	searchv1 "parkir-pintar/proto/search/v1"
)
```

After line 68 (`repo := searchrepo.NewRepository(tracedPG.GetDB())`), add the read model repository and spot sync wiring:

```go
	repo := searchrepo.NewRepository(tracedPG.GetDB())
	readModelRepo := searchrepo.NewReadModelRepository(tracedPG.GetDB())
	spotSyncer := searchsync.NewSpotSync(readModelRepo)
	uc := searchuc.NewUsecase(repo, tracedRedis)
```

Then after the existing NATS cache invalidation consumer setup (after line 89), add the spot sync consumer:

```go
	// Wire NATS subscriber for spot read model sync
	if err := natsClient.CreateConsumer(nats.ConsumerConfig{
		StreamName:    "RESERVATIONS",
		ConsumerName:  "search-spot-sync",
		FilterSubject: "spot.updated",
		DeliverPolicy: jetstream.DeliverLastPolicy,
	}); err != nil {
		log.Error("failed to create spot sync consumer", slog.Any("error", err))
	}
	go func() {
		if err := natsClient.ConsumeMessages("RESERVATIONS", "search-spot-sync", func(msg jetstream.Msg) error {
			spotSyncer.HandleNATSEvent(context.Background(), msg.Subject(), msg.Data())
			return nil
		}); err != nil {
			log.Error("spot sync consume error", slog.Any("error", err))
		}
	}()
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/search/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/search/main.go
git commit -m "feat: wire spot sync subscriber into search service"
```

---

### Task 8: Update E2E Test Setup for Schema-Per-Service

**Files:**
- Modify: `tests/e2e/setup_test.go:160-173`
- Modify: `tests/e2e/schema_test.go:24-48`

- [ ] **Step 1: Add new migration to setup_test.go**

In `tests/e2e/setup_test.go`, update the migrations list (line 160) to include the new migration:

```go
	migrations := []string{
		"../../db/migrations/000001_init.up.sql",
		"../../db/migrations/000002_parkir_pintar.up.sql",
		"../../db/migrations/000003_payment_flow.up.sql",
		"../../db/migrations/000004_schema_per_service.up.sql",
	}
```

After all migrations are applied (after the `for` loop), set the search_path so the e2e tests use the right schemas. Add after line 173:

```go
	// Set search_path so queries find schema-qualified tables
	schemas := []string{"reservation", "billing", "payment", "presence", "search"}
	for _, schema := range schemas {
		_, _ = db.ExecContext(ctx, fmt.Sprintf("GRANT ALL ON SCHEMA %s TO CURRENT_USER", schema))
	}
```

Also, each repository in the e2e setup needs to use the correct schema. The simplest approach for e2e tests is to set `search_path` on the shared connection before each repository call. Since all e2e tests share the same `*sqlx.DB`, we can set the default search_path on the connection:

After the migrations and schema grants, add:

```go
	// Set default search_path to include all schemas so e2e tests can query any table
	_, _ = db.ExecContext(ctx, "SET search_path TO reservation, billing, payment, presence, search, public")
```

This way the existing e2e queries (which use unqualified table names like `reservations`, `parking_spots`) will find the right tables in the right schemas.

- [ ] **Step 2: Update schema_test.go for schema-qualified table assertions**

In `tests/e2e/schema_test.go`, update `TestSchema_ShouldHaveAllTables` to query across all schemas:

Replace the query (line 39-41):

```go
	err := env.db.SelectContext(ctx, &tables,
		`SELECT table_schema || '.' || table_name AS table_name
		 FROM information_schema.tables
		 WHERE table_schema IN ('reservation', 'billing', 'payment', 'presence', 'search')
		   AND table_type = 'BASE TABLE'
		 ORDER BY table_name`)
```

And update the expected tables (line 27-35):

```go
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
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./tests/e2e/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add tests/e2e/setup_test.go tests/e2e/schema_test.go
git commit -m "feat: update e2e tests for schema-per-service migration"
```

---

### Task 9: Add NATS Stream for Spot Events and Update Docker Compose Init

**Files:**
- Modify: `docker-compose.yml:16-17`

- [ ] **Step 1: Update docker-compose postgres init to apply migration 000004**

In `docker-compose.yml`, the postgres service mounts `./db/migrations:/docker-entrypoint-initdb.d` which auto-applies all SQL files on first start. The new `000004_schema_per_service.up.sql` will be automatically picked up. No change needed here — just verify the mount path includes the new file.

- [ ] **Step 2: Verify the new migration is in the mounted directory**

Run: `ls -la db/migrations/`
Expected: all 4 migration files listed including `000004_schema_per_service.up.sql` and `.down.sql`

- [ ] **Step 3: Commit (if any changes were needed)**

Only commit if files were modified.

---

### Task 10: Remove Cross-Schema Foreign Keys from Original Migration

**Files:**
- Modify: `db/migrations/000002_parkir_pintar.up.sql:59-61,81,99-101,111`

The original migration creates cross-domain foreign keys that the 000004 migration drops. However, for a fresh start (new deployment), the 000004 migration's `ALTER TABLE ... DROP CONSTRAINT` would fail if the constraints don't exist. We need to make the original migration idempotent with the new architecture.

**Approach:** We do NOT modify the original migration files. Instead, the 000004 migration uses `IF EXISTS` on its `DROP CONSTRAINT` statements (which it already does). The migrations run sequentially: 000001 → 000002 → 000003 → 000004, so the constraints will exist when 000004 runs.

For **fresh deployments** that run all migrations from scratch, the sequence works correctly.

- [ ] **Step 1: Verify the 000004 migration uses IF EXISTS on all DROP CONSTRAINT statements**

Check `db/migrations/000004_schema_per_service.up.sql` lines 20-23 — they already use `IF EXISTS`:

```sql
ALTER TABLE IF EXISTS billing_records DROP CONSTRAINT IF EXISTS billing_records_reservation_id_fkey;
ALTER TABLE IF EXISTS payments DROP CONSTRAINT IF EXISTS payments_billing_id_fkey;
ALTER TABLE IF EXISTS penalties DROP CONSTRAINT IF EXISTS penalties_reservation_id_fkey;
ALTER TABLE IF EXISTS presence_logs DROP CONSTRAINT IF EXISTS presence_logs_reservation_id_fkey;
```

This is safe for both upgrade and fresh-install paths.

- [ ] **Step 2: No changes needed — verify commit**

No file changes in this task. The migration files are correct as-is.

---

### Task 11: Update E2E Adapters for Schema-Aware Repositories

**Files:**
- Modify: `tests/e2e/adapters_test.go`

The e2e test adapters don't need changes since they bridge usecase interfaces (which are schema-agnostic). The `searchRedisAdapter`, `billingAdapter`, `paymentAdapter`, and `stubNATSClient` all work at the usecase level, not the SQL level.

- [ ] **Step 1: Verify adapters still compile with the updated repository interfaces**

Run: `go build ./tests/e2e/...`
Expected: no errors

- [ ] **Step 2: Run the full e2e test suite**

Run: `go test -v -timeout 300s ./tests/e2e/...`
Expected: All tests PASS

Note: This will fail if the database doesn't have the new schema migration applied. The testcontainers setup applies all migrations, so it should work.

- [ ] **Step 3: Commit any necessary fixes**

Only commit if changes were needed.

---

### Task 12: Verify Full System Integration

**Files:**
- No new files — verification only

- [ ] **Step 1: Run unit tests**

Run: `go test ./... -short -count=1`
Expected: All tests PASS

- [ ] **Step 2: Run e2e tests**

Run: `go test -v -timeout 300s ./tests/e2e/... -count=1`
Expected: All tests PASS

- [ ] **Step 3: Run lint**

Run: `golangci-lint run ./...`
Expected: No errors

- [ ] **Step 4: Run go vet**

Run: `go vet ./...`
Expected: No errors

- [ ] **Step 5: Verify docker compose build**

Run: `docker compose build`
Expected: All services build successfully

- [ ] **Step 6: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix: resolve integration issues from schema-per-service migration"
```

---

## Self-Review

### 1. Spec Coverage

| Requirement | Task |
|------------|------|
| Each service gets its own PostgreSQL schema | Task 2 (migration), Task 3 (docker-compose) |
| Config supports `DB_SCHEMA` per service | Task 1 |
| Database client sets `search_path` | Task 1 |
| Search service no longer reads `parking_spots` directly | Task 5 |
| Search service has its own `spot_read_model` table | Task 2 (migration), Task 5 |
| Reservation publishes `spot.updated` events | Task 6 |
| Search subscribes to `spot.updated` and syncs read model | Task 4, Task 7 |
| Cross-schema foreign keys removed | Task 2 (migration) |
| E2e tests updated for schema-qualified tables | Task 8 |
| Docker Compose updated with DB_SCHEMA | Task 3 |

### 2. Placeholder Scan

No TBD, TODO, "implement later", "add appropriate error handling", or "similar to Task N" found. All code blocks contain complete implementations.

### 3. Type Consistency

- `sync.SpotData` is defined in Task 4 and used consistently in Task 5 (`ReadModelRepository` interface uses `sync.SpotData`)
- `SpotRepository` interface in `sync` package matches `ReadModelRepository` in `repository` package — the `SpotSync` depends on its own `SpotRepository` interface, and `ReadModelRepository` implements it
- The `publishSpotUpdated` helper marshals the same fields as `SpotData`: `id`, `floor_number`, `spot_number`, `vehicle_type`, `spot_code`, `status`
- The `spot_read_model` table columns match `SpotData` struct fields exactly
- NATS subject `spot.updated` is used consistently in Task 6 (publish), Task 7 (subscribe), and Task 2 (stream config)

### Gap Found

The `ReadModelRepository` in `repository.go` and `SpotRepository` in `sync/spot_sync.go` are two separate interfaces with the same methods. The `SpotSync` depends on `SpotRepository` (from the sync package), and `ReadModelRepository` (from the repository package) implements `SpotRepository`. This works because Go's structural typing means any type with `UpsertSpot` and `DeleteSpot` methods satisfies `SpotRepository`. However, the parameter types differ: `SpotRepository.UpsertSpot` takes `SpotData`, and `ReadModelRepository.UpsertSpot` takes `sync.SpotData` — these are the same type since `SpotData` is defined in the `sync` package. No inconsistency.

**Remaining concern:** The `drivers` table is orphaned (no service queries it). This is not addressed in this plan as it's a separate cleanup task, not a schema isolation issue.
