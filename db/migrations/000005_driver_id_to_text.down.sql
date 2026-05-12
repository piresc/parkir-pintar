-- 000005_driver_id_to_text.down.sql
-- Revert drivers.id and reservations.driver_id back to UUID

-- Remove test drivers
DELETE FROM reservation.drivers WHERE id IN ('driver-1', 'driver-2', 'driver-3', 'driver-4');

-- Drop FK
ALTER TABLE reservation.reservations
    DROP CONSTRAINT IF EXISTS reservations_driver_id_fkey;

-- Revert to UUID
ALTER TABLE reservation.reservations
    ALTER COLUMN driver_id TYPE UUID USING driver_id::UUID;

ALTER TABLE reservation.drivers
    ALTER COLUMN id TYPE UUID USING id::UUID,
    ALTER COLUMN id SET DEFAULT uuid_generate_v4();

-- Re-add FK
ALTER TABLE reservation.reservations
    ADD CONSTRAINT reservations_driver_id_fkey
    FOREIGN KEY (driver_id) REFERENCES reservation.drivers(id);
