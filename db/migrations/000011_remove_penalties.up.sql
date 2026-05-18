-- 000009_remove_penalties.up.sql
-- Remove penalty/cancellation fee system. Business rule: driver forfeits booking fee only.

DROP TABLE IF EXISTS billing.penalties;

ALTER TABLE billing.billing_records DROP COLUMN IF EXISTS cancellation_fee;
ALTER TABLE billing.billing_records DROP COLUMN IF EXISTS penalty_amount;
