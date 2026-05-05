-- 000002_parkir_pintar.up.sql
-- ParkirPintar schema: drivers, parking_spots, reservations, billing_records,
-- payments, penalties, presence_logs + indexes + seed data (400 spots).

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- drivers
-- ============================================================================
CREATE TABLE IF NOT EXISTS drivers (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name          VARCHAR(255) NOT NULL,
    phone         VARCHAR(20)  NOT NULL UNIQUE,
    email         VARCHAR(255) UNIQUE,
    vehicle_type  VARCHAR(20)  NOT NULL CHECK (vehicle_type IN ('car', 'motorcycle')),
    vehicle_plate VARCHAR(20)  NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- parking_spots
-- ============================================================================
CREATE TABLE IF NOT EXISTS parking_spots (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    floor_number INT         NOT NULL CHECK (floor_number BETWEEN 1 AND 5),
    spot_number  INT         NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('car', 'motorcycle')),
    spot_code    VARCHAR(10) NOT NULL UNIQUE,
    status       VARCHAR(20) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'reserved', 'occupied')),
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- reservations
-- ============================================================================
CREATE TABLE IF NOT EXISTS reservations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id       UUID        NOT NULL REFERENCES drivers(id),
    spot_id         UUID        NOT NULL REFERENCES parking_spots(id),
    vehicle_type    VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('car', 'motorcycle')),
    assignment_mode VARCHAR(20) NOT NULL CHECK (assignment_mode IN ('system_assigned', 'user_selected')),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'confirmed', 'checked_in', 'checked_out', 'expired', 'cancelled')),
    idempotency_key VARCHAR(64) NOT NULL UNIQUE,
    confirmed_at    TIMESTAMP WITH TIME ZONE,
    expires_at      TIMESTAMP WITH TIME ZONE,
    checked_in_at   TIMESTAMP WITH TIME ZONE,
    checked_out_at  TIMESTAMP WITH TIME ZONE,
    cancelled_at    TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- billing_records
-- ============================================================================
CREATE TABLE IF NOT EXISTS billing_records (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reservation_id   UUID    NOT NULL UNIQUE REFERENCES reservations(id),
    booking_fee      BIGINT  NOT NULL DEFAULT 0,
    parking_fee      BIGINT  NOT NULL DEFAULT 0,
    overnight_fee    BIGINT  NOT NULL DEFAULT 0,
    cancellation_fee BIGINT  NOT NULL DEFAULT 0,
    penalty_amount   BIGINT  NOT NULL DEFAULT 0,
    total_amount     BIGINT  NOT NULL DEFAULT 0,
    duration_minutes INT     NOT NULL DEFAULT 0,
    billed_hours     INT     NOT NULL DEFAULT 0,
    is_overnight     BOOLEAN NOT NULL DEFAULT FALSE,
    idempotency_key  VARCHAR(64) NOT NULL UNIQUE,
    status           VARCHAR(20) NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending', 'calculated', 'invoiced', 'paid')),
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- payments
-- ============================================================================
CREATE TABLE IF NOT EXISTS payments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    billing_id      UUID        NOT NULL REFERENCES billing_records(id),
    amount          BIGINT      NOT NULL,
    payment_method  VARCHAR(20) NOT NULL CHECK (payment_method IN ('qris', 'credit_card', 'debit', 'ewallet')),
    payment_gateway VARCHAR(50) NOT NULL,
    transaction_ref VARCHAR(100),
    idempotency_key VARCHAR(64) NOT NULL UNIQUE,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'success', 'failed', 'refunded')),
    paid_at         TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- penalties
-- ============================================================================
CREATE TABLE IF NOT EXISTS penalties (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reservation_id UUID        NOT NULL REFERENCES reservations(id),
    penalty_type   VARCHAR(20) NOT NULL CHECK (penalty_type IN ('wrong_spot', 'cancellation')),
    amount         BIGINT      NOT NULL,
    description    TEXT,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- presence_logs
-- ============================================================================
CREATE TABLE IF NOT EXISTS presence_logs (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reservation_id UUID           NOT NULL REFERENCES reservations(id),
    latitude       DECIMAL(10,7)  NOT NULL,
    longitude      DECIMAL(10,7)  NOT NULL,
    accuracy       FLOAT,
    recorded_at    TIMESTAMP WITH TIME ZONE NOT NULL
);

-- ============================================================================
-- Indexes
-- ============================================================================

-- Prevent double-booking: unique active reservation per spot
CREATE UNIQUE INDEX idx_reservations_active_spot
    ON reservations (spot_id)
    WHERE status IN ('confirmed', 'checked_in');

-- Fast lookup by driver
CREATE INDEX idx_reservations_driver ON reservations (driver_id, status);

-- Fast availability queries
CREATE INDEX idx_parking_spots_availability ON parking_spots (vehicle_type, status, floor_number);

-- Reservation expiry job
CREATE INDEX idx_reservations_expiry ON reservations (status, expires_at)
    WHERE status = 'confirmed';

-- Billing lookup
CREATE INDEX idx_billing_reservation ON billing_records (reservation_id);

-- Payment lookup
CREATE INDEX idx_payments_billing ON payments (billing_id, status);

-- Presence time-series
CREATE INDEX idx_presence_reservation_time ON presence_logs (reservation_id, recorded_at);

-- ============================================================================
-- Seed data: 400 parking spots (5 floors × 30 car + 50 motorcycle)
-- ============================================================================
DO $$
DECLARE
    f INT;
    s INT;
BEGIN
    FOR f IN 1..5 LOOP
        -- Car spots: 30 per floor
        FOR s IN 1..30 LOOP
            INSERT INTO parking_spots (id, floor_number, spot_number, vehicle_type, spot_code, status)
            VALUES (
                uuid_generate_v4(),
                f,
                s,
                'car',
                'F' || f || '-C-' || LPAD(s::TEXT, 3, '0'),
                'available'
            );
        END LOOP;

        -- Motorcycle spots: 50 per floor
        FOR s IN 1..50 LOOP
            INSERT INTO parking_spots (id, floor_number, spot_number, vehicle_type, spot_code, status)
            VALUES (
                uuid_generate_v4(),
                f,
                s,
                'motorcycle',
                'F' || f || '-M-' || LPAD(s::TEXT, 3, '0'),
                'available'
            );
        END LOOP;
    END LOOP;
END $$;
