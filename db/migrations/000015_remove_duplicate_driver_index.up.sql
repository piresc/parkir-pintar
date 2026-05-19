-- 000016_remove_duplicate_driver_index.up.sql
-- Drop the older, redundant partial unique index from migration 000006.
-- The equivalent index idx_reservations_one_active_per_driver (from 000014) is kept.
DROP INDEX IF EXISTS reservation.idx_reservations_active_driver;
