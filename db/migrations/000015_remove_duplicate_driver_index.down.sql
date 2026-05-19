-- 000016_remove_duplicate_driver_index.down.sql
-- Recreate the index that was dropped.
CREATE UNIQUE INDEX idx_reservations_active_driver
    ON reservation.reservations (driver_id)
    WHERE status IN ('waiting_payment', 'confirmed', 'checked_in');
