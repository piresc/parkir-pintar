-- 000003_payment_flow.up.sql
-- Add waiting_payment and failed statuses to reservations.
-- Extend double-booking prevention to cover waiting_payment state.
-- Add index for payment timeout worker scans.

-- Expand status CHECK constraint
ALTER TABLE reservations DROP CONSTRAINT reservations_status_check;
ALTER TABLE reservations ADD CONSTRAINT reservations_status_check
  CHECK (status IN (
    'pending', 'waiting_payment', 'confirmed', 'checked_in',
    'checked_out', 'expired', 'cancelled', 'failed'
  ));

-- Include waiting_payment in double-booking partial unique index
DROP INDEX IF EXISTS idx_reservations_active_spot;
CREATE UNIQUE INDEX idx_reservations_active_spot
  ON reservations (spot_id)
  WHERE status IN ('waiting_payment', 'confirmed', 'checked_in');

-- Worker scan index for payment timeout detection
CREATE INDEX IF NOT EXISTS idx_reservations_stale_payment
  ON reservations (status, created_at)
  WHERE status = 'waiting_payment';
