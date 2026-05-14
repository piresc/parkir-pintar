-- 000010_remove_spot_sync_trigger.up.sql
-- Remove the DB trigger that synced parking_spots to spot_read_model.
-- This is now handled by NATS JetStream events from the reservation service
-- to the search service (subject: reservation.search.spot-updated).

DROP TRIGGER IF EXISTS trg_sync_spot_read_model ON reservation.parking_spots;
DROP FUNCTION IF EXISTS reservation.sync_spot_to_read_model();
