-- 000009_spot_read_model_sync_trigger.up.sql
-- Add a trigger on reservation.parking_spots to automatically sync status
-- changes to search.spot_read_model, ensuring microservice table isolation.
-- The search service reads only from spot_read_model; this trigger keeps it
-- in sync without requiring the search service to access parking_spots.

CREATE OR REPLACE FUNCTION reservation.sync_spot_to_read_model()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO search.spot_read_model (id, floor_number, spot_number, vehicle_type, spot_code, status, updated_at)
    VALUES (NEW.id, NEW.floor_number, NEW.spot_number, NEW.vehicle_type, NEW.spot_code, NEW.status, NOW())
    ON CONFLICT (id) DO UPDATE SET
        floor_number = EXCLUDED.floor_number,
        spot_number  = EXCLUDED.spot_number,
        vehicle_type = EXCLUDED.vehicle_type,
        spot_code    = EXCLUDED.spot_code,
        status       = EXCLUDED.status,
        updated_at   = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sync_spot_read_model
    AFTER INSERT OR UPDATE ON reservation.parking_spots
    FOR EACH ROW
    EXECUTE FUNCTION reservation.sync_spot_to_read_model();
