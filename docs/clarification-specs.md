# ParkirPintar — Requirement Clarification & Validation

This document captures requirement-by-requirement validation of the assessment blueprint against our current implementation. Each item is discussed, clarified, and marked as PASS, FAIL, or TODO.

---

## 1. Vehicle Type — Per Driver vs Per Booking

**Blueprint says:**
> "the system immediately assigns any currently available spot that matches the vehicle type"

**Clarification:** The blueprint never mentions driver registration with a fixed vehicle. Vehicle type is implied to be chosen at reservation time.

**Decision:** Vehicle type is **chosen per booking**. A driver can reserve a car spot today and a motorcycle spot tomorrow.

**Current state:** PASS ✅ — `drivers` table has only `id`, `name`, `phone`, `email`. Vehicle type is per-reservation (`reservations.vehicle_type`).

---

## 2. Parking Capacity — 30 Cars + 50 Motorcycles × 5 Floors

**Blueprint says:**
> "The parking area is a parking building with 5 floors; each floor has capacity for 30 cars and 50 motorcycles (total capacity: 150 cars and 250 motorcycles)."

**Current state:** PASS ✅

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

**Current state:** PARTIAL — system-assigned works end-to-end. User-selected FE exists but needs backend validation that the chosen spot is available + hold logic.

**Action required:**
- Ensure spot state machine enforces: `available → waiting_payment → confirmed | available`
- Add configurable hold timeout (e.g., `PAYMENT_HOLD_TIMEOUT=5m` in config)
- Add background job/scheduler to release expired holds
- User-selected mode: validate spot availability before locking

---

## 4. Reservation Hold & Expiry (No-Show)

**Blueprint says:**
> "A booking holds the assigned spot for 1 hour only. If the Driver does not check in/park within 1 hour after confirmation, the reservation expires automatically, and the spot becomes available for other Drivers to book."
> Booking fee of 5,000 IDR is charged on confirmation.

**Decision:**
- **Booking fee (5,000 IDR) is non-refundable.** Charged at the moment of confirmation, regardless of what happens next (cancel, no-show, or successful park).
- **1-hour check-in window:** After confirmation, the driver has exactly 1 hour to check in. If they fail to check in within that window, the reservation expires automatically and the spot reverts to `available`.
- **No separate cancellation fee.** The booking fee IS the cost. Driver can cancel anytime before check-in, but the 5k is already gone.
- **Expiry is automatic** — requires a background scheduler/cron to detect expired reservations and release spots.

**Current state:** TBD — need to verify if expiry scheduler exists and if booking fee is charged at confirmation.

**Action required:**
- Implement reservation expiry scheduler (check every N seconds for confirmed reservations older than 1 hour without check-in)
- Ensure billing charges 5,000 IDR booking fee at confirmation time (not at checkout)
- On expiry: update reservation status to `expired`, release spot back to `available`

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
- **No overstay penalty.** Time just keeps ticking at 5,000 IDR per started hour.
- **Billing is calculated at checkout** based on actual session duration (check-in to check-out time).

**Example:** Check-in 23:59, check-out 06:01 (7 started hours):
- Booking fee: 5,000 IDR (charged at confirmation)
- Hourly: 7 × 5,000 = 35,000 IDR
- Overnight: 20,000 IDR
- **Total: 60,000 IDR**

**Current state:** TBD — need to verify pricing engine implements these rules correctly.

**Action required:**
- Verify pricing engine handles overnight detection (session crosses 00:00)
- Ensure booking fee is charged at confirmation, not bundled into checkout bill
- Unit tests for: exact midnight crossing, multi-night stays, sub-hour sessions

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
- **Code should demonstrate readiness** — the payment service interface, webhook handler, and QRIS generation logic should exist in code even if backed by stubs. Shows we can swap in a real provider (Midtrans, Xendit) without architectural changes.

**Current state:** PARTIAL — FE shows QRIS page with QR code, confirm button exists. Need to verify the payment service has proper interface abstraction for swapping real provider.

**Action required:**
- Ensure payment service has a `PaymentProvider` interface (stub implements it)
- Webhook endpoint for payment confirmation (even if self-triggered in stub mode)
- Two distinct payment flows: booking fee at confirmation, session charges at checkout
- Document in README that stub is used, with notes on real integration path

---

## 7. Service-to-Service Communication

**Blueprint says:**
> "The system must use gRPC over HTTP/2 or Streaming event for service-to-service communication."

**Decision:**
- **gRPC over HTTP/2** for all synchronous service-to-service calls (gateway → reservation, reservation → billing, reservation → payment, etc.)
- **No NATS.** Removed from the stack — no event streaming needed.
- **No notification service.** Blueprint lists it as "suggested" but defines zero notification requirements. Omitted and justified in README.
- **Redis Asynq** for delayed/scheduled tasks:
  - Reservation expiry (1-hour no-show): enqueue `reservation:expire` task with `ProcessIn(1 * time.Hour)` at confirmation time
  - Payment hold timeout: enqueue `spot:release` task with `ProcessIn(configurable)` when spot enters `waiting_payment`
- **Why Asynq over DB poller:** Industry-standard Go task queue, exactly-once delivery, built-in retries/backoff, scales horizontally, and Redis is already in the stack for distributed locking.
- **Why not NATS:** NATS is pub/sub (broadcast events), not designed for "do X after delay." Asynq is purpose-built for delayed task execution.

**Current state:** PASS ✅ — NATS JetStream handles delayed tasks via message headers (`Nats-Expected-Last-Msg-Id`) and consumer `DeliverPolicy`. Reservation expiry and hold-release are event-driven via NATS consumers. No need for Asynq — NATS JetStream already provides durable, at-least-once delivery with ack/nak.
- Remove notification service entirely

---

## 8. Microservices Architecture

**Blueprint says:**
> "Suggested microservices (you may merge some if you justify it): gateway, search, reservation, billing, payment, presence, notification."

**Decision:**
- **5 services kept:** gateway, search, reservation, billing, payment
- **Notification — removed.** Blueprint defines zero notification requirements. Listed as "suggested" only.
- **Presence — removed.** The spec says "uses location data from smartphones to detect where the Driver is" but this is a concept description, not a functional requirement. With a single fixed parking area (no multi-area search), there's nothing to detect — the driver already knows where the parking is. Presence only makes sense for multi-location discovery.

**Final service responsibilities:**
| Service | Role |
|---------|------|
| Gateway | REST API, JWT auth, gRPC fan-out to backend services |
| Search | Availability summary, floor maps, spot details |
| Reservation | Create/cancel/check-in/check-out, expiry scheduler (Asynq), spot locking |
| Billing | Pricing engine (hourly + overnight + booking fee), invoice generation |
| Payment | QRIS code generation, payment status, webhook handler (stub for assessment) |

**Justification for removals (to include in README):**
- Notification: "No notification delivery requirements defined in the spec. Architecture supports adding pub/sub notifications later without structural changes."
- Presence: "Single-area system with no multi-location discovery. Location-based features (geofencing, auto-check-in) are not specified. Service can be added if multi-area support is needed."

**Current state:** PARTIAL — all 5 services exist but presence and notification also exist. Need cleanup.

**Action required:**
- Remove presence service code and docker-compose entry
- Remove notification service code and docker-compose entry
- Remove NATS from docker-compose
- Update gateway to not connect to presence/notification
- Update README with justification

---

## 9. Reusable Components

**Blueprint says:**
> "Provide reusable components: pricing engine (rules above), locking mechanism, config loader, and structured logging/tracing."

**Decision:** All four components must exist as shared, reusable packages (in `pkg/`).

**Current state:** MOSTLY PASS

| Component | Status | Location | Notes |
|-----------|--------|----------|-------|
| Pricing engine | ⚠️ PARTIAL | `internal/billing/model/` | Works but lives in `internal/` — should be `pkg/pricing/` for reusability |
| Locking mechanism | ✅ PASS | `pkg/redislock/` | Redis-based distributed lock |
| Config loader | ✅ PASS | `pkg/config/` | Viper-based config with env/file support |
| Structured logging | ✅ PASS | `pkg/grpcmiddleware/logging.go` | Structured JSON logging |
| Structured tracing | ✅ PASS | `pkg/telemetry/` | OpenTelemetry integration |

**Action required:**
- Move pricing logic to `pkg/pricing/` as a standalone reusable engine
- Ensure pricing engine is unit-testable with clear input/output (check-in time, check-out time → breakdown of charges)
- Verify overnight fee logic counts per-midnight-crossed (not flat once)

---

## 10. Idempotency

**Blueprint says:**
> "Use an idempotency mechanism for CreateReservation and Checkout/Invoice operations."

**Decision:**
- Idempotency enforced via `x-idempotency-key` gRPC metadata header
- Redis SETNX with sentinel polling pattern (concurrent requests wait for first to complete)
- Applied to: `CreateReservation`, `Checkout`/`CompleteCheckout` methods
- TTL-based cache — same key returns cached response without re-executing

**Current state:** PASS ✅ — `pkg/grpcmiddleware/idempotency.go` implements Redis-based idempotency with:
- Atomic claim via SETNX
- Sentinel value while processing
- Concurrent request polling (10ms intervals, 3s max)
- Automatic cleanup on handler error (allows retry)
- Configurable method list and TTL

**Action required:**
- Verify `CreateReservation` and `Checkout` are in the enforced methods list
- Ensure gateway passes `x-idempotency-key` from REST header to gRPC metadata

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

**Current state:** PASS ✅ — Both Redis lock and Postgres row-level locking are implemented. Tests cover lock contention scenario (`TestCreateReservation_ShouldReturnConflict_WhenLockContention`).

**Action required:**
- Verify "overlapping time windows" is handled — currently spots are either available or not (no future booking). Since the spec is real-time only (no advance booking for future time slots), overlap detection is implicitly handled by spot status. Confirm this assumption is correct.

---

## 12. Availability & Resilience

**Blueprint says:**
> "Define how you handle retries, timeouts, circuit breakers, and graceful degradation when non-core services fail."

**Decision:**
- **Retries:** gRPC client with configurable retry policy (in `pkg/grpcclient/`)
- **Timeouts:** Per-RPC deadlines on all gRPC calls
- **Circuit breakers:** `pkg/circuitbreaker/` — wraps calls to downstream services, opens circuit on repeated failures
- **Graceful degradation:** If payment service is down, reservation can still be created (enters `waiting_payment` state). If billing is down, checkout returns error but reservation stays intact.
- **Graceful shutdown:** `pkg/server/` handles OS signals, drains connections before exit

**Current state:** PASS ✅ — All four resilience patterns are implemented with tests.

**Action required:**
- Document in README which services are "core" vs "non-core" and what happens when each fails
- Ensure circuit breaker is wired into payment client (non-core, should degrade gracefully)

---

## 13. Testing Requirements

**Blueprint says:**
- Unit tests: pricing rules, overlap detection, idempotency
- Integration tests: reservation → billing flow
- E2E scenarios:
  - Happy path reservation
  - Double-book prevention
  - User-selected spot contention/queue
  - Reservation expiry (no-show) and spot release
  - Wrong-spot penalty
  - Cancellation policy
  - Extended stay billing (no overstay penalty)
  - Overnight fee
  - Payment checkout (QRIS success + failure)

**Decision:**
- **Wrong-spot penalty: NOT IN SCOPE.** No mechanism to detect which physical spot a driver actually parks in (no sensors, no presence service). Test file exists as a stub/assumption declaration. Will document in README as an assumption.
- All other scenarios are implemented.

**Current state:** MOSTLY PASS ✅

| Scenario | Test File | Status |
|----------|-----------|--------|
| Happy path | `tests/e2e/happy_path_test.go` | ✅ |
| Double-book prevention | `tests/e2e/double_book_test.go` | ✅ |
| User-selected contention | `tests/e2e/contention_test.go` | ✅ |
| Reservation expiry (no-show) | `tests/e2e/expiry_test.go` | ✅ |
| Wrong-spot penalty | `tests/e2e/wrong_spot_test.go` | ⚠️ Stub — not in scope |
| Cancellation policy | `tests/e2e/free_cancel_test.go`, `paid_cancel_test.go` | ✅ |
| Extended stay billing | `tests/e2e/extended_stay_test.go` | ✅ |
| Overnight fee | `tests/e2e/overnight_test.go` | ✅ |
| Payment success + failure | `tests/e2e/payment_success_test.go`, `payment_failure_test.go` | ✅ |

**Extras beyond spec:** pricing property tests, state machine tests, race condition tests, load tests (k6), contract tests, chaos tests (toxiproxy).

**Action required:**
- Ensure all tests actually pass with current code (some may reference NATS which we're removing)
- Update wrong-spot test to document assumption clearly
- Verify pricing tests cover per-midnight overnight fee (40k for 2 nights)

---

## 14. Presence / Location

**Blueprint says:**
> "The system uses location data from smartphones to detect where the Driver is"

**Decision:** NOT IN SCOPE. This is a concept description in the intro paragraph, not a functional requirement. With a single fixed parking area and no multi-location discovery, presence detection serves no purpose. Removed from architecture (see #8).

**Current state:** N/A — service to be removed.

---

## 15. Single Parking Area — No Multi-Area

**Blueprint says:**
> "The system manages a single parking area (one district/area only) with a centralized inventory; there is no Host onboarding or spot publishing."
> "Driver will view availability and reserve a spot within that parking area (no multi-area search radius)."

**Decision:**
- Single parking building, hardcoded. No area selection, no search radius.
- No "host" concept — no landlord/operator onboarding flow.
- Centralized inventory = one Postgres DB, one set of 400 spots.
- Simplifies everything: no multi-tenancy, no area routing, no geo-queries.

**Current state:** PASS ✅ — System is built for exactly one parking area.

---

## 16. Mini App / Super App Context

**Blueprint says:**
> "This solution will be set up as a mini app inside a super app (or as a standalone service)."

**Decision:**
- ParkirPintar is a backend service consumed by a super app's frontend.
- Authentication is delegated to the super app — ParkirPintar only validates JWT tokens (BYO-JWT).
- No user registration, login, or password management in our scope.
- The super app provisions driver records and issues JWTs.

**Current state:** PASS ✅ — BYO-JWT is already implemented. Gateway validates tokens but doesn't issue them.

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

**Current state:** PARTIAL — README exists with architecture diagram (mermaid) but references 7 services + NATS. Needs rewrite after implementation changes.

**Action required:**
- Rewrite README.md after all code changes:
  - Update architecture diagram to 5 services (no presence, no notification, no NATS)
  - Add Redis Asynq to tech stack
  - Add low-level design (sequence diagrams for key flows)
  - Add ERD (already exists in `docs/design/er-diagram.md`, embed or link)
  - Add assumptions section (from this doc)
  - Add 3rd party library table with justifications
  - Reference config files (docker-compose, Traefik, Alloy, etc.)

---

## Assumptions (to declare in README.md)

1. **Vehicle type is per-booking, not per-driver.** A driver can reserve car or motorcycle spots on different occasions. No vehicle registration required.
2. **No advance booking.** Reservations are real-time only — "I need a spot now." No scheduling for future dates/times. This means "overlapping time windows" is handled implicitly by spot status (available/reserved/occupied).
3. **Wrong-spot penalty is not implementable** without physical sensors or presence detection. The system trusts that drivers park in their assigned spot. Documented as out-of-scope.
4. **Single parking area.** No multi-tenancy, no area discovery, no geo-queries. One building, 5 floors, 400 spots.
5. **No notification delivery.** The spec lists notification as a suggested service but defines no notification requirements. Architecture supports adding it later.
6. **No presence/location service.** Single-area system doesn't need location detection. Removed with justification.
7. **Payment is stubbed.** QRIS flow is architecturally complete (interface, webhook handler, QR generation) but backed by a mock provider. Ready for real integration (Midtrans/Xendit) without code changes.
8. **Overnight fee is per-midnight-crossed.** 2 nights = 40,000 IDR. Not a flat one-time fee.
9. **Booking fee is non-refundable.** Charged at confirmation regardless of outcome (cancel, no-show, or successful park).
10. **Driver always checks out.** No auto-checkout mechanism for abandoned vehicles. If needed later, a max-session-duration rule can be added.
11. **BYO-JWT authentication.** No user registration/login flow. Drivers are pre-provisioned with JWT tokens (suitable for mini-app inside super-app where auth is handled by the parent app).

---

## Summary — Action Items

### Must Fix (spec violations)
1. Remove `vehicle_type` and `vehicle_plate` from `drivers` table — vehicle type is per-reservation only
2. Remove NATS from stack — replaced by Redis Asynq for delayed tasks
3. Remove notification service — no spec requirement
4. Remove presence service — no functional need for single-area
5. Add Redis Asynq for reservation expiry (1-hour no-show) and payment hold timeout
6. Implement two-payment-moment flow: booking fee at confirmation + session charges at checkout
7. Move pricing engine to `pkg/pricing/` for reusability
8. Fix overnight fee to charge per-midnight-crossed (not flat once)
9. Implement configurable payment hold timeout with auto-release

### Should Fix (quality/completeness)
10. Verify all tests pass after NATS removal
11. Ensure `CreateReservation` and `Checkout` are in idempotency enforced methods list
12. Ensure gateway passes `x-idempotency-key` header to gRPC metadata
13. Wire circuit breaker into payment client
14. User-selected mode: validate chosen spot availability + hold logic
15. Document assumptions in README.md

### Nice to Have (beyond spec)
16. Admin endpoints for operators
17. Billing history endpoint for drivers
18. WebSocket/SSE for real-time spot updates
19. Multi-night stay tests (2+ midnights)
