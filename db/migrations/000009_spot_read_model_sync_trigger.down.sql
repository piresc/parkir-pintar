-- 000009_spot_read_model_sync_trigger.down.sql
-- Remove the sync trigger and function.

DROP TRIGGER IF EXISTS trg_sync_spot_read_model ON reservation.parking_spots;
DROP FUNCTION IF EXISTS reservation.sync_spot_to_read_model();
