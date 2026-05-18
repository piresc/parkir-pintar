-- 000009_remove_penalties.down.sql
-- Recreate penalties table and billing_records columns removed in up migration.

CREATE TABLE IF NOT EXISTS billing.penalties (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reservation_id UUID        NOT NULL REFERENCES reservation.reservations(id),
    penalty_type   VARCHAR(20) NOT NULL CHECK (penalty_type IN ('wrong_spot', 'cancellation')),
    amount         BIGINT      NOT NULL,
    description    TEXT,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

ALTER TABLE billing.billing_records ADD COLUMN IF NOT EXISTS cancellation_fee BIGINT NOT NULL DEFAULT 0;
ALTER TABLE billing.billing_records ADD COLUMN IF NOT EXISTS penalty_amount BIGINT NOT NULL DEFAULT 0;
