-- 000004_schema_per_service.down.sql
-- Rollback: move all tables back to public schema.

-- Drop search read model
DROP TABLE IF EXISTS search.spot_read_model;

-- Move tables back to public
ALTER TABLE reservation.drivers SET SCHEMA public;
ALTER TABLE reservation.parking_spots SET SCHEMA public;
ALTER TABLE reservation.reservations SET SCHEMA public;
ALTER TABLE billing.billing_records SET SCHEMA public;
ALTER TABLE billing.penalties SET SCHEMA public;
ALTER TABLE payment.payments SET SCHEMA public;
ALTER TABLE presence.presence_logs SET SCHEMA public;

-- Re-add cross-domain foreign keys
ALTER TABLE public.billing_records
    ADD CONSTRAINT billing_records_reservation_id_fkey
    FOREIGN KEY (reservation_id) REFERENCES public.reservations(id);
ALTER TABLE public.payments
    ADD CONSTRAINT payments_billing_id_fkey
    FOREIGN KEY (billing_id) REFERENCES public.billing_records(id);
ALTER TABLE public.penalties
    ADD CONSTRAINT penalties_reservation_id_fkey
    FOREIGN KEY (reservation_id) REFERENCES public.reservations(id);
ALTER TABLE public.presence_logs
    ADD CONSTRAINT presence_logs_reservation_id_fkey
    FOREIGN KEY (reservation_id) REFERENCES public.reservations(id);

-- Drop schemas
DROP SCHEMA IF EXISTS reservation;
DROP SCHEMA IF EXISTS billing;
DROP SCHEMA IF EXISTS payment;
DROP SCHEMA IF EXISTS presence;
DROP SCHEMA IF EXISTS search;
