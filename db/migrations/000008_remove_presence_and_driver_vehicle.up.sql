-- 000008_remove_presence_and_driver_vehicle.up.sql
-- Remove presence_logs table (service removed), outbox_events (if exists),
-- and vehicle_type/vehicle_plate from drivers (vehicle type is per-booking now).

-- 1. Drop presence schema and table
DROP TABLE IF EXISTS presence.presence_logs CASCADE;
DROP SCHEMA IF EXISTS presence CASCADE;

-- 2. Drop outbox_events if it exists
DROP TABLE IF EXISTS reservation.outbox_events CASCADE;
DROP TABLE IF EXISTS outbox_events CASCADE;

-- 3. Remove vehicle_type and vehicle_plate from drivers
ALTER TABLE reservation.drivers
    DROP COLUMN IF EXISTS vehicle_type,
    DROP COLUMN IF EXISTS vehicle_plate;
