-- Parkir Pintar — Full Schema (squashed)
-- Schemas: reservation, billing, payment, search, presence, analytics

BEGIN;

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;

-- Schemas
CREATE SCHEMA IF NOT EXISTS reservation;
CREATE SCHEMA IF NOT EXISTS billing;
CREATE SCHEMA IF NOT EXISTS payment;
CREATE SCHEMA IF NOT EXISTS search;
CREATE SCHEMA IF NOT EXISTS presence;
CREATE SCHEMA IF NOT EXISTS analytics;

-- =============================================================================
-- RESERVATION SCHEMA
-- =============================================================================

CREATE TABLE reservation.drivers (
    id TEXT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL UNIQUE,
    email VARCHAR(255) UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reservation.parking_spots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    floor_number INTEGER NOT NULL CHECK (floor_number >= 1 AND floor_number <= 5),
    spot_number INTEGER NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('car', 'motorcycle')),
    spot_code VARCHAR(10) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'reserved', 'occupied')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reservation.reservations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id TEXT NOT NULL REFERENCES reservation.drivers(id),
    spot_id UUID NOT NULL REFERENCES reservation.parking_spots(id),
    vehicle_type VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('car', 'motorcycle')),
    assignment_mode VARCHAR(20) NOT NULL CHECK (assignment_mode IN ('system_assigned', 'user_selected')),
    status VARCHAR(20) NOT NULL DEFAULT 'waiting_payment'
        CHECK (status IN ('waiting_payment', 'confirmed', 'checked_in', 'checked_out', 'completed', 'cancelled', 'expired', 'failed')),
    idempotency_key VARCHAR(64) NOT NULL UNIQUE,
    confirmed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    checked_in_at TIMESTAMPTZ,
    checked_out_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reservations_driver ON reservation.reservations (driver_id, status);
CREATE INDEX idx_reservations_expiry ON reservation.reservations (status, expires_at) WHERE status = 'confirmed';
CREATE INDEX idx_reservations_stale_payment ON reservation.reservations (status, created_at) WHERE status = 'waiting_payment';
CREATE UNIQUE INDEX idx_reservations_active_spot ON reservation.reservations (spot_id)
    WHERE status IN ('waiting_payment', 'confirmed', 'checked_in');
CREATE UNIQUE INDEX idx_reservations_one_active_per_driver ON reservation.reservations (driver_id)
    WHERE status IN ('waiting_payment', 'confirmed', 'checked_in');
CREATE INDEX idx_parking_spots_availability ON reservation.parking_spots (vehicle_type, status, floor_number);

CREATE TABLE reservation.reservation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id UUID NOT NULL,
    driver_id TEXT NOT NULL,
    spot_id UUID NOT NULL,
    vehicle_type TEXT NOT NULL,
    status TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reservation_events_reservation_id ON reservation.reservation_events (reservation_id);
CREATE INDEX idx_reservation_events_status ON reservation.reservation_events (status);
CREATE INDEX idx_reservation_events_timestamp ON reservation.reservation_events (timestamp);

-- =============================================================================
-- BILLING SCHEMA
-- =============================================================================

CREATE TABLE billing.billing_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reservation_id UUID NOT NULL UNIQUE,
    booking_fee BIGINT NOT NULL DEFAULT 0,
    parking_fee BIGINT NOT NULL DEFAULT 0,
    overnight_fee BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,
    duration_minutes INTEGER NOT NULL DEFAULT 0,
    billed_hours INTEGER NOT NULL DEFAULT 0,
    is_overnight BOOLEAN NOT NULL DEFAULT false,
    idempotency_key VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'calculated', 'invoiced', 'paid')),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_billing_reservation ON billing.billing_records (reservation_id);

-- =============================================================================
-- PAYMENT SCHEMA
-- =============================================================================

CREATE TABLE payment.payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    billing_id UUID NOT NULL,
    amount BIGINT NOT NULL,
    payment_method VARCHAR(20) NOT NULL CHECK (payment_method IN ('qris', 'credit_card', 'debit', 'ewallet')),
    payment_gateway VARCHAR(50) NOT NULL,
    transaction_ref VARCHAR(100),
    idempotency_key VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed', 'refunded')),
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_billing ON payment.payments (billing_id, status);

-- =============================================================================
-- SEARCH SCHEMA (read model)
-- =============================================================================

CREATE TABLE search.spot_read_model (
    id UUID PRIMARY KEY,
    floor_number INTEGER NOT NULL CHECK (floor_number >= 1 AND floor_number <= 5),
    spot_number INTEGER NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('car', 'motorcycle')),
    spot_code VARCHAR(10) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'reserved', 'occupied')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_search_spot_availability ON search.spot_read_model (vehicle_type, status, floor_number);
CREATE INDEX idx_search_spot_floor ON search.spot_read_model (floor_number, spot_number);

-- =============================================================================
-- PRESENCE SCHEMA
-- =============================================================================

CREATE TABLE presence.presence_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reservation_id UUID NOT NULL,
    latitude NUMERIC(10,7) NOT NULL,
    longitude NUMERIC(10,7) NOT NULL,
    accuracy DOUBLE PRECISION,
    recorded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_presence_reservation_time ON presence.presence_logs (reservation_id, recorded_at);

-- =============================================================================
-- ANALYTICS SCHEMA
-- =============================================================================

CREATE TABLE analytics.reservation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id UUID NOT NULL,
    driver_id TEXT NOT NULL,
    spot_id UUID NOT NULL,
    vehicle_type TEXT NOT NULL,
    status TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_analytics_events_reservation_id ON analytics.reservation_events (reservation_id);
CREATE INDEX idx_analytics_events_status ON analytics.reservation_events (status);
CREATE INDEX idx_analytics_events_timestamp ON analytics.reservation_events (timestamp);
CREATE INDEX idx_analytics_events_created_at ON analytics.reservation_events (timestamp, status);

CREATE TABLE analytics.spot_snapshot (
    id UUID PRIMARY KEY,
    floor_number INTEGER NOT NULL,
    spot_number INTEGER NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL,
    spot_code VARCHAR(10) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_analytics_spot_snapshot_status ON analytics.spot_snapshot (status);
CREATE INDEX idx_analytics_spot_snapshot_vehicle ON analytics.spot_snapshot (vehicle_type);

-- Seed: 400 parking spots (5 floors × 30 car + 50 motorcycle per floor)
-- spot_code format: F1-C-001 (car floor 1 spot 1), F3-M-025 (motorcycle floor 3 spot 25)
INSERT INTO reservation.parking_spots (floor_number, spot_number, vehicle_type, spot_code)
SELECT
    floor,
    spot,
    'car',
    'F' || floor || '-C-' || LPAD(spot::text, 3, '0')
FROM generate_series(1, 5) AS floor,
     generate_series(1, 30) AS spot;

INSERT INTO reservation.parking_spots (floor_number, spot_number, vehicle_type, spot_code)
SELECT
    floor,
    spot,
    'motorcycle',
    'F' || floor || '-M-' || LPAD(spot::text, 3, '0')
FROM generate_series(1, 5) AS floor,
     generate_series(1, 50) AS spot;

COMMIT;
