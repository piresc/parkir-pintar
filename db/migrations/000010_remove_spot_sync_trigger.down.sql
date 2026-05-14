-- 000010_remove_spot_sync_trigger.down.sql
-- Restore the DB trigger for spot_read_model sync (rollback from NATS).

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
