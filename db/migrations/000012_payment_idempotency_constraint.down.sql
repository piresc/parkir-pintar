-- 000012_payment_idempotency_constraint (rollback)
ALTER TABLE payment.payments
    DROP CONSTRAINT IF EXISTS uq_payments_idempotency_key;
