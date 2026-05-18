# Phase 2: Concurrency & Data Integrity Fixes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate all TOCTOU races, double-booking gaps, and data integrity issues across billing, payment, and reservation domains.

**Architecture:** Add DB-level unique constraints for idempotency, wrap check-then-act sequences in transactions with `SELECT FOR UPDATE`, add optimistic locking where appropriate, fix event publisher race, and fix NATS message ID uniqueness.

**Tech Stack:** Go, PostgreSQL (transactions, constraints), Redis, NATS JetStream

---

### Task 1: Add DB Unique Constraint for Payment Idempotency Key

**Files:**
- Create: `db/migrations/000012_payment_idempotency_constraint.up.sql`
- Create: `db/migrations/000012_payment_idempotency_constraint.down.sql`
- Modify: `internal/payment/repository/repository.go`

- [ ] **Step 1: Create the up migration**

```sql
-- 000012_payment_idempotency_constraint
-- Adds unique constraint on idempotency_key to prevent TOCTOU race in payment creation.
ALTER TABLE payment.payments
    ADD CONSTRAINT uq_payments_idempotency_key UNIQUE (idempotency_key);
```

- [ ] **Step 2: Create the down migration**

```sql
-- 000012_payment_idempotency_constraint (rollback)
ALTER TABLE payment.payments
    DROP CONSTRAINT IF EXISTS uq_payments_idempotency_key;
```

- [ ] **Step 3: Update repository to handle duplicate key error**

In `internal/payment/repository/repository.go`, modify `CreatePayment` to detect the unique violation and return the existing record:

```go
import (
    "github.com/jackc/pgx/v5/pgconn"
)

func (r *sqlxRepository) CreatePayment(ctx context.Context, payment *model.Payment) error {
    _, err := r.db.NamedExecContext(ctx, `
        INSERT INTO payment.payments (id, billing_id, amount, payment_method, payment_gateway, idempotency_key, status, created_at, updated_at)
        VALUES (:id, :billing_id, :amount, :payment_method, :payment_gateway, :idempotency_key, :status, :created_at, :updated_at)`,
        payment)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            // Unique violation on idempotency_key — return conflict error
            return fmt.Errorf("%w: idempotency_key=%s", ErrConflict, payment.IdempotencyKey)
        }
        return fmt.Errorf("create payment: %w", err)
    }
    return nil
}
```

- [ ] **Step 4: Add ErrConflict sentinel to repository**

```go
var ErrConflict = errors.New("conflict: duplicate record")
```

- [ ] **Step 5: Update usecase to handle conflict by fetching existing**

In `internal/payment/usecase/usecase.go`, after `CreatePayment` call (line 91):

```go
if err := uc.repo.CreatePayment(ctx, payment); err != nil {
    if errors.Is(err, repository.ErrConflict) {
        // Race condition: another request created it first. Fetch and return.
        existing, fetchErr := uc.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
        if fetchErr != nil {
            return nil, fmt.Errorf("process payment fetch after conflict: %w", fetchErr)
        }
        return existing, nil
    }
    return nil, fmt.Errorf("process payment create: %w", err)
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/payment/... -v`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add db/migrations/000012* internal/payment/
git commit -m "fix(payment): add DB unique constraint for idempotency key, handle race"
```

---

### Task 2: Fix Double-Refund Race Condition

**Files:**
- Modify: `internal/payment/usecase/usecase.go`
- Modify: `internal/payment/repository/repository.go`

- [ ] **Step 1: Add optimistic update method to repository**

Add to `internal/payment/repository/repository.go`:

```go
// UpdatePaymentWithStatusCheck updates a payment only if its current status matches expectedStatus.
// Returns ErrConflict if the status has changed (optimistic lock failure).
func (r *sqlxRepository) UpdatePaymentWithStatusCheck(ctx context.Context, payment *model.Payment, expectedStatus model.PaymentStatus) error {
    result, err := r.db.ExecContext(ctx, `
        UPDATE payment.payments
        SET status = $1, transaction_ref = $2, idempotency_key = $3, paid_at = $4, updated_at = $5
        WHERE id = $6 AND status = $7`,
        payment.Status, payment.TransactionRef, payment.IdempotencyKey, payment.PaidAt, payment.UpdatedAt,
        payment.ID, expectedStatus)
    if err != nil {
        return fmt.Errorf("update payment with status check: %w", err)
    }
    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("%w: payment status changed concurrently", ErrConflict)
    }
    return nil
}
```

- [ ] **Step 2: Add to Repository interface**

```go
type Repository interface {
    // ... existing methods ...
    UpdatePaymentWithStatusCheck(ctx context.Context, payment *model.Payment, expectedStatus model.PaymentStatus) error
}
```

- [ ] **Step 3: Use optimistic update in RefundPayment**

In `internal/payment/usecase/usecase.go`, replace the refund update (around line 229-237):

```go
payment.Status = model.PaymentStatusRefunded
payment.UpdatedAt = time.Now()

if err := uc.repo.UpdatePaymentWithStatusCheck(ctx, payment, model.PaymentStatusSuccess); err != nil {
    if errors.Is(err, repository.ErrConflict) {
        return nil, fmt.Errorf("refund failed: payment status changed concurrently (possible double-refund)")
    }
    return nil, fmt.Errorf("refund payment update: %w", err)
}
```

- [ ] **Step 4: Remove the idempotency key overwrite**

Remove lines 231-233 that overwrite `payment.IdempotencyKey`. Refund idempotency should not mutate the original payment's key:

```go
// REMOVE this block:
// if req.IdempotencyKey != "" {
//     payment.IdempotencyKey = req.IdempotencyKey
// }
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/payment/... -v`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/payment/
git commit -m "fix(payment): prevent double-refund with optimistic locking, stop key overwrite"
```

---

### Task 3: Fix Payment EventPublisher Race Condition

**Files:**
- Modify: `internal/payment/usecase/usecase.go`

- [ ] **Step 1: Inject EventPublisher via constructor**

Replace `SetEventPublisher` with constructor injection:

```go
// NewUsecase creates a new payment Usecase with all required dependencies.
// eventPublisher is optional (nil-safe).
func NewUsecase(repo repository.Repository, gw gateway.PaymentGateway, eventPublisher EventPublisher) Usecase {
    return &paymentUsecase{
        repo:           repo,
        gw:             gw,
        eventPublisher: eventPublisher,
    }
}

// Remove SetEventPublisher method entirely.
```

- [ ] **Step 2: Remove SetEventPublisher from interface**

```go
type Usecase interface {
    ProcessPayment(ctx context.Context, req *model.ProcessPaymentRequest) (*model.Payment, error)
    ProcessQRIS(ctx context.Context, req *model.ProcessQRISRequest) (*model.Payment, error)
    RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.Payment, error)
    GetPaymentStatus(ctx context.Context, req *model.GetPaymentStatusRequest) (*model.Payment, error)
    // SetEventPublisher removed — inject via NewUsecase
}
```

- [ ] **Step 3: Update cmd/payment/main.go to pass publisher to constructor**

In `cmd/payment/main.go`, change the usecase initialization:

```go
// Before:
// uc := usecase.NewUsecase(repo, gw)
// uc.SetEventPublisher(natsGateway)

// After:
uc := usecase.NewUsecase(repo, gw, natsGateway)
```

- [ ] **Step 4: Update tests that use SetEventPublisher**

In test files, pass the mock publisher to `NewUsecase` instead of calling `SetEventPublisher`.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/payment/... ./cmd/payment/ -v`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/payment/ cmd/payment/
git commit -m "fix(payment): inject EventPublisher via constructor to eliminate race"
```

---

### Task 4: Add Max Amount Validation to Payment Handler

**Files:**
- Modify: `internal/payment/handler/handler.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/payment/handler/handler_test.go`:

```go
func TestProcessPayment_RejectsExcessiveAmount(t *testing.T) {
    ctx := t.Context()
    handler := setupTestHandler(t)

    req := &paymentpb.ProcessPaymentRequest{
        BillingId:      "billing-1",
        Amount:         100_000_001, // Over 100M IDR limit
        PaymentMethod:  "qris",
        IdempotencyKey: "key-1",
    }

    _, err := handler.ProcessPayment(ctx, req)
    require.Error(t, err)
    st, _ := status.FromError(err)
    assert.Equal(t, codes.InvalidArgument, st.Code())
    assert.Contains(t, st.Message(), "amount exceeds maximum")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/payment/handler/ -run TestProcessPayment_RejectsExcessiveAmount -v`
Expected: FAIL

- [ ] **Step 3: Add max amount validation**

In `internal/payment/handler/handler.go`, after the `amount > 0` check (around line 47):

```go
const maxPaymentAmount = 100_000_000 // 100M IDR

if req.GetAmount() <= 0 {
    return nil, status.Error(codes.InvalidArgument, "amount must be positive")
}
if req.GetAmount() > maxPaymentAmount {
    return nil, status.Error(codes.InvalidArgument, "amount exceeds maximum allowed (100,000,000)")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/payment/handler/ -run TestProcessPayment_RejectsExcessiveAmount -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/payment/handler/
git commit -m "fix(payment): add maximum amount validation (100M IDR)"
```

---

### Task 5: Fix Billing TOCTOU — Wrap StartBilling in Transaction

**Files:**
- Modify: `internal/billing/repository/repository.go`
- Modify: `internal/billing/usecase/usecase.go`

- [ ] **Step 1: Add unique constraint migration for billing**

Create `db/migrations/000013_billing_idempotency_constraint.up.sql`:

```sql
-- 000013_billing_idempotency_constraint
-- Prevent duplicate billing records via DB constraint.
ALTER TABLE billing.billing_records
    ADD CONSTRAINT uq_billing_records_idempotency_key UNIQUE (idempotency_key);

ALTER TABLE billing.billing_records
    ADD CONSTRAINT uq_billing_records_reservation_id UNIQUE (reservation_id);
```

Create `db/migrations/000013_billing_idempotency_constraint.down.sql`:

```sql
ALTER TABLE billing.billing_records
    DROP CONSTRAINT IF EXISTS uq_billing_records_idempotency_key;

ALTER TABLE billing.billing_records
    DROP CONSTRAINT IF EXISTS uq_billing_records_reservation_id;
```

- [ ] **Step 2: Handle duplicate key error in billing repository**

In `internal/billing/repository/repository.go`, update `CreateBillingRecord`:

```go
func (r *sqlxRepository) CreateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
    _, err := r.db.NamedExecContext(ctx, `
        INSERT INTO billing.billing_records (id, reservation_id, booking_fee, parking_fee, overnight_fee, total_amount, duration_minutes, billed_hours, is_overnight, status, idempotency_key, created_at, updated_at)
        VALUES (:id, :reservation_id, :booking_fee, :parking_fee, :overnight_fee, :total_amount, :duration_minutes, :billed_hours, :is_overnight, :status, :idempotency_key, :created_at, :updated_at)`,
        record)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return fmt.Errorf("%w: duplicate billing record", ErrConflict)
        }
        return fmt.Errorf("create billing record: %w", err)
    }
    return nil
}
```

- [ ] **Step 3: Add ErrConflict to billing repository**

```go
var ErrConflict = errors.New("conflict: duplicate record")
```

- [ ] **Step 4: Update StartBilling to handle conflict gracefully**

In `internal/billing/usecase/usecase.go`, after `CreateBillingRecord`:

```go
if err := uc.repo.CreateBillingRecord(ctx, record); err != nil {
    if errors.Is(err, repository.ErrConflict) {
        // Race: another request created it. Fetch by reservation ID.
        existing, fetchErr := uc.repo.GetByReservationID(ctx, req.ReservationID)
        if fetchErr != nil {
            return nil, fmt.Errorf("start billing fetch after conflict: %w", fetchErr)
        }
        return existing, nil
    }
    return nil, fmt.Errorf("start billing: %w", err)
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/billing/... -v`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add db/migrations/000013* internal/billing/
git commit -m "fix(billing): add DB constraints to prevent duplicate billing records"
```

---

### Task 6: Enforce Billing State Machine Transitions

**Files:**
- Modify: `internal/billing/usecase/usecase.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/billing/usecase/usecase_test.go`:

```go
func TestGenerateInvoice_RejectsPendingStatus(t *testing.T) {
    repo := new(mocks.MockRepository)
    uc := NewUsecase(repo)

    record := &model.BillingRecord{
        ID:     "bill-1",
        Status: model.BillingStatusPending, // Not yet calculated
    }
    repo.On("GetByIdempotencyKey", mock.Anything, "key-1").Return(nil, repository.ErrNotFound)
    repo.On("GetByReservationID", mock.Anything, "res-1").Return(record, nil)

    _, err := uc.GenerateInvoice(context.Background(), &model.GenerateInvoiceRequest{
        ReservationID:  "res-1",
        IdempotencyKey: "key-1",
    })

    require.Error(t, err)
    assert.Contains(t, err.Error(), "cannot invoice")
}

func TestCalculateFee_RejectsAlreadyInvoiced(t *testing.T) {
    repo := new(mocks.MockRepository)
    uc := NewUsecase(repo)

    record := &model.BillingRecord{
        ID:     "bill-1",
        Status: model.BillingStatusInvoiced,
    }
    repo.On("GetByReservationID", mock.Anything, "res-1").Return(record, nil)

    _, err := uc.CalculateFee(context.Background(), &model.CalculateFeeRequest{
        ReservationID: "res-1",
        CheckInAt:     time.Now().Add(-1 * time.Hour),
        CheckOutAt:    time.Now(),
    })

    require.Error(t, err)
    assert.Contains(t, err.Error(), "cannot calculate fee")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/billing/usecase/ -run "TestGenerateInvoice_RejectsPendingStatus|TestCalculateFee_RejectsAlreadyInvoiced" -v`
Expected: FAIL

- [ ] **Step 3: Add state guards to GenerateInvoice**

In `internal/billing/usecase/usecase.go`, after fetching the record in `GenerateInvoice` (line 118-121):

```go
record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
if err != nil {
    return nil, fmt.Errorf("generate invoice get record: %w", err)
}

if record.Status != model.BillingStatusCalculated {
    return nil, fmt.Errorf("cannot invoice billing record in status %q: expected %q", record.Status, model.BillingStatusCalculated)
}
```

- [ ] **Step 4: Add state guard to CalculateFee**

After fetching the record in `CalculateFee` (line 83-86):

```go
record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
if err != nil {
    return nil, fmt.Errorf("calculate fee get record: %w", err)
}

if record.Status != model.BillingStatusPending {
    return nil, fmt.Errorf("cannot calculate fee for billing record in status %q: expected %q", record.Status, model.BillingStatusPending)
}
```

- [ ] **Step 5: Make ApplyOvernightFee idempotent**

In `ApplyOvernightFee`, add a no-op guard:

```go
record, err := uc.repo.GetByReservationID(ctx, req.ReservationID)
if err != nil {
    return nil, fmt.Errorf("apply overnight fee get record: %w", err)
}

// Idempotent: if already marked overnight, return as-is
if record.IsOvernight {
    return record, nil
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/billing/usecase/ -v`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/billing/usecase/
git commit -m "fix(billing): enforce state machine transitions, make overnight fee idempotent"
```

---

### Task 7: Fix Reservation Active-Check Atomicity

**Files:**
- Create: `db/migrations/000014_active_reservation_constraint.up.sql`
- Create: `db/migrations/000014_active_reservation_constraint.down.sql`

- [ ] **Step 1: Create partial unique index migration (up)**

```sql
-- 000014_active_reservation_constraint
-- Enforce at most one active reservation per driver at the database level.
CREATE UNIQUE INDEX idx_reservations_one_active_per_driver
    ON reservation.reservations (driver_id)
    WHERE status IN ('waiting_payment', 'confirmed', 'checked_in');
```

- [ ] **Step 2: Create down migration**

```sql
DROP INDEX IF EXISTS reservation.idx_reservations_one_active_per_driver;
```

- [ ] **Step 3: Handle constraint violation in reservation usecase**

In `internal/reservation/usecase/usecase.go`, in `CreateReservation`, after the transaction creates the reservation, handle the unique violation:

```go
// Inside the WithTransaction callback, after CreateReservationTx:
if err := uc.repo.CreateReservationTx(ctx, tx, reservation); err != nil {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
        pgErr.ConstraintName == "idx_reservations_one_active_per_driver" {
        return apperror.New("CONFLICT", "driver already has an active reservation", 409)
    }
    return fmt.Errorf("create reservation: %w", err)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/reservation/... -v`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add db/migrations/000014* internal/reservation/usecase/
git commit -m "fix(reservation): add DB constraint for one active reservation per driver"
```

---

### Task 8: Fix failReservationInternal — Add FOR UPDATE

**Files:**
- Modify: `internal/reservation/usecase/usecase.go`

- [ ] **Step 1: Add row lock to failReservationInternal**

Replace the current `failReservationInternal` (lines 382-397):

```go
// failReservationInternal transitions a waiting_payment reservation to failed,
// releases the spot. Uses FOR UPDATE to prevent concurrent modification.
func (uc *reservationUsecase) failReservationInternal(ctx context.Context, reservation *model.Reservation) {
    if txErr := uc.repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
        // Re-fetch with lock to prevent TOCTOU
        locked, err := uc.repo.GetByIDForUpdate(ctx, tx, reservation.ID)
        if err != nil {
            return fmt.Errorf("fail reservation lock: %w", err)
        }
        // Only transition from waiting_payment
        if locked.Status != model.StatusWaitingPayment {
            return nil // Already transitioned by another path — no-op
        }

        now := time.Now()
        locked.Status = model.StatusFailed
        locked.UpdatedAt = now

        if err := uc.repo.UpdateReservationTx(ctx, tx, locked); err != nil {
            return err
        }
        return uc.repo.UpdateSpotStatusTx(ctx, tx, locked.SpotID, spotStatusAvailable)
    }); txErr != nil {
        slog.Error("failed to release spot on payment failure",
            slog.String("reservation_id", reservation.ID),
            slog.Any("error", txErr))
    }
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/reservation/... -v`
Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/reservation/usecase/usecase.go
git commit -m "fix(reservation): add FOR UPDATE to failReservationInternal to prevent race"
```

---

### Task 9: Fix NATS Event Message ID Uniqueness

**Files:**
- Modify: `internal/reservation/gateway/nats.go`

- [ ] **Step 1: Add timestamp to message IDs**

Replace the `msgID` generation in `internal/reservation/gateway/nats.go`:

```go
import (
    "time"
)

// PublishSpotUpdated publishes a spot status change to the search service.
func (p *NATSEventPublisher) PublishSpotUpdated(ctx context.Context, event SpotUpdatedEvent) error {
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("marshal spot updated event: %w", err)
    }
    // Include UnixNano to ensure uniqueness across repeated transitions
    msgID := fmt.Sprintf("spot-%s-%s-%d", event.SpotID, event.Status, time.Now().UnixNano())
    return p.publisher.Publish(ctx, pkgnats.SubjectReservationSearchSpotUpdated, data, msgID)
}

// PublishReservationEvent publishes a reservation lifecycle event to analytics.
func (p *NATSEventPublisher) PublishReservationEvent(ctx context.Context, subject string, event ReservationEvent) error {
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("marshal reservation event: %w", err)
    }
    msgID := fmt.Sprintf("res-%s-%s-%d", event.ReservationID, event.Status, time.Now().UnixNano())
    return p.publisher.Publish(ctx, subject, data, msgID)
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/reservation/... -v`
Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/reservation/gateway/nats.go
git commit -m "fix(reservation): make NATS event msgID globally unique with timestamp"
```

---

### Task 10: Fix pkg/apperror Importing internal/ Packages

**Files:**
- Delete: `pkg/apperror/grpc.go`
- Create: `internal/shared/grpcerror/mapper.go`
- Modify: `internal/billing/handler/handler.go`
- Modify: `internal/payment/handler/handler.go`
- Modify: `internal/reservation/handler/handler.go`
- Modify: `internal/search/handler/handler.go`

- [ ] **Step 1: Create the new internal shared package**

Create `internal/shared/grpcerror/mapper.go`:

```go
// Package grpcerror provides domain-aware gRPC error mapping.
// This lives in internal/ because it depends on domain-specific sentinel errors.
package grpcerror

import (
    "errors"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "parkir-pintar/pkg/apperror"

    billingrepo "parkir-pintar/internal/billing/repository"
    paymentrepo "parkir-pintar/internal/payment/repository"
    reservationmodel "parkir-pintar/internal/reservation/model"
    searchrepo "parkir-pintar/internal/search/repository"
)

// MapToGRPCError maps domain errors to gRPC status codes.
func MapToGRPCError(err error) error {
    if err == nil {
        return nil
    }

    // Check repository/model sentinel errors (NotFound variants).
    if errors.Is(err, searchrepo.ErrNotFound) ||
        errors.Is(err, billingrepo.ErrNotFound) ||
        errors.Is(err, paymentrepo.ErrNotFound) ||
        errors.Is(err, reservationmodel.ErrNotFound) {
        return status.Error(codes.NotFound, err.Error())
    }

    // Check reservation model sentinel errors.
    if errors.Is(err, reservationmodel.ErrConflict) {
        return status.Error(codes.AlreadyExists, err.Error())
    }
    if errors.Is(err, reservationmodel.ErrInvalidTransition) {
        return status.Error(codes.FailedPrecondition, err.Error())
    }
    if errors.Is(err, reservationmodel.ErrSpotUnavailable) {
        return status.Error(codes.FailedPrecondition, err.Error())
    }

    // Check structured AppError with HTTP status mapping.
    var appErr *apperror.AppError
    if errors.As(err, &appErr) {
        switch appErr.HTTPStatus {
        case 400:
            return status.Error(codes.InvalidArgument, appErr.Message)
        case 403:
            return status.Error(codes.PermissionDenied, appErr.Message)
        case 404:
            return status.Error(codes.NotFound, appErr.Message)
        case 409:
            return status.Error(codes.AlreadyExists, appErr.Message)
        default:
            return status.Error(codes.Internal, appErr.Message)
        }
    }

    return status.Error(codes.Internal, err.Error())
}
```

- [ ] **Step 2: Delete pkg/apperror/grpc.go**

Remove the file `pkg/apperror/grpc.go`.

- [ ] **Step 3: Update all handler imports**

In each handler that imports `pkg/apperror.MapToGRPCError`, change to:

```go
// Before:
// "parkir-pintar/pkg/apperror"
// apperror.MapToGRPCError(err)

// After:
"parkir-pintar/internal/shared/grpcerror"
// grpcerror.MapToGRPCError(err)
```

Update these files:
- `internal/billing/handler/handler.go`
- `internal/payment/handler/handler.go`
- `internal/reservation/handler/handler.go`
- `internal/search/handler/handler.go`
- `internal/analytics/handler/handler.go`

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: All packages compile.

- [ ] **Step 5: Run all tests**

Run: `go test ./internal/... ./pkg/... -v`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/shared/ pkg/apperror/ internal/billing/handler/ internal/payment/handler/ internal/reservation/handler/ internal/search/handler/ internal/analytics/handler/
git commit -m "refactor: move MapToGRPCError to internal/shared/grpcerror to fix pkg→internal import"
```
