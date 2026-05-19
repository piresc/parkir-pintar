-- Add version column for optimistic locking on billing_records.
ALTER TABLE billing.billing_records ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 0;
