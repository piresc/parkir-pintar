-- 000013_billing_idempotency_constraint
-- Prevent duplicate billing records via DB constraint.
ALTER TABLE billing.billing_records
    ADD CONSTRAINT uq_billing_records_idempotency_key UNIQUE (idempotency_key);

ALTER TABLE billing.billing_records
    ADD CONSTRAINT uq_billing_records_reservation_id UNIQUE (reservation_id);
