-- 000016_add_completed_status.up.sql
-- Add 'completed' status to reservations to distinguish between
-- "checked out but unpaid" and "checked out and payment completed".
-- This makes the state machine unambiguous: checked_out = awaiting payment,
-- completed = payment processed and spot released.

ALTER TABLE reservation.reservations DROP CONSTRAINT IF EXISTS reservations_status_check;
ALTER TABLE reservation.reservations ADD CONSTRAINT reservations_status_check
  CHECK (status IN (
    'pending', 'waiting_payment', 'confirmed', 'checked_in',
    'checked_out', 'completed', 'expired', 'cancelled', 'failed'
  ));
