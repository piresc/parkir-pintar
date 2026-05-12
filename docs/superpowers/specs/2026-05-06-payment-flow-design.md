# Payment-Before-Confirmation Reservation Flow

**Date:** 2026-05-06
**Status:** Draft
**Author:** ParkirPintar Team
**PRD Reference:** Gap Analysis — Reservation hold during payment (§8), Booking fee at confirmation (§9.1)

---

## 1. Problem

**Current behavior:** `CreateReservation` creates the reservation directly with `status=confirmed`, then calls `Billing.StartBilling` as a non-critical operation (graceful degradation). If billing fails, the reservation is still marked confirmed but the 5,000 IDR booking fee is never recorded.

**Gap 1 (User-selected hold):** The assessment requires that for user-selected spots, the spot be held *during payment accomplishment*, not just during DB insertion. If payment fails or times out, the spot should be released without creating a confirmed reservation.

**Gap 2 (Payment-before-confirmation):** The assessment requires a `waiting_payment` status. Reservation only transitions to `confirmed` after payment succeeds. If payment fails or times out after X minutes, the reservation transitions to `failed` and the spot is released.

Both modes (system-assigned and user-selected) must go through this payment-before-confirmation flow.

---

## 2. Design

### 2.1 Target Flow

```
Driver clicks "Reserve"
  → Idempotency check (return existing if duplicate)
  → Find available spot (system-assigned) or validate spot_id (user-selected)
  → Acquire Redis distributed lock (TTL = payment_timeout + 2min)
  → Double-check spot availability under lock
  → DB transaction:
      - INSERT reservation (status=waiting_payment, spot_id=spot)
      - UPDATE spot status='reserved'
  → Billing.StartBilling (create billing record with booking_fee=5000)
  → Payment.ProcessPayment (processing_fee=5000, method=qris)
  → ┌─ SUCCESS ──────────────────────────────────────────┐
    │   UPDATE reservation (status=confirmed,              │
    │     confirmed_at=now, expires_at=now+1h)              │
    │   Publish reservation.confirmed event                 │
    │   Release Redis lock                                  │
    └──────────────────────────────────────────────────────┘
  → ┌─ FAILURE ──────────────────────────────────────────┐
    │   DB transaction:                                    │
    │     UPDATE reservation (status=failed)                │
    │     UPDATE spot (status=available)                    │
    │   Publish reservation.payment_failed event            │
    │   Release Redis lock                                  │
    └──────────────────────────────────────────────────────┘
```

### 2.2 Background Worker

```
Every 30 seconds:
  SELECT * FROM reservations
  WHERE status = 'waiting_payment'
    AND created_at < NOW() - INTERVAL '$timeout minutes'
  → For each: FailReservation(id)
      → Transaction: FOR UPDATE → validate → status=failed → spot=available
      → Publish reservation.payment_failed event
```

### 2.3 State Machine

```
                    ┌─→ confirmed ──→ checked_in ──→ checked_out
waiting_payment ───┤                  (1h expiry)
                    │                      ↓
                    │                   expired
                    │                  
                    ├─→ failed (payment failure / timeout, spot released)
                    │
                    └─→ cancelled (driver cancels before payment completes)
```

### 2.4 State Transitions Table

| From | To | Trigger |
|------|----|---------|
| (none) | `waiting_payment` | CreateReservation lock + insert |
| `waiting_payment` | `confirmed` | Payment gateway success |
| `waiting_payment` | `failed` | Payment gateway failure or timeout |
| `waiting_payment` | `cancelled` | Driver cancels while payment pending |
| `confirmed` | `checked_in` | Driver arrives, CheckIn RPC |
| `confirmed` | `expired` | No check-in within 1 hour |
| `confirmed` | `cancelled` | Driver cancels before check-in |
| `checked_in` | `checked_out` | CheckOut RPC (billing + payment) |

### 2.5 Cancellation During Payment

If a driver cancels while in `waiting_payment` status:
- No cancellation fee (the booking fee was never successfully charged)
- Spot is released immediately
- Reservation transitions to `cancelled`

---

## 3. Data Model Changes

### 3.1 New Status Values

```go
StatusWaitingPayment = "waiting_payment"
StatusFailed         = "failed"
```

### 3.2 Migration (`000003_payment_flow.up.sql`)

```sql
-- 1. Expand status CHECK constraint
ALTER TABLE reservations DROP CONSTRAINT reservations_status_check;
ALTER TABLE reservations ADD CONSTRAINT reservations_status_check
  CHECK (status IN (
    'pending', 'waiting_payment', 'confirmed', 'checked_in',
    'checked_out', 'expired', 'cancelled', 'failed'
  ));

-- 2. Include waiting_payment in double-booking partial unique index
DROP INDEX idx_reservations_active_spot;
CREATE UNIQUE INDEX idx_reservations_active_spot
  ON reservations (spot_id)
  WHERE status IN ('waiting_payment', 'confirmed', 'checked_in');

-- 3. Worker scan index for payment timeouts
CREATE INDEX idx_reservations_stale_payment
  ON reservations (status, created_at)
  WHERE status = 'waiting_payment';
```

### 3.3 Config

```go
// New: ReservationConfig
type ReservationConfig struct {
    PaymentTimeoutMinutes int // default 10
}
```

Environment variable: `PAYMENT_TIMEOUT_MINUTES=10`

Loaded in `pkg/config/config.go` alongside existing config. Default 10 minutes.

### 3.4 Redis Lock TTL

Lock key: `lock:spot:{spot_id}`
TTL: `(payment_timeout_minutes + 2) * 60` seconds (buffer for network latency)

The lock is released explicitly on success/failure. The TTL is a safety net for crash recovery.

---

## 4. Repository Changes

### 4.1 Reservation Repository

New method:
```go
FindStalePaymentReservations(ctx context.Context, timeoutMinutes int) ([]*model.Reservation, error)
```
SQL: `SELECT * FROM reservations WHERE status = 'waiting_payment' AND created_at < NOW() - make_interval(mins => $1)`

No other repository changes needed. Existing `GetByIDForUpdate`, `UpdateReservationTx`, `UpdateSpotStatusTx` are sufficient.

### 4.2 Billing Repository

No changes. `StartBilling` (creates billing record with booking_fee) and `GetByReservationID` already exist.

### 4.3 Payment Repository

No changes. `ProcessPayment` with idempotency key already exists. The reservation usecase passes a payment-specific idempotency key (`"booking-payment-{reservation_id}"`).

---

## 5. Usecase Interface Changes

### 5.1 CreateReservation (rewrite)

**Before:**
- Lock → validate → insert (confirmed) → StartBilling (fire-and-forget) → publish → return

**After:**
- Lock (extended TTL) → validate → insert (waiting_payment) → StartBilling → ProcessPayment → on success: update to confirmed + set expiry → publish → release lock
- On failure: transaction to failed + spot released → publish → release lock

**Criticality shift:** `Billing.StartBilling` and `Payment.ProcessPayment` become **critical** — if either fails, the reservation is not confirmed.

### 5.2 New: FailReservation

```go
FailReservation(ctx context.Context, req *model.FailReservationRequest) error
```
Used by the payment timeout worker. Validates transition from `waiting_payment` to `failed`, releases spot in transaction.

### 5.3 CancelReservation (update)

Allow cancellation from `waiting_payment` status. No cancellation fee applies (booking fee was never charged).

### 5.4 BillingClient interface (unchanged)

```go
StartBilling(ctx, reservationID, bookingFee, idempotencyKey) error
```

### 5.5 PaymentClient interface (unchanged)

```go
ProcessPayment(ctx, billingID, amount, paymentMethod, idempotencyKey) (string, error)
```

---

## 6. Worker Architecture

Two independent background workers in `cmd/reservation/main.go`:

| Worker | Query Filter | Action | Interval |
|--------|-------------|--------|----------|
| Expiry | `status='confirmed' AND expires_at < NOW()` | `ExpireReservation` | 30s |
| PaymentTimeout | `status='waiting_payment' AND created_at < NOW() - timeout` | `FailReservation` | 30s |

Both workers use `SELECT FOR UPDATE` + transaction to prevent TOCTOU races.

---

## 7. Test Plan

### 7.1 Unit Tests (`internal/reservation/usecase/usecase_test.go`)

| Test | Scenario |
|------|----------|
| `TestCreateReservation_PaymentSuccess` | Payment succeeds → reservation=confirmed, confirmed_at/expires_at set, spot=reserved |
| `TestCreateReservation_PaymentFailure` | Payment fails → reservation=failed, spot=available |
| `TestCreateReservation_PaymentTimeout` | Background worker detects stale waiting_payment → fail, spot released |
| `TestCreateReservation_IdempotencyWaitingPayment` | Duplicate key returns existing waiting_payment (not a new one) |
| `TestCreateReservation_IdempotencyConfirmed` | Duplicate key returns already-confirmed reservation |
| `TestCancelReservation_FromWaitingPayment` | Cancel during payment → cancelled, spot released, no fee |
| `TestFailReservation_ValidTransition` | Fail from waiting_payment → status=failed, spot=available |
| `TestFailReservation_InvalidTransition` | Fail from confirmed → error |

### 7.2 Integration Tests

| Test | Scenario |
|------|----------|
| `TestReservationPaymentFlow` | Create → waiting_payment → payment → confirmed → checkin → checkout → billing |

### 7.3 E2E Tests (new/updated)

| File | Change |
|------|--------|
| `tests/e2e/happy_path_test.go` | Update to account for waiting_payment intermediate state |
| `tests/e2e/double_book_test.go` | Add: second driver cannot reserve same spot while payment pending |
| `tests/e2e/contention_test.go` | Add: user-selected spot held during payment, contention resolved |
| `tests/e2e/payment_failure_test.go` | Update: simulate payment gateway failure, verify spot released |
| `tests/e2e/expiry_test.go` | Add: payment timeout (differs from 1h expiry) |
| New: `tests/e2e/payment_before_confirm_test.go` | Full E2E: create → waiting_payment → pay → confirmed |

---

## 8. File Checklist

| # | File | Action |
|---|------|--------|
| 1 | `internal/reservation/model/model.go` | Add `StatusWaitingPayment`, `StatusFailed`, update transitions |
| 2 | `db/migrations/000003_payment_flow.up.sql` | New migration |
| 3 | `pkg/config/config.go` | Add `ReservationConfig` struct + `PaymentTimeoutMinutes` |
| 4 | `config/.env.example` | Add `PAYMENT_TIMEOUT_MINUTES` |
| 5 | `config/.env` | Add `PAYMENT_TIMEOUT_MINUTES` |
| 6 | `docker-compose.yml` | Add `PAYMENT_TIMEOUT_MINUTES` to reservation service env |
| 7 | `internal/reservation/repository/repository.go` | Add `FindStalePaymentReservations` |
| 8 | `internal/reservation/usecase/usecase.go` | Rewrite `CreateReservation`, add `FailReservation`, update `CancelReservation` |
| 9 | `internal/reservation/worker/expiry.go` | Add `RunPaymentTimeoutWorker` |
| 10 | `cmd/reservation/main.go` | Start payment timeout worker |
| 11 | `internal/reservation/model/model.go` | Add `FailReservationRequest` struct + `transition.go` content |
| 12 | `internal/reservation/usecase/usecase_test.go` | Add/update unit tests |
| 13 | `tests/e2e/payment_before_confirm_test.go` | New E2E test |
| 14 | `tests/e2e/happy_path_test.go` | Update for waiting_payment intermediate state |
| 15 | `tests/e2e/double_book_test.go` | Update for payment-hold double-book prevention |
| 16 | `tests/e2e/contention_test.go` | Update for user-selected payment contention |
| 17 | `tests/e2e/payment_failure_test.go` | Update for booking fee payment failure |
| 18 | `tests/e2e/expiry_test.go` | Add payment timeout test (separate from 1h expiry) |

---

## 9. Assumptions

1. Payment timeout defaults to **10 minutes**, configurable via `PAYMENT_TIMEOUT_MINUTES`.
2. Booking fee payment method is **QRIS** (`payment_method="qris"`).
3. Both system-assigned and user-selected modes use the same payment-before-confirmation flow.
4. The Redis lock TTL = payment timeout + 2 minutes to cover crash recovery.
5. Cancellation from `waiting_payment` has **no fee** (booking fee was never charged).
6. The idempotency key for payment is `"booking-payment-{reservation_id}"` to allow distinguishing booking fee payment from checkout payment.
7. If `CreateReservation` crashes between payment success and status update, the payment idempotency key in the payment service prevents double charges on retry.
8. The existing `idx_reservations_active_spot` partial unique index is dropped and recreated with `waiting_payment` included.

---

## 10. Rollback Plan

If the new flow causes issues:
1. The `000003_payment_flow.up.sql` migration can be reverted with a down migration (`000003_payment_flow.down.sql`):
   - Revert CHECK constraint to original
   - Recreate `idx_reservations_active_spot` without `waiting_payment`
   - Drop `idx_reservations_stale_payment`
2. Revert `CreateReservation` to the original flow (confirmed immediately, billing fire-and-forget).
3. Config change can be reverted by removing `PAYMENT_TIMEOUT_MINUTES`.
