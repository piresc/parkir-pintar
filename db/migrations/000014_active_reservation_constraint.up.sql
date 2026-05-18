-- 000014_active_reservation_constraint
-- Enforce at most one active reservation per driver at the database level.
CREATE UNIQUE INDEX idx_reservations_one_active_per_driver
    ON reservation.reservations (driver_id)
    WHERE status IN ('waiting_payment', 'confirmed', 'checked_in');
