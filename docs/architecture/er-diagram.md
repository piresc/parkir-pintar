# Entity-Relationship Diagram — ParkirPintar

## Overview

ParkirPintar uses a schema-per-service pattern within a single PostgreSQL instance. Each microservice owns its schema. Cross-service data access happens exclusively through gRPC and NATS events.

---

## ER Diagram (Mermaid)

```mermaid
erDiagram
    %% === Reservation Schema (reservation.*) ===
    drivers {
        text id PK
        varchar name
        varchar phone UK
        varchar email UK
        timestamptz created_at
        timestamptz updated_at
    }

    parking_spots {
        uuid id PK
        int floor_number
        int spot_number
        varchar vehicle_type "car | motorcycle"
        varchar spot_code UK
        varchar status "available | reserved | occupied"
        timestamptz created_at
        timestamptz updated_at
    }

    reservations {
        uuid id PK
        text driver_id FK
        uuid spot_id FK
        varchar vehicle_type "car | motorcycle"
        varchar license_plate
        varchar status "waiting_payment | confirmed | checked_in | checked_out | completed | cancelled | expired | failed"
        timestamptz check_in_time
        timestamptz check_out_time
        bigint booking_fee
        bigint hourly_rate
        bigint total_amount
        bigint overnight_fee
        varchar idempotency_key UK
        timestamptz created_at
        timestamptz updated_at
    }

    reservation_events {
        uuid id PK
        uuid reservation_id FK
        varchar event_type
        varchar old_status
        varchar new_status
        jsonb metadata
        timestamptz created_at
    }

    %% === Billing Schema (billing.*) ===
    billing_records {
        uuid id PK
        uuid reservation_id
        varchar status "pending | calculated | invoiced | paid"
        bigint booking_fee
        bigint hourly_fee
        bigint overnight_fee
        bigint total_amount
        int version
        timestamptz created_at
        timestamptz updated_at
    }

    %% === Payment Schema (payment.*) ===
    payments {
        uuid id PK
        uuid reservation_id
        bigint amount
        varchar currency "IDR"
        varchar status "pending | success | failed | refunded"
        varchar payment_method "qris"
        varchar external_transaction_id
        varchar idempotency_key UK
        timestamptz created_at
        timestamptz updated_at
    }

    %% === Presence Schema (presence.*) ===
    presence_logs {
        uuid id PK
        uuid reservation_id
        uuid spot_id
        varchar status "occupied | empty"
        boolean is_correct_spot
        timestamptz recorded_at
        timestamptz created_at
    }

    %% === Search Schema (search.*) ===
    spot_cache {
        uuid spot_id PK
        int floor_number
        int spot_number
        varchar vehicle_type
        varchar spot_code
        varchar status
        timestamptz synced_at
    }

    %% === Relationships ===
    drivers ||--o{ reservations : "makes"
    parking_spots ||--o{ reservations : "reserved_for"
    reservations ||--o{ reservation_events : "emits"
    reservations ||--o| billing_records : "generates"
    reservations ||--o{ payments : "paid_by"
    reservations ||--o{ presence_logs : "verified_by"
    parking_spots ||--|| spot_cache : "projected_to"
```

---

## Schema Ownership

| Schema | Service | Tables |
|--------|---------|--------|
| `reservation` | Reservation | `drivers`, `parking_spots`, `reservations`, `reservation_events` |
| `billing` | Billing | `billing_records` |
| `payment` | Payment | `payments` |
| `presence` | Presence | `presence_logs` |
| `search` | Search | `spot_cache` |

All schemas live in a single `parkir_pintar` database. Services never query another service's schema directly.

---

## Cross-Service References

Services store IDs from other services but do not enforce foreign keys across schemas:

- `billing_records.reservation_id` → references `reservations.id` (no FK)
- `payments.reservation_id` → references `reservations.id` (no FK)
- `presence_logs.reservation_id` → references `reservations.id` (no FK)
- `presence_logs.spot_id` → references `parking_spots.id` (no FK)

Data consistency across services is maintained through NATS events and eventual consistency.

---

## Index Strategy

```sql
-- Reservation
CREATE INDEX idx_parking_spots_availability ON reservation.parking_spots (vehicle_type, status, floor_number);
CREATE UNIQUE INDEX parking_spots_spot_code_key ON reservation.parking_spots (spot_code);
CREATE INDEX idx_reservations_driver ON reservation.reservations (driver_id);
CREATE INDEX idx_reservations_spot_status ON reservation.reservations (spot_id, status);
CREATE UNIQUE INDEX idx_reservations_idempotency ON reservation.reservations (idempotency_key);

-- Billing
CREATE INDEX idx_billing_reservation ON billing.billing_records (reservation_id);

-- Payment
CREATE INDEX idx_payments_reservation ON payment.payments (reservation_id);
CREATE UNIQUE INDEX idx_payments_idempotency ON payment.payments (idempotency_key);

-- Presence
CREATE INDEX idx_presence_reservation_time ON presence.presence_logs (reservation_id, recorded_at);
```
