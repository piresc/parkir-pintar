-- 000002_parkir_pintar.down.sql
-- Revert ParkirPintar schema: drop all tables and indexes in reverse order.

DROP INDEX IF EXISTS idx_presence_reservation_time;
DROP INDEX IF EXISTS idx_payments_billing;
DROP INDEX IF EXISTS idx_billing_reservation;
DROP INDEX IF EXISTS idx_reservations_expiry;
DROP INDEX IF EXISTS idx_parking_spots_availability;
DROP INDEX IF EXISTS idx_reservations_driver;
DROP INDEX IF EXISTS idx_reservations_active_spot;

DROP TABLE IF EXISTS presence_logs;
DROP TABLE IF EXISTS penalties;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS billing_records;
DROP TABLE IF EXISTS reservations;
DROP TABLE IF EXISTS parking_spots;
DROP TABLE IF EXISTS drivers;
