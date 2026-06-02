# ParkirPintar — Requirement Clarification & Assumptions

This document captures requirement-by-requirement interpretation of the assessment blueprint, clarifying ambiguities, documenting design decisions, and declaring assumptions where the spec is silent or open to interpretation.

---

## 1. Vehicle Type — Per Driver vs Per Booking

**Blueprint says:**
> "the system immediately assigns any currently available spot that matches the vehicle type"

**Clarification:** The blueprint never mentions driver registration with a fixed vehicle. Vehicle type is implied to be chosen at reservation time.

**Decision:** Vehicle type is **chosen per booking**. A driver can reserve a car spot today and a motorcycle spot tomorrow.

**Implementation:** `drivers` table has only `id`, `name`, `phone`, `email`. Vehicle type lives on `reservations.vehicle_type` — strictly per-reservation.

---

## 2. Parking Capacity — 30 Cars + 50 Motorcycles × 5 Floors

**Blueprint says:**
> "The parking area is a parking building with 5 floors; each floor has capacity for 30 cars and 50 motorcycles (total capacity: 150 cars and 250 motorcycles)."

**Implementation:**

```
floor_number | vehicle_type | count
1            | car          | 30
1            | motorcycle   | 50
2            | car          | 30
2            | motorcycle   | 50
3            | car          | 30
3            | motorcycle   | 50
4            | car          | 30
4            | motorcycle   | 50
5            | car          | 30
5            | motorcycle   | 50
```

Total: 150 cars + 250 motorcycles = 400 spots. Matches blueprint exactly.

---

## 3. Spot Assignment Modes

**Blueprint says:**
> (1) System-assigned (fastest) — the system immediately assigns any currently available spot that matches the vehicle type
> (2) User-selected — the Driver can choose a specific spot, but the spot list may involve a short hold of time during payment accomplishment.

**Decision:**
- Both modes follow the **same payment flow**. The only difference is who picks the spot:
  - System-assigned: system picks → QRIS payment → confirmed
  - User-selected: driver picks from map → QRIS payment → confirmed
- **Spot state machine:** `available` → `waiting_payment` → `reserved/confirmed` (on payment success) OR back to `available` (on payment failure/timeout)
- **Hold timeout is configurable** (not hardcoded). If payment is not completed within the configured hold time, the spot reverts to `available` automatically.

**Implementation:**
- Default timeout: 10 minutes (configurable via `PAYMENT_TIMEOUT_MINUTES` env var)
- Implemented via Asynq delayed task (`task:payment:hold_timeout`)
- On timeout: `PaymentTimeoutHandler` calls `FailReservation` to release the spot
- On successful payment: timeout task is cancelled
- Spot state machine enforced: `available → waiting_payment → confirmed | available`

---

## 4. Reservation Hold & Expiry (No-Show)

**Blueprint says:**
> "A booking holds the assigned spot for 1 hour only. If the Driver does not check in/park within 1 hour after confirmation, the reservation expires automatically, and the spot becomes available for other Drivers to book."
> Booking fee of 5,000 IDR is charged on confirmation.

**Decision:**
- **Booking fee (5,000 IDR) is non-refundable.** Charged at the moment of confirmation, regardless of what happens next (cancel, no-show, or successful park).
- **1-hour check-in window:** After confirmation, the driver has exactly 1 hour to check in. If they fail to check in within that window, the reservation expires automatically and the spot reverts to `available`.
- **No separate cancellation fee.** The booking fee IS the cost. Driver can cancel anytime before check-in, but the 5k is already gone.
- **Expiry is automatic** via background task scheduler.

**Implementation:**
- At confirmation, a `task:reservation:expiry` Asynq task is enqueued with `ProcessIn(1 * time.Hour)`
- On expiry: reservation status updated to `expired`, spot released back to `available`
- Booking fee (5,000 IDR) charged at confirmation time via payment service

---

## 5. Billing & Pricing Rules

**Blueprint says:**
- First hour: 5,000 IDR
- Each subsequent started hour: 5,000 IDR
- Overnight fee (crosses midnight): flat 20,000 IDR
- Booking fee: 5,000 IDR per confirmed reservation
- No overstay penalty — additional time uses same hourly rate
- Rate calculated from actual session time, not locked upfront

**Decision:**
- **Booking fee (5,000 IDR) is separate from hourly charges.** Minimum cost for a successful park = 5,000 (booking) + 5,000 (first hour) = 10,000 IDR.
- **Overnight fee (20,000 IDR) is an ADDITIONAL fee**, not a replacement. It's charged on top of hourly rates if the driver is still parked (not checked out) when midnight (00:00) passes.
- **Overnight trigger:** Any session where the driver has NOT checked out before 00:00. Even if they check in at 23:59, if they're still parked at 00:00, the overnight fee applies.
- **Overnight is per-midnight-crossed.** A session spanning 2 midnights = 40,000 IDR overnight fee. Not a flat one-time charge.
- **No overstay penalty.** Time just keeps ticking at 5,000 IDR per started hour.
- **Billing is calculated at checkout** based on actual session duration (check-in to check-out time).

**Example:** Check-in 23:59, check-out 06:01 (7 started hours):
- Booking fee: 5,000 IDR (charged at confirmation)
- Hourly: 7 × 5,000 = 35,000 IDR
- Overnight: 20,000 IDR (1 midnight crossed)
- **Total: 60,000 IDR**

**Implementation:**
- `countMidnightsCrossed()` calculates calendar-day boundaries crossed in WIB (UTC+7)
- Overnight fee is `OvernightPerNight (20,000 IDR) × nightsCrossed`
- Hourly calculation uses started-hour logic (ceil)
- Booking fee charged at confirmation (separate payment moment), not bundled into checkout bill

---

## 6. Payment — QRIS Integration

**Blueprint says:**
> "The system must support checkout via payment gateway integration, including QRIS payments."

**Decision:**
- **QRIS only** for now. No other payment methods.
- **Two payment moments:**
  1. **At reservation confirmation** — 5,000 IDR booking fee (QRIS)
  2. **At checkout** — parking session charges (hourly + overnight if applicable) (QRIS)
- **Stub/mock is acceptable** for the assessment. No real payment gateway integration required.
- **Flow architecture (ready for real integration):**
  1. System generates QRIS code (stub: generates a fake QR image/string)
  2. Driver scans QR (simulated: auto-confirmed after short delay or manual trigger)
  3. Webhook confirms payment (simulated: internal callback that marks payment as success)

**Implementation:**
- Payment service exposes a `PaymentProvider` interface — stub implements it
- Webhook endpoint exists for payment confirmation (self-triggered in stub mode)
- Circuit breaker wraps payment client calls (threshold: 5, open timeout: 30s)
- Architecture is ready to swap in a real provider (Midtrans, Xendit) without structural code changes

---

## 7. Service-to-Service Communication

**Blueprint says:**
> "The system must use gRPC over HTTP/2 or Streaming event for service-to-service communication."

**Decision:**
- **gRPC over HTTP/2** for all synchronous service-to-service calls (gateway → reservation, reservation → billing, reservation → payment, etc.)
- **NATS JetStream** for asynchronous inter-service events (reservation events → search, analytics consumers)
- **Redis Asynq** for delayed/scheduled tasks (reservation expiry, payment hold timeout)

**Why both NATS and Asynq:** They serve different purposes:
- NATS JetStream handles event broadcasting (pub/sub) between services (e.g., reservation confirmed → search updates availability, analytics records event)
- Asynq handles delayed task execution (e.g., "release this spot in 10 minutes if payment fails," "expire this reservation in 1 hour if no check-in")
- Using the right tool for each job rather than forcing one tool to do both

---

## 8. Microservices Architecture

**Blueprint says:**
> "Suggested microservices (you may merge some if you justify it): gateway, search, reservation, billing, payment, presence, notification."

**Decision:**
- **7 services:** gateway, search, reservation, billing, payment, presence, analytics
- **Notification — removed.** Blueprint defines zero notification requirements (no push notifications, no emails, no SMS specified). Listed as "suggested" only.
- **Presence — kept.** Provides location-based check-in detection via smartphone geofencing. Enables automatic check-in when the driver arrives at the parking area — better UX than manual check-in.
- **Analytics — added.** Consumes NATS events to provide operational insights (occupancy trends, revenue reports). Demonstrates event-driven architecture capabilities.

**Service responsibilities:**
| Service | Role |
|---------|------|
| Gateway | REST API, JWT auth, gRPC fan-out to backend services |
| Search | Availability summary, floor maps, spot details |
| Reservation | Create/cancel/check-in/check-out, expiry scheduler (Asynq), spot locking |
| Billing | Pricing engine (hourly + overnight + booking fee), invoice generation |
| Payment | QRIS code generation, payment status, webhook handler (stub for assessment) |
| Presence | Location-based check-in detection, geofence events |
| Analytics | Event consumer, occupancy metrics, revenue reporting |

**Justification for notification removal:**
No notification delivery requirements are defined in the spec. The architecture supports adding pub/sub notifications later without structural changes (NATS JetStream already provides the event backbone).

---

## 9. Reusable Components

**Blueprint says:**
> "Provide reusable components: pricing engine (rules above), locking mechanism, config loader, and structured logging/tracing."

**Decision:** All four components exist as isolated, reusable modules.

| Component | Location | Description |
|-----------|----------|-------------|
| Pricing engine | `internal/billing/pricing/` | Self-contained package with clear interface (check-in time, check-out time → charge breakdown). Comprehensive unit tests. |
| Locking mechanism | `pkg/redislock/` | Redis-based distributed lock with configurable TTL and retry |
| Config loader | `pkg/config/` | Viper-based config with environment variable and file support |
| Structured logging | `pkg/grpcmiddleware/logging.go` | Structured JSON logging with request context |
| Structured tracing | `pkg/telemetry/` | OpenTelemetry integration with trace propagation |

**Note on pricing engine location:** Lives in `internal/billing/pricing/` rather than `pkg/pricing/`. This is intentional — only the billing service needs to calculate prices directly. Other services request billing via gRPC. The code itself is fully reusable (clean interface, no external deps, comprehensive tests), it simply lives within the billing service's domain boundary.

---

## 10. Idempotency

**Blueprint says:**
> "Use an idempotency mechanism for CreateReservation and Checkout/Invoice operations."

**Decision:**
- Idempotency enforced via two complementary mechanisms:
  1. **Database-level:** `UNIQUE` constraint on `reservations.idempotency_key` and `billing_records.idempotency_key` — prevents duplicate records at the data layer
  2. **Middleware-level:** `pkg/grpcmiddleware/idempotency.go` provides a Redis-based interceptor with atomic claim (SETNX), sentinel polling, and cached response replay — available for use
- The idempotency key is passed as a **protobuf request field** (not gRPC metadata header)
- Applied to: `CreateReservation`, `Checkout`/`CompleteCheckout` methods

**Design choice:** DB-level idempotency is the primary enforcement. The Redis middleware exists as reusable infrastructure but is not wired into the gRPC chain because:
- DB unique constraints provide the core guarantee (duplicate key → error, not double-write)
- Current behavior: duplicate idempotency key → returns conflict error to the client
- Alternative (Redis middleware): duplicate key → returns cached first response silently
- Both are valid idempotency patterns. DB-level was chosen for simplicity and correctness without additional infrastructure dependency in the hot path.

---

## 11. Consistency — Double-Booking Prevention

**Blueprint says:**
> "Consistency: prevent double-booking for the same spot and overlapping time windows."

**Decision:**
- **Two-layer locking:**
  1. Redis distributed lock (`pkg/redislock/`) — acquired per spot (`spot:<id>`) before reservation logic runs. Prevents concurrent requests from racing.
  2. Postgres `FOR UPDATE SKIP LOCKED` — row-level lock on the spot during the transaction. If another transaction holds it, skip and try next available.
- **DB constraint:** One active reservation per driver (prevents same driver double-booking).
- **Spot status check:** Only spots with `status = 'available'` can be reserved.

**Assumption on "overlapping time windows":** Since the system is real-time only (no advance booking for future time slots), overlap detection is implicitly handled by spot status. A spot is either `available`, `waiting_payment`, `reserved`, or `occupied` — it cannot be double-booked because only `available` spots can transition to `waiting_payment`. There is no concept of "book spot X from 2pm to 4pm tomorrow."

---

## 12. Availability & Resilience

**Blueprint says:**
> "Define how you handle retries, timeouts, circuit breakers, and graceful degradation when non-core services fail."

**Decision:**
- **Retries:** gRPC client with configurable retry policy (`pkg/grpcclient/`)
- **Timeouts:** Per-RPC deadlines on all gRPC calls
- **Circuit breakers:** `pkg/circuitbreaker/` — wraps calls to downstream services, opens circuit on repeated failures
- **Graceful degradation:** If payment service is down, reservation can still be created (enters `waiting_payment` state). If billing is down, checkout returns error but reservation stays intact.
- **Graceful shutdown:** `pkg/server/` handles OS signals, drains connections before exit

**Core vs non-core services:**
| Service | Classification | Degradation behavior |
|---------|---------------|---------------------|
| Reservation | Core | Must be available for any booking flow |
| Search | Core | Must be available for spot discovery |
| Billing | Core | Required for checkout; checkout blocked if down |
| Payment | Non-core | Circuit breaker (threshold: 5, timeout: 30s). Reservation created in `waiting_payment`; payment retried when service recovers |
| Presence | Non-core | Check-in falls back to manual if geofencing unavailable |
| Analytics | Non-core | Event loss is acceptable; NATS provides replay via JetStream |

---

## 13. Testing Requirements

**Blueprint says:**
- Unit tests: pricing rules, overlap detection, idempotency
- Integration tests: reservation → billing flow
- E2E scenarios: happy path, double-book, contention, expiry, wrong-spot, cancellation, extended stay, overnight, payment success/failure

**Decision on wrong-spot penalty:** NOT IN SCOPE. No mechanism exists to detect which physical spot a driver actually parks in (no sensors, no camera system). The system trusts that drivers park in their assigned spot. Test file exists as a documented stub declaring this assumption.

**Test coverage:**

| Scenario | Test File | Notes |
|----------|-----------|-------|
| Happy path | `tests/e2e/happy_path_test.go` | Full flow: reserve → pay → check-in → check-out → billing |
| Double-book prevention | `tests/e2e/double_book_test.go` | Concurrent reservation attempts for same spot |
| User-selected contention | `tests/e2e/contention_test.go` | Multiple drivers selecting same spot simultaneously |
| Reservation expiry (no-show) | `tests/e2e/expiry_test.go` | 1-hour timeout triggers automatic spot release |
| Wrong-spot penalty | `tests/e2e/wrong_spot_test.go` | Stub — documents out-of-scope assumption |
| Cancellation policy | `tests/e2e/free_cancel_test.go`, `paid_cancel_test.go` | Booking fee retained on cancel |
| Extended stay billing | `tests/e2e/extended_stay_test.go` | No overstay penalty, same hourly rate |
| Overnight fee | `tests/e2e/overnight_test.go` | Per-midnight-crossed calculation |
| Payment success + failure | `tests/e2e/payment_success_test.go`, `payment_failure_test.go` | QRIS stub success/timeout/failure |

**Extras beyond spec:** pricing property tests, state machine tests, race condition tests, load tests (k6), contract tests, chaos tests (toxiproxy).

---

## 14. Presence / Location

**Blueprint says:**
> "The system uses location data from smartphones to detect where the Driver is"

**Decision:** Kept in scope. The presence service provides location-based check-in detection via smartphone geofencing. Even with a single parking area, it enables automatic check-in when a driver arrives — better UX than requiring a manual check-in button press.

**Implementation:**
- Geofence boundary defined for the parking building
- When a driver with an active reservation enters the geofence, check-in is triggered automatically
- Fallback: manual check-in via API if location services are unavailable

---

## 15. Single Parking Area — No Multi-Area

**Blueprint says:**
> "The system manages a single parking area (one district/area only) with a centralized inventory; there is no Host onboarding or spot publishing."
> "Driver will view availability and reserve a spot within that parking area (no multi-area search radius)."

**Decision:**
- Single parking building, hardcoded. No area selection, no search radius.
- No "host" concept — no landlord/operator onboarding flow.
- Centralized inventory = one Postgres DB, one set of 400 spots.
- Simplifies everything: no multi-tenancy, no area routing, no geo-queries for discovery.

---

## 16. Mini App / Super App Context

**Blueprint says:**
> "This solution will be set up as a mini app inside a super app (or as a standalone service)."

**Decision:**
- ParkirPintar is a backend service consumed by a super app's frontend.
- Authentication is delegated to the super app — ParkirPintar only validates JWT tokens (BYO-JWT).
- No user registration, login, or password management in our scope.
- The super app provisions driver records and issues JWTs.

**Implementation:** Gateway validates JWT tokens but does not issue them. Driver records are assumed to exist (seeded or created by the parent app).

---

## 17. Documentation Requirements

**Blueprint says:**
- Solution diagrams in README.md:
  - High-level design architecture
  - Low-level design architecture
  - ERD document
- Configuration files for each tool (queuing, load balancer, cloud system) committed in repo
- 3rd party library list with justification
- All assumptions declared in README.md

**Implementation:**
- Architecture diagrams: Mermaid in README.md (high-level) + `docs/architecture/` (detailed)
- ERD: `docs/architecture/er-diagram.md`
- Sequence diagrams: `docs/architecture/sequence-diagrams.md`
- Config files committed: `deploy/local/docker-compose.yml`, Traefik config, Grafana Alloy config, Kubernetes manifests
- Assumptions declared in README.md (sourced from this document)

---

## Assumptions

These are interpretive decisions made where the blueprint is silent or ambiguous:

1. **Vehicle type is per-booking, not per-driver.** A driver can reserve car or motorcycle spots on different occasions. No vehicle registration required.

2. **No advance booking.** Reservations are real-time only — "I need a spot now." No scheduling for future dates/times. This means "overlapping time windows" is handled implicitly by spot status (available/reserved/occupied).

3. **Wrong-spot penalty is not implementable** without physical sensors or camera systems. The system trusts that drivers park in their assigned spot.

4. **Single parking area.** No multi-tenancy, no area discovery, no geo-queries. One building, 5 floors, 400 spots.

5. **No notification delivery.** The spec lists notification as a suggested service but defines no notification requirements (no push, email, or SMS). Architecture supports adding it later via NATS event consumers.

6. **Payment is stubbed.** QRIS flow is architecturally complete (interface, webhook handler, QR generation) but backed by a mock provider. Ready for real integration (Midtrans/Xendit) by implementing the `PaymentProvider` interface.

7. **Overnight fee is per-midnight-crossed.** 2 nights = 40,000 IDR. Not a flat one-time fee regardless of duration.

8. **Booking fee is non-refundable.** Charged at confirmation regardless of outcome (cancel, no-show, or successful park).

9. **Driver always checks out.** No auto-checkout mechanism for abandoned vehicles. If needed later, a max-session-duration rule can be added.

10. **BYO-JWT authentication.** No user registration/login flow. Drivers are pre-provisioned with JWT tokens (suitable for mini-app inside super-app where auth is handled by the parent app).

11. **NATS JetStream + Asynq coexist.** NATS handles inter-service event broadcasting (pub/sub); Asynq handles delayed/scheduled task execution. Each tool serves its intended purpose — no overlap.

12. **Idempotency via DB constraints.** Duplicate idempotency keys return a conflict error rather than a cached replay of the original response. Both are valid patterns; DB-level was chosen for simplicity.

13. **Payment hold timeout is 10 minutes by default.** The spec says "a short hold of time during payment accomplishment" — 10 minutes is reasonable for QRIS scanning. Configurable via environment variable.

14. **Reservation expiry is exactly 1 hour.** Measured from confirmation time (payment success), not from reservation creation (payment initiation).
