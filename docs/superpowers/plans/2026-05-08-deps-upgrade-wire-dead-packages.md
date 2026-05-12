# Dependency Upgrade & Dead Package Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade deprecated dependencies (go-redis v8→v9, jwt v4→v5, lib/pq→pgx), remove dead code, wire unused packages (grpcmiddleware, circuitbreaker, redislock) into services, and add repository layer tests.

**Architecture:** Incremental migration — each task produces a compiling, testable state. Dependency upgrades first (no behavior change), then dead code removal, then wiring unused packages, then testing improvements. Each phase is independent and can be verified independently.

**Tech Stack:** Go 1.25, redis/go-redis/v9, golang-jwt/jwt/v5, jackc/pgx/v5, DATA-DOG/go-sqlmock

---

## File Structure

### Files Modified
- `go.mod` / `go.sum` — dependency updates
- `pkg/redis/redis.go` — import path `go-redis/redis/v8` → `redis/go-redis/v9`
- `pkg/redis/redis_traced.go` — import path change + `GeoRadius` → `GeoSearch`
- `pkg/redis/redis_integration_test.go` — import path change
- `pkg/auth/jwt.go` — import path `golang-jwt/jwt/v4` → `golang-jwt/jwt/v5`
- `pkg/auth/jwt_test.go` — import path change
- `pkg/middleware/auth.go` — uses `pkg/auth` (transitive)
- `pkg/middleware/middleware_test.go` — import path change
- `pkg/grpcmiddleware/auth.go` — uses `pkg/auth` (transitive)
- `pkg/grpcmiddleware/auth_test.go` — import path change
- `pkg/grpcmiddleware/auth_property_test.go` — import path change
- `pkg/grpcmiddleware/idempotency.go` — import `go-redis/redis/v8` → `redis/go-redis/v9`
- `pkg/grpcmiddleware/idempotency_test.go` — import path change
- `pkg/grpcmiddleware/idempotency_property_test.go` — import path change
- `pkg/database/postgres.go` — `_ "github.com/lib/pq"` → `_ "github.com/jackc/pgx/v5/stdlib"`
- `cmd/presence/main.go` — import `go-redis/redis/v8` → `redis/go-redis/v9`
- `cmd/reservation/main.go` — wire grpcmiddleware
- `cmd/billing/main.go` — wire grpcmiddleware
- `cmd/payment/main.go` — wire grpcmiddleware
- `cmd/search/main.go` — wire grpcmiddleware
- `cmd/presence/main.go` — wire grpcmiddleware
- `cmd/notification/main.go` — wire grpcmiddleware + fix `streamForSubject`
- `internal/reservation/usecase/usecase.go` — use redislock + circuitbreaker
- `internal/gateway/handler/handler.go` — jwt v5 transitive
- `tests/e2e/setup_test.go` — `_ "github.com/lib/pq"` → `_ "github.com/jackc/pgx/v5/stdlib"`
- `tests/e2e/adapters_test.go` — import path change for go-redis
- `docker-compose.yml` — remove `cmd/api` target (if present)
- `Dockerfile` — remove `cmd/api` from build targets
- `Makefile` — remove `cmd/api` from build targets
- `.gitignore` — add binary artifacts

### Files Deleted
- `pkg/pricing/` — entire directory (739 lines)
- `pkg/crypto/` — entire directory (609 lines)
- `pkg/httpclient/` — entire directory (1,201 lines)
- `internal/example/` — entire directory (970 lines)
- `cmd/api/` — entire directory (126 lines)
- `./reservation` — committed binary
- `./search` — committed binary

### Files Created
- `internal/reservation/repository/repository_test.go` — go-sqlmock unit tests
- `internal/billing/repository/repository_test.go` — go-sqlmock unit tests
- `internal/payment/repository/repository_test.go` — go-sqlmock unit tests

---

## Phase 1: Dependency Upgrades

### Task 1: Upgrade go-redis v8 → v9

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `pkg/redis/redis.go`
- Modify: `pkg/redis/redis_traced.go`
- Modify: `pkg/redis/redis_integration_test.go`
- Modify: `pkg/grpcmiddleware/idempotency.go`
- Modify: `pkg/grpcmiddleware/idempotency_test.go`
- Modify: `pkg/grpcmiddleware/idempotency_property_test.go`
- Modify: `cmd/presence/main.go`
- Modify: `tests/e2e/adapters_test.go`
- Modify: `tests/e2e/setup_test.go`

- [ ] **Step 1: Update go.mod**

```bash
go get github.com/redis/go-redis/v9@latest
go mod tidy
```

- [ ] **Step 2: Update import paths in all files**

Replace all occurrences of `"github.com/go-redis/redis/v8"` with `"github.com/redis/go-redis/v9"`.

Files that need the change:
- `pkg/redis/redis.go:11`
- `pkg/redis/redis_traced.go:7`
- `pkg/redis/redis_integration_test.go:18`
- `pkg/grpcmiddleware/idempotency.go:11`
- `cmd/presence/main.go:10`
- `tests/e2e/adapters_test.go:19`
- `tests/e2e/setup_test.go:22`

And the test files:
- `pkg/grpcmiddleware/idempotency_test.go`
- `pkg/grpcmiddleware/idempotency_property_test.go`

- [ ] **Step 3: Fix API breaking change — GeoRadius removed in v9**

In `pkg/redis/redis.go`, the `GeoRadius` method uses `client.GeoRadius()` which was removed in go-redis v9. Replace with `GeoSearch`:

Old code in `pkg/redis/redis.go`:
```go
func (r *RedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, unit string) ([]redis.GeoLocation, error) {
	return r.client.GeoRadius(ctx, key, longitude, latitude, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      unit,
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
	}).Result()
}
```

New code:
```go
func (r *RedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, unit string) ([]redis.GeoLocation, error) {
	return r.client.GeoSearchLocation(ctx, key, &redis.GeoSearchQuery{
		Longitude:  longitude,
		Latitude:   latitude,
		Radius:     radius,
		RadiusUnit: unit,
		Sort:       "ASC",
	}).Result()
}
```

In `pkg/redis/redis_traced.go`, the `GeoRadius` method signature stays the same (it delegates to `RedisClient`), but the return type references `redis.GeoLocation` which still exists in v9.

- [ ] **Step 4: Update cmd/presence/main.go adapter**

The `presenceRedisAdapter` references `goredis.Client` and `goredis.XAddArgs`:

Old import:
```go
goredis "github.com/go-redis/redis/v8"
```

New import:
```go
goredis "github.com/redis/go-redis/v9"
```

- [ ] **Step 5: Verify compilation**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 6: Run tests**

```bash
go test ./pkg/redis/... ./pkg/grpcmiddleware/... ./cmd/presence/... -count=1
```

Expected: all tests pass

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat(deps): upgrade go-redis v8 to redis/go-redis v9"
```

---

### Task 2: Upgrade golang-jwt v4 → v5

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `pkg/auth/jwt.go`
- Modify: `pkg/auth/jwt_test.go`
- Modify: `pkg/middleware/auth.go`
- Modify: `pkg/middleware/middleware_test.go`
- Modify: `pkg/grpcmiddleware/auth_test.go`
- Modify: `pkg/grpcmiddleware/auth_property_test.go`
- Modify: `internal/gateway/handler/handler_test.go`

- [ ] **Step 1: Update go.mod**

```bash
go get github.com/golang-jwt/jwt/v5@latest
go mod tidy
```

- [ ] **Step 2: Update import paths**

Replace `"github.com/golang-jwt/jwt/v4"` with `"github.com/golang-jwt/jwt/v5"` in:
- `pkg/auth/jwt.go:10`
- `pkg/auth/jwt_test.go:16`
- `pkg/middleware/auth.go:10`
- `pkg/middleware/middleware_test.go:29`
- `pkg/grpcmiddleware/auth_test.go:17`
- `pkg/grpcmiddleware/auth_property_test.go:13`
- `internal/gateway/handler/handler_test.go:20`

- [ ] **Step 3: Fix API breaking changes in pkg/auth/jwt.go**

In jwt v5, `jwt.RegisteredClaims` was renamed to `jwt.RegisteredClaims` (same name, but `ExpiresAt` and `IssuedAt` now use `jwt.NewNumericDate` which already works). The main breaking change is that `jwt.ParseWithClaims` now requires `jwt.WithValidMethods()` option for explicit algorithm validation instead of checking in the key function.

Old `ValidateToken`:
```go
token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return []byte(secret), nil
})
```

New `ValidateToken`:
```go
token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
    return []byte(secret), nil
}, jwt.WithValidMethods([]string{"HS256"}))
```

Remove the `*jwt.SigningMethodHMAC` type check from the key function since `jwt.WithValidMethods` handles it.

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 5: Run tests**

```bash
go test ./pkg/auth/... ./pkg/middleware/... ./pkg/grpcmiddleware/... ./internal/gateway/handler/... -count=1
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(deps): upgrade golang-jwt v4 to v5"
```

---

### Task 3: Replace lib/pq with pgx/v5

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `pkg/database/postgres.go`
- Modify: `tests/e2e/setup_test.go`

- [ ] **Step 1: Update go.mod**

```bash
go get github.com/jackc/pgx/v5@latest
go mod tidy
```

- [ ] **Step 2: Update driver import in pkg/database/postgres.go**

Old:
```go
_ "github.com/lib/pq" // PostgreSQL driver
```

New:
```go
_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
```

- [ ] **Step 3: Update DSN format for pgx**

pgx stdlib uses a different DSN format. The current `host=... port=... user=... password=... dbname=... sslmode=...` key-value format is compatible with pgx stdlib, so no DSN change is needed. However, we should switch to URL format for clarity:

Old:
```go
dsn := fmt.Sprintf(
    "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
    cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode,
)
```

New:
```go
dsn := fmt.Sprintf(
    "postgres://%s:%s@%s:%d/%s?sslmode=%s",
    url.PathEscape(cfg.Username),
    url.PathEscape(cfg.Password),
    cfg.Host,
    cfg.Port,
    cfg.Database,
    cfg.SSLMode,
)
```

Add `"net/url"` to imports.

- [ ] **Step 4: Update driver import in tests/e2e/setup_test.go**

Old:
```go
_ "github.com/lib/pq"
```

New:
```go
_ "github.com/jackc/pgx/v5/stdlib"
```

- [ ] **Step 5: Fix search_path initialization for pgx**

pgx supports `ConnInitSQL` via its connection pool, but since we use `database/sql` through `sqlx`, we need to set the search_path via `OnConnect` or execute it after each connection reset. The current approach (exec `SET search_path` once) is fragile. Add a `ConnInitSQL` approach by executing the SET on each `Ping`:

Update `NewPostgresClient` — add a `db.SetConnMaxIdleTime` and re-execute search_path in a `BeforeAcquire` callback pattern. Since `sqlx` doesn't support `BeforeAcquire` (that's a pgx pool feature), keep the current approach but note it's a known limitation. The `SET search_path` will work for the initial connection; pooled connections that get recycled will need it re-set. For now, this is acceptable — the schema feature is for future multi-tenancy.

- [ ] **Step 6: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 7: Run tests**

```bash
go test ./pkg/database/... -count=1
```

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat(deps): replace lib/pq with jackc/pgx/v5"
```

---

### Task 4: Remove go-redis/redis/v8 and lib/pq from go.mod indirect

- [ ] **Step 1: Run go mod tidy**

After Tasks 1-3, the old modules should be gone from go.mod. Verify:

```bash
go mod tidy
grep -E "go-redis/redis/v8|lib/pq|golang-jwt/jwt/v4" go.mod
```

Expected: no output (these modules are no longer referenced)

- [ ] **Step 2: Verify full build**

```bash
go build ./...
go vet ./...
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "chore(deps): clean up replaced dependencies from go.mod"
```

---

## Phase 2: Dead Code Removal

### Task 5: Remove dead packages

**Files Deleted:**
- `pkg/pricing/` — duplicate of `internal/billing/model/`
- `pkg/crypto/` — never used
- `pkg/httpclient/` — never used
- `internal/example/` — only used by dead `cmd/api/`
- `cmd/api/` — legacy monolith

- [ ] **Step 1: Verify no imports exist for these packages**

```bash
grep -r '"parkir-pintar/pkg/pricing"' --include='*.go' .
grep -r '"parkir-pintar/pkg/crypto"' --include='*.go' .
grep -r '"parkir-pintar/pkg/httpclient"' --include='*.go' .
grep -r '"parkir-pintar/internal/example"' --include='*.go' .
grep -r '"parkir-pintar/cmd/api"' --include='*.go' .
```

Expected: only self-references and test files within those packages

- [ ] **Step 2: Delete the packages**

```bash
rm -rf pkg/pricing/ pkg/crypto/ pkg/httpclient/ internal/example/ cmd/api/
```

- [ ] **Step 3: Remove DATA-DOG/go-sqlmock if no longer used**

After removing `internal/example/`, check if go-sqlmock is still imported:

```bash
grep -r '"github.com/DATA-DOG/go-sqlmock"' --include='*.go' .
```

If no results, it will be removed by `go mod tidy`. If we're adding repo tests (Phase 4), keep it.

- [ ] **Step 4: Update Dockerfile to remove cmd/api build target**

In the Dockerfile, find the build loop that compiles all cmd/* binaries and remove `api` from it.

- [ ] **Step 5: Update Makefile to remove cmd/api build target**

Remove `api` from the build targets list.

- [ ] **Step 6: Run go mod tidy and verify**

```bash
go mod tidy
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: remove dead packages (pricing, crypto, httpclient, example, cmd/api)"
```

---

### Task 6: Remove committed binary artifacts and update .gitignore

- [ ] **Step 1: Remove committed binaries**

```bash
rm -f ./reservation ./search
```

- [ ] **Step 2: Update .gitignore**

Add to `.gitignore`:
```
# Build artifacts
/reservation
/search
```

Or more broadly:
```
# Build artifacts
/*.exe
/reservation
/search
```

- [ ] **Step 3: Remove from git tracking (if still tracked)**

```bash
git rm --cached reservation search 2>/dev/null || true
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove committed binaries, update .gitignore"
```

---

## Phase 3: Wire Unused Packages

### Task 7: Wire grpcmiddleware interceptors into all gRPC services

This is the most critical wiring task. Currently, gRPC services have no interceptors — no auth, no recovery, no logging, no tracing, no rate limiting.

**Files:**
- Modify: `cmd/reservation/main.go`
- Modify: `cmd/billing/main.go`
- Modify: `cmd/payment/main.go`
- Modify: `cmd/search/main.go`
- Modify: `cmd/presence/main.go`
- Modify: `cmd/notification/main.go`
- Modify: `pkg/grpcmiddleware/idempotency.go` — update go-redis import (done in Task 1)

- [ ] **Step 1: Fix cmd/notification/main.go streamForSubject function**

Replace the fragile index-based matching:

Old:
```go
func streamForSubject(subject string) string {
	switch {
	case len(subject) > 12 && subject[:12] == "reservation.":
		return "RESERVATIONS"
	case len(subject) > 8 && subject[:8] == "billing.":
		return "BILLING"
	case len(subject) > 8 && subject[:8] == "payment.":
		return "PAYMENTS"
	default:
		return "RESERVATIONS"
	}
}
```

New:
```go
func streamForSubject(subject string) string {
	switch {
	case strings.HasPrefix(subject, "reservation.") || strings.HasPrefix(subject, "spot."):
		return "RESERVATIONS"
	case strings.HasPrefix(subject, "billing."):
		return "BILLING"
	case strings.HasPrefix(subject, "payment."):
		return "PAYMENTS"
	default:
		return "RESERVATIONS"
	}
}
```

Add `"strings"` to imports.

- [ ] **Step 2: Wire interceptors in cmd/reservation/main.go**

After creating `redisClient`, add interceptors initialization:

```go
import (
    grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
)

// After redisClient is created, add:
interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, redisClient)

grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, 30*time.Second,
    grpc.ChainUnaryInterceptor(
        interceptors.RecoveryUnaryInterceptor(),
        interceptors.AuthUnaryInterceptor([]string{
            "/reservation.v1.ReservationService/HealthCheck",
        }),
        interceptors.LoggingUnaryInterceptor(),
        interceptors.TracingUnaryInterceptor(),
        interceptors.RateLimitUnaryInterceptor(grpcmiddleware.RateLimitConfig{
            RequestsPerSecond: 100,
            BurstSize:        200,
            CleanupInterval:  5 * time.Minute,
        }),
    ),
)
```

- [ ] **Step 3: Wire interceptors in cmd/billing/main.go**

Same pattern as reservation, but no redis client needed (idempotency interceptor uses redis, skip it):

```go
interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, nil)

grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, 30*time.Second,
    grpc.ChainUnaryInterceptor(
        interceptors.RecoveryUnaryInterceptor(),
        interceptors.AuthUnaryInterceptor(nil),
        interceptors.LoggingUnaryInterceptor(),
        interceptors.TracingUnaryInterceptor(),
        interceptors.RateLimitUnaryInterceptor(grpcmiddleware.RateLimitConfig{
            RequestsPerSecond: 100,
            BurstSize:        200,
            CleanupInterval:  5 * time.Minute,
        }),
    ),
)
```

- [ ] **Step 4: Wire interceptors in cmd/payment/main.go**

Same pattern as billing.

- [ ] **Step 5: Wire interceptors in cmd/search/main.go**

Same pattern, but includes redisClient (for idempotency if needed later):

```go
interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, redisClient)
```

- [ ] **Step 6: Wire interceptors in cmd/presence/main.go**

Same pattern, includes redisClient.

- [ ] **Step 7: Wire interceptors in cmd/notification/main.go**

Same pattern, no redisClient needed.

- [ ] **Step 8: Verify compilation**

```bash
go build ./cmd/...
```

- [ ] **Step 9: Run unit tests**

```bash
go test ./pkg/grpcmiddleware/... -count=1
```

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "feat(grpc): wire interceptors (auth, recovery, logging, tracing, rate-limit) into all gRPC services"
```

---

### Task 8: Wire redislock into reservation usecase

Currently, the reservation usecase uses raw `SetNX`/`Delete` for distributed locking. The `pkg/redislock` package provides a more robust implementation with atomic Lua-script release and lock token verification.

**Files:**
- Modify: `internal/reservation/usecase/usecase.go`

- [ ] **Step 1: Update RedisClient interface in usecase**

Replace the raw `SetNX`/`Delete` interface with `redislock.Locker`:

Old:
```go
type RedisClient interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
}
```

New — add `redislock.Locker` as a dependency:
```go
import "parkir-pintar/pkg/redislock"

type reservationUsecase struct {
	repo          repository.Repository
	locker        redislock.Locker
	nats          NATSClient
	billingClient BillingClient
	paymentClient PaymentClient
}
```

Update `NewUsecase`:
```go
func NewUsecase(
	repo repository.Repository,
	locker redislock.Locker,
	nats NATSClient,
	billingClient BillingClient,
	paymentClient PaymentClient,
) Usecase {
	return &reservationUsecase{
		repo:          repo,
		locker:        locker,
		nats:          nats,
		billingClient: billingClient,
		paymentClient: paymentClient,
	}
}
```

- [ ] **Step 2: Update lock acquisition in CreateReservation**

Old:
```go
lockKey := fmt.Sprintf("lock:spot:%s", spotID)
acquired, err := uc.redis.SetNX(ctx, lockKey, "locked", lockTTL)
if err != nil || !acquired {
    return nil, apperror.New("CONFLICT", "spot is being reserved by another driver", 409)
}

locked := true
defer func() {
    if locked {
        if delErr := uc.redis.Delete(ctx, lockKey); delErr != nil {
            slog.Error("failed to release spot lock", slog.String("lock_key", lockKey), slog.Any("error", delErr))
        }
    }
}()
```

New:
```go
lockKey := fmt.Sprintf("lock:spot:%s", spotID)
lock, err := uc.locker.Acquire(ctx, lockKey, lockTTL)
if err != nil {
    if errors.Is(err, redislock.ErrLockNotAcquired) {
        return nil, apperror.New("CONFLICT", "spot is being reserved by another driver", 409)
    }
    return nil, fmt.Errorf("acquire lock: %w", err)
}

defer func() {
    if unlockErr := lock.Release(ctx); unlockErr != nil {
        slog.Error("failed to release spot lock", slog.String("lock_key", lockKey), slog.Any("error", unlockErr))
    }
}()
```

- [ ] **Step 3: Update failReservationInternal to use lock.Release**

Old:
```go
if locked != nil && *locked {
    if delErr := uc.redis.Delete(ctx, lockKey); delErr != nil {
        slog.Error("failed to release spot lock", slog.String("lock_key", lockKey), slog.Any("error", delErr))
    }
    *locked = false
}
```

New — the `defer lock.Release()` in CreateReservation handles it automatically. Remove the manual lock release from `failReservationInternal` and remove the `locked` parameter:

```go
func (uc *reservationUsecase) failReservationInternal(ctx context.Context, reservation *model.Reservation) {
```

The `defer lock.Release()` in CreateReservation will fire when the function returns, releasing the lock after the transaction completes. This fixes the double-release race from C2 in the code review.

- [ ] **Step 4: Update cmd/reservation/main.go to pass redislock.Locker**

Add import and create locker:

```go
import "parkir-pintar/pkg/redislock"

// After creating redisClient:
locker := redislock.NewLocker(redisClient)
```

Update usecase construction:
```go
uc := usecase.NewUsecase(
    repo, locker, natsClient,
    client.NewBillingClient(billingGRPC),
    client.NewPaymentClient(paymentGRPC),
)
```

- [ ] **Step 5: Update mock for RedisClient**

The existing mock in `internal/reservation/mocks/mock_redis_client.go` needs to be replaced with a mock for `redislock.Locker`. Regenerate:

```bash
mockgen -destination=internal/reservation/mocks/mock_locker.go -package=mocks parkir-pintar/pkg/redislock Locker
```

Or update the existing tests that mock `RedisClient` to instead mock `redislock.Locker`.

- [ ] **Step 6: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/reservation/... ./pkg/redislock/... -count=1
```

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat(reservation): use redislock package for distributed locking"
```

---

### Task 9: Wire circuitbreaker into gRPC client calls

Currently, the payment and billing gRPC client adapters don't use circuit breakers. Wire `pkg/circuitbreaker` into the reservation service's gRPC client adapters.

**Files:**
- Modify: `internal/reservation/client/billing_client.go`
- Modify: `internal/reservation/client/payment_client.go`
- Modify: `cmd/reservation/main.go`

- [ ] **Step 1: Wrap billing client calls with circuit breaker**

In `internal/reservation/client/billing_client.go`, wrap each gRPC call:

```go
import "parkir-pintar/pkg/circuitbreaker"

type billingClient struct {
    client billingv1.BillingServiceClient
    cb     *circuitbreaker.CircuitBreaker
}

func NewBillingClient(client billingv1.BillingServiceClient) BillingClient {
    return &billingClient{
        client: client,
        cb: circuitbreaker.New(circuitbreaker.Config{
            FailureThreshold: 5,
            OpenTimeout:      30 * time.Second,
            HalfOpenMaxProbes: 1,
        }),
    }
}
```

Wrap each method call:
```go
func (c *billingClient) StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*billingmodel.BillingRecord, error) {
    var result *billingmodel.BillingRecord
    err := c.cb.Execute(func() error {
        var err error
        result, err = c.startBillingInner(ctx, reservationID, bookingFee, idempotencyKey)
        return err
    })
    if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
        return nil, apperror.New("SERVICE_UNAVAILABLE", "billing service temporarily unavailable", 503)
    }
    return result, err
}
```

Same pattern for `CalculateFee`, `GenerateInvoice`, `ApplyPenalty`.

- [ ] **Step 2: Wrap payment client calls with circuit breaker**

Same pattern in `internal/reservation/client/payment_client.go`.

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/reservation/... ./pkg/circuitbreaker/... -count=1
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(reservation): wrap billing/payment gRPC calls with circuit breaker"
```

---

## Phase 4: Repository Layer Tests

### Task 10: Add go-sqlmock repository tests

**Files Created:**
- `internal/reservation/repository/repository_test.go`
- `internal/billing/repository/repository_test.go`
- `internal/payment/repository/repository_test.go`

- [ ] **Step 1: Ensure go-sqlmock is in go.mod**

```bash
go get github.com/DATA-DOG/go-sqlmock@latest
```

- [ ] **Step 2: Write reservation repository tests**

Create `internal/reservation/repository/repository_test.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/reservation/model"
)

func setupTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return sqlx.NewDb(db, "sqlmock"), mock
}

func TestFindByIdempotencyKey_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM reservations WHERE idempotency_key = $1")).
		WithArgs("key-123").
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.FindByIdempotencyKey(context.Background(), "key-123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByIdempotencyKey_Found(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "spot_id", "vehicle_type", "assignment_mode",
		"status", "idempotency_key", "confirmed_at", "expires_at",
		"checked_in_at", "checked_out_at", "cancelled_at", "created_at", "updated_at",
	}).AddRow(
		"res-1", "driver-1", "spot-1", "car", "system_assigned",
		"confirmed", "key-123", now, now.Add(time.Hour),
		nil, nil, nil, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM reservations WHERE idempotency_key = $1")).
		WithArgs("key-123").
		WillReturnRows(rows)

	repo := NewRepository(db)
	result, err := repo.FindByIdempotencyKey(context.Background(), "key-123")

	assert.NoError(t, err)
	assert.Equal(t, "res-1", result.ID)
	assert.Equal(t, "key-123", result.IdempotencyKey)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindAvailableSpot(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "floor_number", "spot_number", "vehicle_type", "spot_code",
		"status", "created_at", "updated_at",
	}).AddRow(
		"spot-1", 1, 1, "car", "F1-C-001",
		"available", time.Now(), time.Now(),
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM parking_spots WHERE vehicle_type = $1 AND status = 'available' ORDER BY floor_number, spot_number LIMIT 1 FOR UPDATE SKIP LOCKED`,
	)).WithArgs("car").WillReturnRows(rows)

	repo := NewRepository(db)
	result, err := repo.FindAvailableSpot(context.Background(), "car")

	assert.NoError(t, err)
	assert.Equal(t, "spot-1", result.ID)
	assert.Equal(t, "car", result.VehicleType)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindAvailableSpot_NoSpots(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM parking_spots WHERE vehicle_type = $1 AND status = 'available' ORDER BY floor_number, spot_number LIMIT 1 FOR UPDATE SKIP LOCKED`,
	)).WithArgs("motorcycle").WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	result, err := repo.FindAvailableSpot(context.Background(), "motorcycle")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateReservationTx(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	reservation := &model.Reservation{
		ID:             "res-1",
		DriverID:       "driver-1",
		SpotID:         "spot-1",
		VehicleType:    "car",
		AssignmentMode: "system_assigned",
		Status:         "waiting_payment",
		IdempotencyKey: "key-123",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO reservations`,
	)).WithArgs(
		reservation.ID, reservation.DriverID, reservation.SpotID,
		reservation.VehicleType, reservation.AssignmentMode, reservation.Status,
		reservation.IdempotencyKey, reservation.ConfirmedAt, reservation.ExpiresAt,
		reservation.CheckedInAt, reservation.CheckedOutAt, reservation.CancelledAt,
		reservation.CreatedAt, reservation.UpdatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := NewRepository(db)
	err := repo.WithTransaction(context.Background(), func(tx *sqlx.Tx) error {
		return repo.CreateReservationTx(context.Background(), tx, reservation)
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSpotStatusTx(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE parking_spots SET status = $1, updated_at = NOW() WHERE id = $2",
	)).WithArgs("reserved", "spot-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewRepository(db)
	err := repo.WithTransaction(context.Background(), func(tx *sqlx.Tx) error {
		return repo.UpdateSpotStatusTx(context.Background(), tx, "spot-1", "reserved")
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByIDForUpdate(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "spot_id", "vehicle_type", "assignment_mode",
		"status", "idempotency_key", "confirmed_at", "expires_at",
		"checked_in_at", "checked_out_at", "cancelled_at", "created_at", "updated_at",
	}).AddRow(
		"res-1", "driver-1", "spot-1", "car", "system_assigned",
		"confirmed", "key-123", now, now.Add(time.Hour),
		nil, nil, nil, now, now,
	)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM reservations WHERE id = $1 FOR UPDATE",
	)).WithArgs("res-1").WillReturnRows(rows)
	mock.ExpectCommit()

	repo := NewRepository(db)
	var result *model.Reservation
	err := repo.WithTransaction(context.Background(), func(tx *sqlx.Tx) error {
		var txErr error
		result, txErr = repo.GetByIDForUpdate(context.Background(), tx, "res-1")
		return txErr
	})

	assert.NoError(t, err)
	assert.Equal(t, "res-1", result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindExpiredReservations(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "spot_id", "vehicle_type", "assignment_mode",
		"status", "idempotency_key", "confirmed_at", "expires_at",
		"checked_in_at", "checked_out_at", "cancelled_at", "created_at", "updated_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM reservations WHERE status = 'confirmed' AND expires_at < NOW()",
	)).WillReturnRows(rows)

	repo := NewRepository(db)
	result, err := repo.FindExpiredReservations(context.Background())

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 3: Write billing repository tests**

Create `internal/billing/repository/repository_test.go` following the same pattern. Test:
- `CreateBillingRecord`
- `FindByReservationID`
- `UpdateBillingRecord`
- `FindByIdempotencyKey`

- [ ] **Step 4: Write payment repository tests**

Create `internal/payment/repository/repository_test.go` following the same pattern. Test:
- `CreatePayment`
- `FindByBillingID`
- `FindByID`
- `UpdatePaymentStatus`

- [ ] **Step 5: Run all repository tests**

```bash
go test ./internal/reservation/repository/... ./internal/billing/repository/... ./internal/payment/repository/... -v -count=1
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "test(repository): add go-sqlmock unit tests for reservation, billing, payment repositories"
```

---

## Phase 5: Security & Cleanup

### Task 11: Remove config/.env from git tracking

- [ ] **Step 1: Remove from git tracking**

```bash
git rm --cached config/.env
```

- [ ] **Step 2: Verify .gitignore includes config/.env**

Check `.gitignore` contains `config/.env`. If not, add it.

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "security: remove config/.env from git tracking"
```

---

### Task 12: Final verification

- [ ] **Step 1: Run full test suite**

```bash
make test
```

- [ ] **Step 2: Run linting**

```bash
make lint
```

- [ ] **Step 3: Run security scanner**

```bash
make gosec
```

- [ ] **Step 4: Run race detector**

```bash
make test-race
```

- [ ] **Step 5: Verify go.mod is clean**

```bash
go mod tidy
go vet ./...
```

- [ ] **Step 6: Verify no old imports remain**

```bash
grep -r "go-redis/redis/v8\|golang-jwt/jwt/v4\|lib/pq" --include='*.go' .
```

Expected: no output

- [ ] **Step 7: Commit if any changes**

```bash
git add -A
git commit -m "chore: final cleanup after dependency upgrade and dead code removal"
```
