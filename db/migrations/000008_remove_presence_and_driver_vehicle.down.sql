-- 000008_remove_presence_and_driver_vehicle.down.sql
-- Reverse: re-add presence schema/table, outbox_events, and driver vehicle columns.

-- 1. Re-add vehicle columns to drivers
ALTER TABLE reservation.drivers
    ADD COLUMN IF NOT EXISTS vehicle_type VARCHAR(20) NOT NULL DEFAULT 'car' CHECK (vehicle_type IN ('car', 'motorcycle')),
    ADD COLUMN IF NOT EXISTS vehicle_plate VARCHAR(20) NOT NULL DEFAULT '';

-- 2. Re-create presence schema and table
CREATE SCHEMA IF NOT EXISTS presence;

CREATE TABLE IF NOT EXISTS presence.presence_logs (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reservation_id TEXT NOT NULL,
    latitude       DOUBLE PRECISION NOT NULL,
    longitude      DOUBLE PRECISION NOT NULL,
    recorded_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_presence_reservation_time
    ON presence.presence_logs (reservation_id, recorded_at);
