-- !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
-- WARNING: This migration DROPS ALL application tables and schemas.
-- All data will be permanently lost. Do NOT run in production without
-- a verified backup and explicit approval.
-- !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

BEGIN;

-- Drop tables in reverse dependency order (foreign keys first)

-- Analytics schema
DROP TABLE IF EXISTS analytics.spot_snapshot;
DROP TABLE IF EXISTS analytics.reservation_events;

-- Presence schema
DROP TABLE IF EXISTS presence.presence_logs;

-- Search schema (read model)
DROP TABLE IF EXISTS search.spot_read_model;

-- Payment schema (references billing)
DROP TABLE IF EXISTS payment.payments;

-- Billing schema
DROP TABLE IF EXISTS billing.billing_records;

-- Reservation schema (dependent tables first)
DROP TABLE IF EXISTS reservation.reservation_events;
DROP TABLE IF EXISTS reservation.reservations;
DROP TABLE IF EXISTS reservation.parking_spots;
DROP TABLE IF EXISTS reservation.drivers;

-- Drop schemas (now empty)
DROP SCHEMA IF EXISTS analytics;
DROP SCHEMA IF EXISTS presence;
DROP SCHEMA IF EXISTS search;
DROP SCHEMA IF EXISTS payment;
DROP SCHEMA IF EXISTS billing;
DROP SCHEMA IF EXISTS reservation;

-- Extensions
DROP EXTENSION IF EXISTS "uuid-ossp";

COMMIT;
