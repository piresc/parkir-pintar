ALTER TABLE billing.billing_records
    DROP CONSTRAINT IF EXISTS uq_billing_records_idempotency_key;

ALTER TABLE billing.billing_records
    DROP CONSTRAINT IF EXISTS uq_billing_records_reservation_id;
