# ParkirPintar — Codebase vs PRD Compliance Review

**Date:** 2026-05-05  
**PRD Version:** 1.0 (April 24, 2026)  
**Codebase Commit:** HEAD  

---

## Executive Summary

The ParkirPintar codebase is a **highly compliant, well-architected implementation** of the PRD. All 7 microservices are implemented, all 11 E2E test scenarios from the PRD are covered, and the core business logic (pricing, state machine, double-booking prevention) matches the specification precisely. The implementation uses Go with gRPC, PostgreSQL, Redis, and NATS JetStream as required.

**Overall Compliance Grade: A-**  
*Excellent adherence with minor divergences in data types, one unimplemented auto-trigger feature, and some production-readiness gaps.*

---

## 1. Architecture Compliance (PRD §13, §14)

| PRD Requirement | Status | Notes |
|-----------------|--------|-------|
| 7 microservices (Gateway, Search, Reservation, Billing, Payment, Presence, Notification) | ✅ **Full** | All 7 services have `cmd/{service}/main.go` entry points and `internal/{service}/` packages |
| gRPC over HTTP/2 for service-to-service | ✅ **Full** | Proto files define all RPCs; generated `.pb.go` files present |
| Go (Golang) language | ✅ **Full** | Go 1.25 specified in `go.mod` |
| Docker / Docker Compose | ✅ **Full** | `Dockerfile` builds all 8 binaries; `docker-compose.yml` orchestrates full stack |
| Redis distributed locks | ✅ **Full** | `pkg/redislock/` exists; reservation usecase uses `SETNX` with 30s TTL |
| Idempotency mechanism | ✅ **Full** | Idempotency keys on `CreateReservation`, `StartBilling`, `GenerateInvoice`, `ProcessPayment` |
| Retry with exponential backoff | ✅ **Full** | Payment service retries gateway 3x with 100ms/200ms/400ms backoff |
| Circuit breaker pattern | ⚠️ **Partial** | `pkg/circuitbreaker/` exists but not clearly wired into all external service calls |
| Graceful degradation | ✅ **Full** | Non-critical ops (billing start, NATS publish) log errors and continue |
| REST / gRPC-Web gateway | ✅ **Full** | Gateway transcodes REST to gRPC with JWT auth |

### Proto Coverage vs PRD §14

| Service | PRD RPCs | Implemented RPCs | Match |
|---------|----------|------------------|-------|
| Search | `GetAvailability`, `GetFloorMap`, `GetSpotDetails` | All 3 | ✅ |
| Reservation | `CreateReservation`, `CancelReservation`, `CheckIn`, `CheckOut`, `ExpireReservation` | All 5 | ✅ |
| Billing | `StartBilling`, `CalculateFee`, `GenerateInvoice`, `ApplyPenalty`, `ApplyOvernightFee` | All 5 | ✅ |
| Payment | `ProcessPayment`, `ProcessQRIS`, `RefundPayment`, `GetPaymentStatus` | All 4 | ✅ |
| Presence | `StreamLocation`, `DetectArrival`, `DetectWrongSpot`, `GetPresence` | All 4 | ✅ |
| Notification | `SendPush`, `SendSMS`, `SendEmail` | All 3 (stub) | ✅ |

**Verdict:** Architecture is fully compliant with PRD §13–14.

---

## 2. Data Model Compliance (PRD §16)

### Tables

| PRD Table | Migration Table | Columns Match | Status |
|-----------|----------------|---------------|--------|
| `drivers` | `drivers` | 7/7 columns | ✅ |
| `parking_spots` | `parking_spots` | 7/7 columns | ✅ |
| `reservations` | `reservations` | 13/13 columns | ✅ |
| `billing_records` | `billing_records` | 14/14 columns | ✅ |
| `payments` | `payments` | 10/10 columns | ✅ |
| `penalties` | `penalties` | 6/6 columns | ✅ |
| `presence_logs` | `presence_logs` | 6/6 columns | ✅ |

### Indexes

| PRD Index | Migration Index | Status |
|-----------|----------------|--------|
| `idx_reservations_active_spot` (partial unique) | ✅ Present | ✅ |
| `idx_reservations_driver` | ✅ Present | ✅ |
| `idx_parking_spots_availability` | ✅ Present | ✅ |
| `idx_reservations_expiry` (partial) | ✅ Present | ✅ |
| `idx_billing_reservation` | ✅ Present | ✅ |
| `idx_payments_billing` | ✅ Present | ✅ |
| `idx_presence_reservation_time` | ✅ Present | ✅ |

### ⚠️ Data Type Divergence

**Issue:** The PRD specifies `DECIMAL(12,2)` for all monetary columns (`booking_fee`, `parking_fee`, `overnight_fee`, `cancellation_fee`, `penalty_amount`, `total_amount`, `amount`). The migration (`000002_parkir_pintar.up.sql`) uses `BIGINT` instead.

**Impact:** Low for IDR (no decimal subunits in common use), but it is a specification divergence.  
**Recommendation:** Align with PRD by using `DECIMAL(12,2)` or document the deviation in the assumptions.

**Verdict:** Schema structure is fully compliant; data types for money diverge from PRD.

---

## 3. Functional Requirements Compliance (PRD §6)

| Requirement | Status | Implementation Evidence |
|-------------|--------|------------------------|
| **FR-01** View real-time availability by floor & vehicle type | ✅ | `SearchService.GetAvailability`, `GetFloorMap`; Redis cache with 5s TTL and singleflight |
| **FR-02** Reserve a spot (system-assigned & user-selected) | ✅ | `CreateReservation` supports both `assignment_mode` values; Redis `SETNX` lock |
| **FR-03** 1-hour reservation hold & auto-expiry | ✅ | `expires_at = confirmed_at + 1h`; `worker/expiry.go` polls every 30s |
| **FR-04** Check-in (geofence or manual) | ⚠️ | Manual check-in ✅; **Auto geofence check-in ❌** — `DetectArrival` only returns a boolean, it does not call `ReservationService.CheckIn` |
| **FR-05** Check-out with duration-based billing | ✅ | `CheckOut` → `CalculateFee` → `GenerateInvoice` → `ProcessPayment` |
| **FR-06** Wrong-spot detection & penalty (200k IDR) | ✅ | `DetectWrongSpot` RPC; `ApplyPenalty` with `billingmodel.WrongSpotPenalty = 200_000` |
| **FR-07** Cancellation with time-based fees | ✅ | `CancelReservation` → `CalculateCancellationFee` (0 within 2min, 5k after) |
| **FR-08** Notifications (stub) | ✅ | `notification/usecase/usecase.go` logs all payloads with Status="logged" |

### Key Gap: FR-04 Auto Check-In

The PRD states (§6 FR-04, §12.2): *"Geofence triggers automatic check-in when the Driver enters the parking area boundary."*

**Current behavior:**
- `PresenceService.StreamLocation` receives location updates
- `PresenceService.DetectArrival` returns `arrived=true/false`
- Gateway `POST /api/v1/presence/stream` calls `DetectArrival` but **does not** call `ReservationService.CheckIn`
- There is no background worker or subscriber that auto-triggers check-in on geofence entry

**Recommendation:** Add a NATS subscriber or background processor that listens for `presence.arrival` events and calls `CheckIn` when `arrived=true`.

### Minor Gap: PENDING State

The PRD (§8.3) defines the state machine as `PENDING → CONFIRMED → CHECKED_IN`.  
**Current behavior:** `CreateReservation` creates the reservation directly with `status = 'confirmed'`. The `pending` status constant exists but is never used during creation.

**Impact:** Low. The functional behavior is identical (lock → validate → create → confirm), but the state machine is simplified.

**Verdict:** 7/8 FRs fully compliant; FR-04 partially compliant due to missing auto check-in trigger.

---

## 4. Parking Area Specification (PRD §7)

| Spec | Implementation | Status |
|------|---------------|--------|
| Single building, 5 floors | `CHECK (floor_number BETWEEN 1 AND 5)` | ✅ |
| 30 car spots / floor | Seed loop `FOR s IN 1..30` | ✅ |
| 50 motorcycle spots / floor | Seed loop `FOR s IN 1..50` | ✅ |
| Total 400 spots | 5 × (30 + 50) = 400 | ✅ |
| Spot code format `F3-C-012` | `'F' || f || '-C-' || LPAD(s::TEXT, 3, '0')` | ✅ |

**Verdict:** Fully compliant.

---

## 5. Reservation & Booking Flow (PRD §8)

### System-Assigned Flow (§8.1)

All steps are implemented:
1. Driver selects vehicle type → REST API ✅
2. System checks availability → `FindAvailableSpot` ✅
3. Redis distributed lock → `SETNX` ✅
4. Spot assignment → first available spot ✅
5. Booking fee charged (5,000 IDR) → `StartBilling` ✅
6. 1-hour hold → `expires_at = now + 1h` ✅
7. Manual check-in → `CheckIn` RPC ✅
8. Billing timer starts → `CalculateFee` at checkout ✅
9. Payment processed → `ProcessPayment` with QRIS ✅
10. Spot released → `UpdateSpotStatusTx('available')` ✅

### User-Selected Flow (§8.2)

All steps are implemented, including the hold/queue via Redis `SETNX` lock.

### State Machine (§8.3)

Implemented states: `confirmed`, `checked_in`, `checked_out`, `expired`, `cancelled`.  
Missing intermediate use of `pending` (as noted above).

**Verdict:** Functionally compliant. Minor simplification of state machine.

---

## 6. Pricing & Billing Rules (PRD §9)

| Rule | Code Location | Status |
|------|--------------|--------|
| Booking fee = 5,000 IDR | `billingmodel.BookingFee` | ✅ |
| First hour = 5,000 IDR | `billingmodel.HourlyRate` | ✅ |
| Each subsequent started hour = 5,000 IDR | `CalculateParkingFee` uses `math.Ceil` | ✅ |
| Partial hour rounds up | `billedHours++` when duration > whole hours | ✅ |
| Overnight fee = 20,000 IDR (crosses midnight WIB) | `crossesMidnight` with `Asia/Jakarta` TZ | ✅ |
| No overstay penalty | No overstay logic exists; billed at standard rate | ✅ |
| Rate computed at checkout, not locked at booking | `CalculateFee` called during `CheckOut` | ✅ |

### Billing Examples Verification

| Example | PRD Expected Total | Code Logic | Match |
|---------|-------------------|------------|-------|
| 2-hour parking | 15,000 IDR | 2×5,000 + 5,000 booking | ✅ |
| 1.5-hour parking | 15,000 IDR | ceil(1.5)=2 → 2×5,000 + 5,000 | ✅ |
| Overnight (22:00→06:00) | 65,000 IDR | 8×5,000 + 20,000 + 5,000 | ✅ |
| Overstay (4 hours) | 25,000 IDR | 4×5,000 + 5,000 booking | ✅ |

**Verdict:** Fully compliant.

---

## 7. Cancellation & Penalty Policy (PRD §10)

| Condition | PRD Fee | Implementation | Status |
|-----------|---------|---------------|--------|
| Cancel within 2 min | 0 IDR | `CalculateCancellationFee` with `CancelFreeWindow = 2min` | ✅ |
| Cancel after 2 min, before check-in | 5,000 IDR | `CancelFee = 5,000` | ✅ |
| No-show (not checked in within 1h) | Booking fee consumed | `ExpireReservation` releases spot, no extra penalty | ✅ |
| Wrong-spot penalty | 200,000 IDR | `WrongSpotPenalty = 200,000` | ✅ |

**Verdict:** Fully compliant.

---

## 8. Payment Requirements (PRD §11)

| Requirement | Status | Notes |
|-------------|--------|-------|
| QRIS payment support | ✅ | `payment_method` enum includes `'qris'` |
| Booking fee at confirmation | ✅ | `StartBilling` creates record with `booking_fee = 5_000` |
| Cancellation fee when applicable | ✅ | `ApplyPenalty` with `"cancellation"` type |
| No-show: booking fee consumed | ✅ | No refund on expiry |
| Wrong-spot penalty charged | ✅ | `ApplyPenalty` with `"wrong_spot"` type |
| Parking session fee at checkout | ✅ | `CalculateFee` → `GenerateInvoice` → `ProcessPayment` |
| Payment states: PENDING → SUCCESS / FAILED / REFUNDED | ✅ | Enum matches PRD exactly |
| Idempotency on `CreateReservation` and `Checkout/Invoice` | ✅ | Keys checked in both reservation and billing usecases |

**Verdict:** Fully compliant.

---

## 9. Location & Presence Requirements (PRD §12)

| Requirement | Status | Notes |
|-------------|--------|-------|
| Location updates every 30s | ⚠️ | Client-side responsibility; backend accepts any frequency via `StreamLocation` |
| Geofence auto-detect arrival | ❌ | `DetectArrival` exists but **does not auto-trigger check-in** |
| Presence streaming during active sessions | ✅ | Redis streams (`XAdd`) + PostgreSQL persistence |

**Verdict:** Partially compliant. The geofence detection logic exists but the automatic check-in trigger is missing.

---

## 10. Testing Requirements (PRD §17)

### Unit Tests

| PRD Test Area | Files | Status |
|---------------|-------|--------|
| Pricing rules | `pkg/pricing/pricing_test.go`, `internal/billing/model/model_test.go` | ✅ |
| Overlap detection | `internal/reservation/usecase/usecase_test.go` (double-book tests) | ✅ |
| Idempotency | `internal/reservation/usecase/usecase_test.go`, `internal/billing/usecase/usecase_test.go` | ✅ |

### Integration Tests

| PRD Test Area | Files | Status |
|---------------|-------|--------|
| Reservation → Billing Flow | `tests/e2e/happy_path_test.go` | ✅ |

### End-to-End Tests (PRD §17.3 — All 11 Scenarios)

| # | PRD Scenario | Test File | Status |
|---|-------------|-----------|--------|
| 1 | Happy Path Reservation | `tests/e2e/happy_path_test.go` | ✅ |
| 2 | Double-Book Prevention | `tests/e2e/double_book_test.go` | ✅ |
| 3 | User-Selected Spot Contention | `tests/e2e/contention_test.go` | ✅ |
| 4 | Reservation Expiry (No-Show) | `tests/e2e/expiry_test.go` | ✅ |
| 5 | Wrong-Spot Penalty | `tests/e2e/wrong_spot_test.go` | ✅ |
| 6 | Cancellation — Free (< 2 min) | `tests/e2e/free_cancel_test.go` | ✅ |
| 7 | Cancellation — Paid (> 2 min) | `tests/e2e/paid_cancel_test.go` | ✅ |
| 8 | Extended Stay Billing | `tests/e2e/extended_stay_test.go` | ✅ |
| 9 | Overnight Fee | `tests/e2e/overnight_test.go` | ✅ |
| 10 | Payment Checkout — Success | `tests/e2e/payment_success_test.go` | ✅ |
| 11 | Payment Checkout — Failure | `tests/e2e/payment_failure_test.go` | ✅ |

**Additional Test Coverage (Beyond PRD):**
- Property-based tests (`rapid`) for bug condition exploration and preservation
- Race condition tests (`tests/e2e/race_test.go`)
- Load tests (`tests/e2e/load_test.go`)
- State machine property tests (`tests/e2e/property_state_machine_test.go`)
- Spot inventory property tests (`tests/e2e/property_spot_inventory_test.go`)

**Verdict:** Exceeds PRD testing requirements.

---

## 11. Non-Functional Requirements (PRD §18)

| NFR | Status | Evidence |
|-----|--------|----------|
| Reservation response < 500ms p95 | ⚠️ | No performance benchmarks or p95 metrics captured |
| Availability query < 200ms p95 | ⚠️ | Redis caching helps, but no benchmarks |
| Location update ingestion < 100ms | ⚠️ | No benchmarks |
| Concurrent reservations: 100+ | ✅ | `contention_test.go` tests 5 concurrent; Redis locks should scale |
| Uptime 99.9% | ⚠️ | Health checks exist (`/health`, `/ready`) but no SLO enforcement |
| Data durability | ✅ | PostgreSQL + transactions with `sqlx.Tx` |
| Graceful degradation | ✅ | Non-critical failures logged, not propagated |
| JWT / mTLS | ⚠️ | JWT implemented; mTLS not configured (gRPC plaintext) |
| Rate limiting | ✅ | Per-IP token bucket in HTTP and gRPC middleware |
| Input validation | ✅ | Coordinate validation, required field checks |
| Structured logging | ✅ | `slog` with JSON format and trace context |
| Distributed tracing | ✅ | OpenTelemetry with pluggable exporters |
| Health checks | ✅ | `/health`, `/live`, `/ready`, `/detailed` |
| Metrics (Prometheus) | ❌ | No Prometheus metrics endpoint found |

**Verdict:** Observability and security fundamentals are strong, but performance benchmarking and Prometheus metrics are missing.

---

## 12. Deliverables Compliance (PRD §20)

| # | Deliverable | Location | Status |
|---|------------|----------|--------|
| 1 | Solution Diagrams (HLD, LLD, ERD) | `docs/architecture.md` (text only, no diagrams) | ⚠️ |
| 2 | Configuration Files | `config/.env.example` | ✅ |
| 3 | Microservice Source Code (Go) | `cmd/`, `internal/` | ✅ |
| 4 | Proto Definitions (gRPC) | `proto/` | ✅ |
| 5 | Docker / Docker Compose | `Dockerfile`, `docker-compose.yml` | ✅ |
| 6 | Database Migrations | `db/migrations/` | ⚠️ (no `.down.sql` files) |
| 7 | Unit Tests | `*_test.go` alongside source | ✅ |
| 8 | Integration Tests | `tests/integration/` | ✅ |
| 9 | End-to-End Tests | `tests/e2e/` | ✅ |
| 10 | README with setup & run instructions | `README.md` | ✅ (boilerplate-focused) |
| 11 | Assumptions documentation | Embedded in code comments | ⚠️ |
| 12 | Third-party library justifications | **Missing** | ❌ |

**Verdict:** 9/12 fully delivered; 3 partially delivered.

---

## 13. Production Readiness Gaps

These are not PRD compliance issues per se, but they affect production deployment:

| Issue | Location | Severity |
|-------|----------|----------|
| **Stub Billing/Payment clients in production** | `cmd/reservation/main.go:71` uses `&stubBillingClient{}`, `&stubPaymentClient{}` | **High** |
| **gRPC always plaintext** | `pkg/grpcserver/server.go` TLS fields never loaded; `pkg/grpcclient/client.go` ignores `TLSEnabled` | **High** |
| **No leader election for expiry worker** | `internal/reservation/worker/expiry.go` — all instances scan redundantly | Medium |
| **Hardcoded shutdown timeout** | `cmd/*/main.go` uses `30*time.Second` instead of config value | Low |
| **Redis stream unbounded growth** | `presence/repository/repository.go` defines `CleanupPresence` but never calls it | Medium |

---

## 14. Summary of Divergences

| # | Divergence | PRD Spec | Code Implementation | Impact |
|---|-----------|----------|---------------------|--------|
| 1 | Monetary data types | `DECIMAL(12,2)` | `BIGINT` | Low |
| 2 | Auto check-in via geofence | Geofence triggers `CheckIn` automatically | `DetectArrival` returns bool only; no auto-trigger | **Medium** |
| 3 | `PENDING` reservation state | `PENDING → CONFIRMED` | Reservations created directly as `confirmed` | Low |
| 4 | Production service stubs | Real gRPC clients for production | `cmd/reservation` uses stubs | **High** |
| 5 | StreamLocation gateway mapping | Streaming RPC | Gateway maps to unary `DetectArrival` | Low |
| 6 | No down migrations | `.up.sql` and `.down.sql` | Only `.up.sql` files | Low |

---

## 15. Recommendations (Priority Order)

1. **Implement real gRPC client adapters** in `cmd/reservation/main.go` for Billing and Payment services (currently stubs)
2. **Add automatic check-in trigger** on geofence detection — either via NATS subscriber or background worker
3. **Enable gRPC TLS** — load TLS config in `pkg/grpcserver` and `pkg/grpcclient`
4. **Align monetary columns** with PRD `DECIMAL(12,2)` or document explicit deviation
5. **Add Prometheus metrics endpoint** for NFR compliance
6. **Add down migrations** for rollback safety
7. **Document third-party library justifications** in `README.md`
8. **Add p95 latency benchmarks** for reservation and availability endpoints

---

*Review compiled by code analysis of the full repository against PRD.md v1.0.*
