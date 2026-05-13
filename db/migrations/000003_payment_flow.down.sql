-- 000003_payment_flow.down.sql
-- Revert payment flow changes: restore original status constraint and index.

-- Remove payment timeout worker index
DROP INDEX IF EXISTS idx_reservations_stale_payment;

-- Restore original double-booking index (without waiting_payment)
DROP INDEX IF EXISTS idx_reservations_active_spot;
CREATE UNIQUE INDEX idx_reservations_active_spot
  ON reservations (spot_id)
  WHERE status IN ('confirmed', 'checked_in');

-- Restore original status CHECK constraint (without waiting_payment and failed)
ALTER TABLE reservations DROP CONSTRAINT reservations_status_check;
ALTER TABLE reservations ADD CONSTRAINT reservations_status_check
  CHECK (status IN (
    'pending', 'confirmed', 'checked_in',
    'checked_out', 'expired', 'cancelled'
  ));
