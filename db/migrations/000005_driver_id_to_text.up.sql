-- 000005_driver_id_to_text.up.sql
-- Change drivers.id and reservations.driver_id from UUID to TEXT
-- to support simple user IDs like "driver-1", "driver-2", etc.

-- 1. Drop FK constraint
ALTER TABLE reservation.reservations
    DROP CONSTRAINT IF EXISTS reservations_driver_id_fkey;

-- 2. Change drivers.id from UUID to TEXT
ALTER TABLE reservation.drivers
    ALTER COLUMN id SET DEFAULT NULL,
    ALTER COLUMN id TYPE TEXT USING id::TEXT;

-- 3. Change reservations.driver_id from UUID to TEXT
ALTER TABLE reservation.reservations
    ALTER COLUMN driver_id TYPE TEXT USING driver_id::TEXT;

-- 4. Re-add FK constraint
ALTER TABLE reservation.reservations
    ADD CONSTRAINT reservations_driver_id_fkey
    FOREIGN KEY (driver_id) REFERENCES reservation.drivers(id);

-- 5. Seed test drivers
INSERT INTO reservation.drivers (id, name, phone, email, vehicle_type, vehicle_plate) VALUES
    ('driver-1', 'Test Driver 1', '+6281000001', 'driver1@test.com', 'car', 'B 1234 ABC'),
    ('driver-2', 'Test Driver 2', '+6281000002', 'driver2@test.com', 'car', 'B 5678 DEF'),
    ('driver-3', 'Test Driver 3', '+6281000003', 'driver3@test.com', 'motorcycle', 'B 9012 GHI'),
    ('driver-4', 'Test Driver 4', '+6281000004', 'driver4@test.com', 'motorcycle', 'B 3456 JKL')
ON CONFLICT (id) DO NOTHING;
