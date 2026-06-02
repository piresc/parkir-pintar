BEGIN;

DELETE FROM reservation.drivers WHERE id IN ('driver-1', 'driver-2', 'driver-3', 'driver-4');

COMMIT;
