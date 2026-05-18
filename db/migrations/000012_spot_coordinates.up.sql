-- Add GPS coordinates to parking spots for presence verification.
ALTER TABLE reservation.parking_spots
    ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION;

-- Also add to search read model for API access.
ALTER TABLE search.spot_read_model
    ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION;

-- Seed sample coordinates for a parking building in Jakarta.
-- Base: -6.2088, 106.8456 (Monas area), with small offsets per floor/spot.
-- Update all spots with coordinates based on floor_number.
UPDATE reservation.parking_spots SET
    latitude = -6.2088 + (floor_number - 1) * 0.00001,
    longitude = 106.8456 + (spot_number - 1) * 0.000005;

UPDATE search.spot_read_model SET
    latitude = -6.2088 + (floor_number - 1) * 0.00001,
    longitude = 106.8456 + (spot_number - 1) * 0.000005;
