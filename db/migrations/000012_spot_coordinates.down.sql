-- Remove GPS coordinates from parking spots.
ALTER TABLE reservation.parking_spots
    DROP COLUMN IF EXISTS latitude,
    DROP COLUMN IF EXISTS longitude;

ALTER TABLE search.spot_read_model
    DROP COLUMN IF EXISTS latitude,
    DROP COLUMN IF EXISTS longitude;
