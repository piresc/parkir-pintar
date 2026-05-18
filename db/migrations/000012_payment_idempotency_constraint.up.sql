-- 000012_payment_idempotency_constraint
-- Adds unique constraint on idempotency_key to prevent TOCTOU race in payment creation.
ALTER TABLE payment.payments
    ADD CONSTRAINT uq_payments_idempotency_key UNIQUE (idempotency_key);
