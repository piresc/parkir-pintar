-- 000016_add_completed_status.down.sql
-- Revert: remove 'completed' from reservations status constraint.

-- First update any existing 'completed' rows back to 'checked_out'
UPDATE reservation.reservations SET status = 'checked_out' WHERE status = 'completed';

ALTER TABLE reservation.reservations DROP CONSTRAINT IF EXISTS reservations_status_check;
ALTER TABLE reservation.reservations ADD CONSTRAINT reservations_status_check
  CHECK (status IN (
    'pending', 'waiting_payment', 'confirmed', 'checked_in',
    'checked_out', 'expired', 'cancelled', 'failed'
  ));
