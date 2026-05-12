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
-- 2. Drop cross-domain foreign keys before moving tables
-- ============================================================================

ALTER TABLE IF EXISTS billing_records DROP CONSTRAINT IF EXISTS billing_records_reservation_id_fkey;
ALTER TABLE IF EXISTS payments DROP CONSTRAINT IF EXISTS payments_billing_id_fkey;
ALTER TABLE IF EXISTS penalties DROP CONSTRAINT IF EXISTS penalties_reservation_id_fkey;
ALTER TABLE IF EXISTS presence_logs DROP CONSTRAINT IF EXISTS presence_logs_reservation_id_fkey;

-- ============================================================================
-- 3. RESERVATION schema: drivers, parking_spots, reservations
-- ============================================================================

ALTER TABLE drivers SET SCHEMA reservation;
ALTER TABLE parking_spots SET SCHEMA reservation;
ALTER TABLE reservations SET SCHEMA reservation;

-- Re-add self-domain foreign keys (both tables now in reservation schema)
-- Drop first if they survived the SET SCHEMA operation, then re-add
ALTER TABLE reservation.reservations DROP CONSTRAINT IF EXISTS reservations_driver_id_fkey;
ALTER TABLE reservation.reservations DROP CONSTRAINT IF EXISTS reservations_spot_id_fkey;
ALTER TABLE reservation.reservations
    ADD CONSTRAINT reservations_driver_id_fkey
    FOREIGN KEY (driver_id) REFERENCES reservation.drivers(id);
ALTER TABLE reservation.reservations
    ADD CONSTRAINT reservations_spot_id_fkey
    FOREIGN KEY (spot_id) REFERENCES reservation.parking_spots(id);

-- ============================================================================
-- 4. BILLING schema: billing_records, penalties
-- ============================================================================

ALTER TABLE billing_records SET SCHEMA billing;
ALTER TABLE penalties SET SCHEMA billing;

-- ============================================================================
-- 5. PAYMENT schema: payments
-- ============================================================================

ALTER TABLE payments SET SCHEMA payment;

-- ============================================================================
-- 6. PRESENCE schema: presence_logs
-- ============================================================================

ALTER TABLE presence_logs SET SCHEMA presence;

-- ============================================================================
-- 7. SEARCH schema: spot_read_model (read model synced via NATS)
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

CREATE INDEX idx_search_spot_availability ON search.spot_read_model (vehicle_type, status, floor_number);
CREATE INDEX idx_search_spot_floor ON search.spot_read_model (floor_number, spot_number);

-- Seed the read model from current parking_spots data
INSERT INTO search.spot_read_model (id, floor_number, spot_number, vehicle_type, spot_code, status, updated_at)
SELECT id, floor_number, spot_number, vehicle_type, spot_code, status, updated_at
FROM reservation.parking_spots;

-- ============================================================================
-- 8. Ensure indexes exist on schema-qualified tables
-- ============================================================================

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
-- 9. Grant schema permissions
-- ============================================================================

GRANT USAGE ON SCHEMA reservation TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA reservation TO CURRENT_USER;
GRANT USAGE ON SCHEMA billing TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA billing TO CURRENT_USER;
GRANT USAGE ON SCHEMA payment TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA payment TO CURRENT_USER;
GRANT USAGE ON SCHEMA presence TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA presence TO CURRENT_USER;
GRANT USAGE ON SCHEMA search TO CURRENT_USER;
GRANT ALL ON ALL TABLES IN SCHEMA search TO CURRENT_USER;
