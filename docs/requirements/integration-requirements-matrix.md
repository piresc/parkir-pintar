# Integration Requirements Matrix

**Project:** ParkirPintar — Smart Parking Backend System  
**Version:** 1.0  
**Date:** 2026-05-13  
**Status:** Approved

---

## 1. Service-to-Service Integration Matrix

### 1.1 Communication Grid (7×7)

Legend: `gRPC` = synchronous gRPC call, `NATS` = async event via NATS JetStream, `—` = no direct communication

| From ↓ / To → | Gateway | Search | Reservation | Billing | Payment | Presence | Notification |
|----------------|---------|--------|-------------|---------|---------|----------|--------------|
| **Gateway** | — | gRPC | gRPC | — | gRPC | gRPC | — |
| **Search** | — | — | — | — | — | — | — |
| **Reservation** | — | NATS | — | gRPC | gRPC | — | NATS |
| **Billing** | — | — | — | — | — | — | NATS |
| **Payment** | — | — | — | — | — | — | NATS |
| **Presence** | — | — | — | — | — | — | NATS |
| **Notification** | — | — | — | — | — | — | — |

### 1.2 Communication Summary

| Service Pair | Direction | Pattern | Protocol | Purpose |
|--------------|-----------|---------|----------|---------|
| Gateway → Search | Unidirectional | Request-Response | gRPC | Availability queries, floor maps, spot details |
| Gateway → Reservation | Unidirectional | Request-Response | gRPC | Create, cancel, confirm, check-in, check-out, list |
| Gateway → Payment | Unidirectional | Request-Response | gRPC | Payment status queries |
| Gateway → Presence | Unidirectional | Request-Response + Streaming | gRPC | Location streaming, arrival/wrong-spot detection |
| Reservation → Billing | Unidirectional | Request-Response | gRPC | StartBilling, CalculateFee, GenerateInvoice, ApplyPenalty |
| Reservation → Payment | Unidirectional | Request-Response | gRPC | ProcessPayment for booking fee and checkout |
| Reservation → Search | Unidirectional | Pub-Sub | NATS | Cache invalidation on reservation state changes |
| Reservation → Notification | Unidirectional | Pub-Sub | NATS | Notify driver of reservation events |
| Billing → Notification | Unidirectional | Pub-Sub | NATS | Notify driver of billing events |
| Payment → Notification | Unidirectional | Pub-Sub | NATS | Notify driver of payment success/failure |
| Presence → Notification | Unidirectional | Pub-Sub | NATS | Notify driver of arrival/wrong-spot detection |

---

## 2. External Integrations

| External System | Integrating Service | Protocol | Purpose | Status |
|-----------------|--------------------|---------|---------|---------| 
| **Payment Gateway (Midtrans/Xendit)** | Payment Service | HTTPS REST | Process QRIS, credit card, e-wallet payments | Stub (simulated) |
| **Push Notification (FCM/APNs)** | Notification Service | HTTPS REST | Send push notifications to mobile devices | Stub (logs payload) |
| **SMS Provider (Twilio/local)** | Notification Service | HTTPS REST | Send SMS for critical events | Stub (logs payload) |
| **Email Provider (SendGrid/SES)** | Notification Service | HTTPS REST/SMTP | Send invoices and receipts | Stub (logs payload) |
| **Grafana Alloy (OTel Collector)** | All Services | OTLP gRPC | Export traces, metrics, logs | Active |
| **PostgreSQL** | All Services (except Gateway, Notification) | TCP (pgx driver) | Primary data store | Active |
| **Redis** | Gateway, Search, Reservation | TCP (go-redis) | Caching, distributed locks, rate limiting | Active |
| **NATS JetStream** | Reservation, Billing, Payment, Presence, Search, Notification | TCP (nats.go) | Async event bus | Active |

---

## 3. Integration Patterns per Service Pair

### 3.1 Synchronous (gRPC) Integrations

| Caller | Callee | RPC Methods | Timeout | Retry Policy | Circuit Breaker |
|--------|--------|-------------|---------|--------------|-----------------|
| Gateway | Search | `GetAvailability`, `GetFloorMap`, `GetSpotDetails` | 5s | 3 retries, exponential backoff | Yes (5 failures → open) |
| Gateway | Reservation | `CreateReservation`, `GetReservation`, `CancelReservation`, `CheckIn`, `CheckOut`, `ConfirmReservation`, `CompleteCheckout`, `ListByDriver` | 10s | 2 retries | Yes |
| Gateway | Payment | `GetPaymentStatus` | 5s | 3 retries | Yes |
| Gateway | Presence | `StreamLocation`, `DetectArrival`, `DetectWrongSpot`, `GetPresence` | 30s (stream), 5s (unary) | 2 retries (unary only) | Yes |
| Reservation | Billing | `StartBilling`, `CalculateFee`, `GenerateInvoice`, `ApplyPenalty`, `ApplyOvernightFee` | 10s | 2 retries | Yes |
| Reservation | Payment | `ProcessPayment`, `ProcessQRIS` | 30s | 1 retry (idempotent) | Yes |

### 3.2 Asynchronous (NATS JetStream) Integrations

| Publisher | Stream | Subjects | Subscribers | Delivery | Consumer Type |
|-----------|--------|----------|-------------|----------|---------------|
| Reservation | `RESERVATIONS` | `reservation.confirmed`, `reservation.checked_in`, `reservation.checked_out`, `reservation.expired`, `reservation.cancelled` | Search, Notification | At-least-once | Durable push |
| Billing | `BILLING` | `billing.calculated`, `billing.invoiced` | Notification | At-least-once | Durable push |
| Payment | `PAYMENTS` | `payment.success`, `payment.failed` | Notification | At-least-once | Durable push |
| Presence | `PRESENCE` | `presence.arrival`, `presence.wrong_spot` | Notification | At-least-once | Durable push |

---

## 4. Data Contracts

### 4.1 Proto File References

| Service | Proto File | Package | Go Package |
|---------|-----------|---------|------------|
| Search | `proto/search/v1/search.proto` | `search.v1` | `parkir-pintar/proto/search/v1;searchv1` |
| Reservation | `proto/reservation/v1/reservation.proto` | `reservation.v1` | `parkir-pintar/proto/reservation/v1;reservationv1` |
| Billing | `proto/billing/v1/billing.proto` | `billing.v1` | `parkir-pintar/proto/billing/v1;billingv1` |
| Payment | `proto/payment/v1/payment.proto` | `payment.v1` | `parkir-pintar/proto/payment/v1;paymentv1` |
| Presence | `proto/presence/v1/presence.proto` | `presence.v1` | `parkir-pintar/proto/presence/v1;presencev1` |
| Notification | `proto/notification/v1/notification.proto` | `notification.v1` | `parkir-pintar/proto/notification/v1;notificationv1` |

### 4.2 NATS Event Payload Contracts

Events are published as JSON-encoded payloads on NATS JetStream subjects.

```json
// reservation.confirmed
{
  "reservation_id": "uuid",
  "driver_id": "uuid",
  "spot_id": "uuid",
  "spot_code": "A1-001",
  "vehicle_type": "car",
  "confirmed_at": "2026-05-13T10:00:00Z"
}

// reservation.checked_out
{
  "reservation_id": "uuid",
  "driver_id": "uuid",
  "spot_id": "uuid",
  "checked_out_at": "2026-05-13T14:00:00Z",
  "total_amount": 25000
}

// payment.success
{
  "payment_id": "uuid",
  "billing_id": "uuid",
  "amount": 25000,
  "payment_method": "qris",
  "transaction_ref": "TXN-123456",
  "paid_at": "2026-05-13T14:00:05Z"
}

// presence.arrival
{
  "reservation_id": "uuid",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "detected_at": "2026-05-13T09:55:00Z"
}
```

### 4.3 Shared Dependencies

| Dependency | Version | Used By |
|-----------|---------|---------|
| `google/protobuf/timestamp.proto` | proto3 | Reservation, Billing, Payment, Presence |
| `google.golang.org/grpc` | v1.80.0 | All services |
| `github.com/nats-io/nats.go` | latest | Reservation, Billing, Payment, Presence, Search, Notification |

---

## 5. SLA Requirements per Integration

### 5.1 Synchronous (gRPC) SLAs

| Integration | Latency P95 | Latency P99 | Availability | Error Budget |
|-------------|-------------|-------------|--------------|--------------|
| Gateway → Search | < 50ms | < 100ms | 99.95% | 0.05% |
| Gateway → Reservation | < 200ms | < 500ms | 99.9% | 0.1% |
| Gateway → Payment | < 100ms | < 300ms | 99.9% | 0.1% |
| Gateway → Presence (unary) | < 100ms | < 200ms | 99.5% | 0.5% |
| Gateway → Presence (stream) | N/A (streaming) | N/A | 99.5% | 0.5% |
| Reservation → Billing | < 100ms | < 300ms | 99.9% | 0.1% |
| Reservation → Payment | < 2000ms | < 5000ms | 99.5% | 0.5% |

### 5.2 Asynchronous (NATS) SLAs

| Integration | Delivery Latency | Max Retry | Dead Letter | Availability |
|-------------|-----------------|-----------|-------------|--------------|
| Reservation → Search (cache invalidation) | < 100ms | 5 | Yes | 99.9% |
| Reservation → Notification | < 500ms | 10 | Yes | 99.5% |
| Billing → Notification | < 500ms | 10 | Yes | 99.5% |
| Payment → Notification | < 500ms | 10 | Yes | 99.5% |
| Presence → Notification | < 500ms | 10 | Yes | 99.5% |

### 5.3 External Integration SLAs

| External System | Expected Latency | Timeout | Fallback |
|-----------------|-----------------|---------|----------|
| Payment Gateway | < 3000ms | 30s | Queue for retry, return pending status |
| Push Notification (FCM) | < 1000ms | 10s | Log and retry async |
| SMS Provider | < 2000ms | 15s | Log and retry async |
| Email Provider | < 5000ms | 30s | Queue for batch delivery |

---

## 6. Failure Modes and Fallback Behavior

### 6.1 Service Failure Matrix

| Failed Service | Impact | Fallback Behavior | Recovery |
|----------------|--------|-------------------|----------|
| **Search** | Drivers cannot query availability | Return stale cached data (Redis) | Auto-restart, health probe |
| **Reservation** | Cannot create/modify reservations | Gateway returns 503, client retries | Auto-restart, lock cleanup |
| **Billing** | Cannot calculate fees or generate invoices | Reservation proceeds, billing deferred | Retry via NATS event replay |
| **Payment** | Cannot process payments | Return "pending" status, queue for retry | Circuit breaker opens, retry on close |
| **Presence** | Location tracking unavailable | Graceful degradation, skip arrival detection | Auto-restart, no data loss (stateless) |
| **Notification** | Notifications not delivered | Silent failure, events remain in NATS | Consumer catches up on restart |
| **Gateway** | All client requests fail | Traefik returns 502, client retries | Auto-restart, multiple replicas |

### 6.2 Infrastructure Failure Matrix

| Failed Component | Impact | Fallback Behavior | Recovery |
|------------------|--------|-------------------|----------|
| **PostgreSQL** | All data operations fail | Services return 503, no writes | Restart container, WAL recovery |
| **Redis** | Cache miss, lock failure | Fallback to DB queries, skip distributed lock (use DB lock) | Auto-reconnect |
| **NATS** | Events not delivered | Publishers buffer locally, consumers stall | Reconnect, replay from last ACK |
| **Grafana Alloy** | Observability data lost | Services continue operating, telemetry dropped | Restart collector, data gap accepted |

### 6.3 Cascading Failure Prevention

| Pattern | Implementation | Scope |
|---------|---------------|-------|
| Circuit Breaker | `pkg/circuitbreaker/` — 5 failures → open, 30s half-open | All gRPC client calls |
| Bulkhead | Separate connection pools per downstream service | Reservation → Billing, Payment |
| Timeout | Context-based deadlines on all RPCs | All gRPC calls |
| Retry with Backoff | Exponential backoff (100ms, 200ms, 400ms) with jitter | Transient failures only |
| Graceful Degradation | Non-critical services (Notification, Presence) fail silently | Event subscribers |
| Singleflight | Coalesce concurrent identical requests | Search cache miss |

---

## 7. Integration Testing Strategy

### 7.1 Test Levels

| Level | Scope | Tools | Frequency |
|-------|-------|-------|-----------|
| **Unit** | Individual service handlers with mocked dependencies | Go `testing`, `testify/mock` | Every commit |
| **Integration** | Service + real database/Redis/NATS | `testcontainers-go`, Docker Compose | Every PR |
| **Contract** | Proto compatibility between caller and callee | `buf breaking` (protobuf linting) | Every proto change |
| **End-to-End** | Full flow through all services | Docker Compose + HTTP client | Nightly / pre-release |
| **Load** | Performance under expected traffic | `k6`, custom Go load tests | Weekly / pre-release |

### 7.2 Integration Test Scenarios

| # | Scenario | Services Involved | Validation |
|---|----------|-------------------|------------|
| 1 | Happy path: search → reserve → confirm → check-in → check-out → pay | All 7 | Reservation status transitions, billing amounts, payment status |
| 2 | Double-booking prevention | Reservation, Redis | Second reservation for same spot returns error |
| 3 | Reservation expiry | Reservation (worker) | Unconfirmed reservation expires after timeout |
| 4 | Wrong-spot penalty | Presence, Billing, Notification | Penalty applied and notification sent |
| 5 | Payment failure and retry | Reservation, Payment | Circuit breaker opens, retry succeeds after recovery |
| 6 | Cache invalidation | Reservation, Search, NATS | Search returns updated availability after reservation |
| 7 | Notification delivery | All publishers, Notification, NATS | All event types trigger appropriate notifications |
| 8 | Concurrent reservations (race) | Reservation, Redis, PostgreSQL | Only one reservation succeeds per spot |

### 7.3 Test Environment

```yaml
# docker-compose.test.yml (simplified)
services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: parkir_pintar_test
  redis:
    image: redis:7
  nats:
    image: nats:latest
    command: ["--jetstream"]
  # All 7 services built from source
  gateway:
    build: .
    command: ["./gateway"]
  search:
    build: .
    command: ["./search"]
  # ... remaining services
```

### 7.4 Contract Testing Process

1. **Proto changes** trigger `buf breaking` check against `main` branch
2. **Backward-compatible** changes (new fields, new RPCs) pass automatically
3. **Breaking changes** (removed fields, renamed RPCs) require version bump (`v1` → `v2`)
4. **Consumer-driven contracts** validated via integration test suite
5. **NATS event schemas** validated via JSON Schema in test assertions

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-05-13 | Engineering Team | Initial integration requirements matrix |
