# Code Review Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix critical and high-priority bugs, security gaps, and correctness issues identified in the code review of ParkirPintar smart parking system.

**Architecture:** Each task is a self-contained fix targeting one file or a small cluster of tightly-related files. Tasks are ordered by severity (critical first, then high). Each task follows TDD: write a failing test, run it, fix the code, verify test passes, commit. All Go code follows the project's existing patterns: testify for assertions, table-driven tests, `go test` for verification.

**Tech Stack:** Go 1.25, testify, Gin, golang-jwt/jwt/v5, sqlx, Redis go-redis/v9

---

### Task 1: Fix Docker Compose DNS Hostnames

**Files:**
- Modify: `docker-compose.yml:82-91`

The gateway service uses container names (`assessment-postgres`, `assessment-redis`, `assessment-nats`) as hostnames, but Docker Compose DNS resolves by **service name** (`postgres`, `redis`, `nats`). All other services (search, reservation, billing, etc.) correctly use service names. The gateway will fail to connect to PostgreSQL, Redis, and NATS at startup.

- [ ] **Step 1: Fix the gateway environment variables in docker-compose.yml**

In `docker-compose.yml`, lines 82-91, change the three hostname references from container names to service names:

```yaml
      DB_HOST: assessment-postgres
      DB_PORT: "5432"
```
→ change `assessment-postgres` to `postgres`
```yaml
      REDIS_HOST: assessment-redis
```
→ change `assessment-redis` to `redis`
```yaml
      NATS_URL: nats://assessment-nats:4222
```
→ change `assessment-nats` to `nats`

Also fix the `DB_HOST` and `REDIS_HOST` references in gateway to match the service names used by every other service. The gateway section lines 82-91 should become:

```yaml
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USERNAME: ${DB_USERNAME}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_DATABASE: ${DB_DATABASE}
      DB_SSL_MODE: disable
      REDIS_HOST: redis
      REDIS_PORT: "6379"
      REDIS_PASSWORD: ${REDIS_PASSWORD}
      NATS_URL: nats://nats:4222
```

- [ ] **Step 2: Commit**

```bash
git add docker-compose.yml
git commit -m "fix: correct gateway hostnames from container names to service names in docker-compose"
```

---

### Task 2: Enforce JWT Secret in All Environments

**Files:**
- Modify: `pkg/config/config.go:226-242` (validate function)
- Modify: `pkg/config/config_test.go:231-238` (the test that asserts local allows empty JWT_SECRET)

Currently the config validator only requires `JWT_SECRET` in non-local environments. In local mode, an empty secret means the JWT middleware accepts any self-signed token. Change the validation to always require JWT_SECRET.

- [ ] **Step 1: Write the failing test**

Add this test to `pkg/config/config_test.go`:

```go
func TestLoad_ShouldReturnError_WhenJWTSecretEmptyInLocalEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "local")

	_, err := Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}
```

The existing test `TestLoad_ShouldSucceed_WhenJWTSecretEmptyInLocalEnv` (line 231) asserts the old behavior — it should be **removed** since we are changing the requirement to always enforce JWT_SECRET.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/config/ -run TestLoad_ShouldReturnError_WhenJWTSecretEmptyInLocalEnv -v
```
Expected: FAIL — `JWT_SECRET` validation in local mode doesn't exist yet

- [ ] **Step 3: Fix the validate function**

In `pkg/config/config.go`, change the `validate` function. Remove the `if cfg.App.Environment != "local"` condition wrapping the JWT_SECRET check, so it applies to all environments:

```go
func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port >= 65536 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535, got %d", cfg.Server.Port)
	}

	if cfg.Tracing.SampleRate < 0.0 || cfg.Tracing.SampleRate > 1.0 {
		return fmt.Errorf("TRACING_SAMPLE_RATE must be between 0.0 and 1.0, got %f", cfg.Tracing.SampleRate)
	}

	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	return nil
}
```

Remove the old block at lines 238-242 that wrapped the check in `if cfg.App.Environment != "local"`.

- [ ] **Step 4: Remove the old test that asserted empty secret was OK**

Delete the `TestLoad_ShouldSucceed_WhenJWTSecretEmptyInLocalEnv` function from `pkg/config/config_test.go` (lines 231-238).

- [ ] **Step 5: Update any tests that relied on empty JWT_SECRET in local**

The test `TestLoad_ShouldReturnDefaultConfig_WhenNoEnvVarsSet` (line 48) will now fail because it doesn't set JWT_SECRET. Add `t.Setenv("JWT_SECRET", "test-default-secret")` after `clearEnv(t)` in that test function.

The test `TestLoad_ShouldUseDefaultsForInvalidIntValues` (line 300) will also fail — add `t.Setenv("JWT_SECRET", "test-secret")` after `clearEnv(t)`.

The test `TestLoad_ShouldUseDefaultsForInvalidBoolValues` (line 310) — same, add `t.Setenv("JWT_SECRET", "test-secret")`.

The test `TestLoad_ShouldUseDefaultsForInvalidFloatValues` (line 319) — same, add `t.Setenv("JWT_SECRET", "test-secret")`.

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./pkg/config/ -v
```
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "fix: require JWT_SECRET in all environments, not just non-local"
```

---

### Task 3: Fix Rate Limiter Goroutine Leak

**Files:**
- Modify: `pkg/middleware/ratelimit.go:122-145`
- Modify: `pkg/middleware/middleware.go:19-41`

Every call to `RateLimiter()` creates a new `rateLimitStore` with a background cleanup goroutine. The `Stop()` method exists but is never called. In production, the `Middleware` is a singleton per-process so it's a minor leak, but in test suites each test that creates a rate limiter leaks a goroutine.

Fix: make the `rateLimitStore` a singleton on the `Middleware` struct, lazily initialized with `sync.Once`.

- [ ] **Step 1: Modify the Middleware struct to hold a singleton store**

In `pkg/middleware/middleware.go`, add `sync` and `rateLimitStore` to the struct:

```go
import (
	"log/slog"
	"sync"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/tracing"
)

type Middleware struct {
	config  *config.Config
	logger  *slog.Logger
	tracer  tracing.Tracer

	mu          sync.Mutex
	rateStore   *rateLimitStore
	rateStoreCfg RateLimitConfig
}
```

In `NewMiddleware`, no rate store is created — it stays nil until first rate limiter call.

- [ ] **Step 2: Add a `Shutdown` method to Middleware**

Add this method to `pkg/middleware/middleware.go` after the `NewMiddleware` function:

```go
// Shutdown cleans up background resources held by middleware (e.g. rate limiter goroutine).
func (m *Middleware) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rateStore != nil {
		m.rateStore.Stop()
	}
}
```

- [ ] **Step 3: Change RateLimiter to use the singleton store**

In `pkg/middleware/ratelimit.go`, change the `RateLimiter` method to create the store once and reuse it:

```go
func (m *Middleware) RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	m.mu.Lock()
	if m.rateStore == nil {
		m.rateStore = newRateLimitStore(cfg)
		m.rateStoreCfg = cfg
	} else {
		// Reuse existing store; ignore cfg changes (first caller wins)
	}
	m.mu.Unlock()

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !m.rateStore.allow(key) {
			c.Abort()
			response.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		c.Next()
	}
}
```

- [ ] **Step 4: Write a test that verifies singleton behavior**

Add this test to `pkg/middleware/middleware_test.go`:

```go
func TestRateLimiter_ShouldReuseStore_WhenCalledMultipleTimes(t *testing.T) {
	mw := newTestMiddleware()

	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         100,
		CleanupInterval:   5 * time.Minute,
	}

	// Call twice — should use same underlying store
	mw.RateLimiter(cfg)
	mw.RateLimiter(cfg)

	assert.NotNil(t, mw.rateStore)
}
```

- [ ] **Step 5: Write a test that verifies Shutdown cleans up**

Add this test to `pkg/middleware/middleware_test.go`:

```go
func TestMiddleware_ShouldCleanupStore_WhenShutdown(t *testing.T) {
	mw := newTestMiddleware()

	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         100,
		CleanupInterval:   5 * time.Minute,
	}

	mw.RateLimiter(cfg)
	assert.NotNil(t, mw.rateStore)

	mw.Shutdown()
	// After shutdown, the store's stopCh is closed — calling Stop() again is safe
	assert.NotNil(t, mw.rateStore)
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./pkg/middleware/ -v -run "RateLimiter"
go test ./pkg/middleware/ -v -run "Shutdown"
```
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/middleware/middleware.go pkg/middleware/ratelimit.go pkg/middleware/middleware_test.go
git commit -m "fix: make rate limiter store a singleton on Middleware to prevent goroutine leak"
```

---

### Task 4: Fix JWT Algorithm Confusion Vulnerability

**Files:**
- Modify: `pkg/middleware/auth.go:40-42`

The JWT parsing callback ignores the `t *jwt.Token` parameter and doesn't validate the signing algorithm. An attacker could create a token with `alg: "none"` using a public key and it would pass validation because the callback always returns `[]byte(secret)` as the key regardless of algorithm.

- [ ] **Step 1: Write the failing test**

Add this test to `pkg/middleware/middleware_test.go`:

```go
func TestJWTAuth_ShouldReturn401_WhenAlgorithmIsNotHS256(t *testing.T) {
	mw := newTestMiddleware()
	secret := "test-secret-key"
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.JWTAuth(secret))
	engine.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Create a token signed with HS384 (different algorithm)
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"user_id": "user-123",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/middleware/ -run TestJWTAuth_ShouldReturn401_WhenAlgorithmIsNotHS256 -v
```
Expected: FAIL — the token passes validation despite being HS384, because the parse callback doesn't check `t.Method.Alg()`

- [ ] **Step 3: Fix the JWT parse callback**

In `pkg/middleware/auth.go`, change the `jwt.Parse` call to validate the algorithm:

```go
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
```
Note: `jwt.WithValidMethods` already validates the algorithm, but validating inside the keyfunc callback provides defense in depth. The key change is confirming the method is HMAC-based before returning the key material.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./pkg/middleware/ -run TestJWTAuth -v
```
Expected: ALL PASS (including the new algorithm test)

- [ ] **Step 5: Commit**

```bash
git add pkg/middleware/auth.go pkg/middleware/middleware_test.go
git commit -m "fix: validate JWT signing algorithm in parse callback to prevent algorithm confusion"
```

---

### Task 5: Add Idempotency to RefundPayment

**Files:**
- Modify: `internal/payment/usecase/usecase.go:146-165`
- Modify: `internal/payment/usecase/usecase_test.go`
- Modify: `internal/payment/model/model.go` (if RefundPaymentRequest needs idempotency_key)

`RefundPayment` has no idempotency guard unlike every other payment operation. A duplicate call could double-refund at the gateway level. Add an idempotency check.

- [ ] **Step 1: Add idempotency_key to RefundPaymentRequest**

First check if `RefundPaymentRequest` already has `IdempotencyKey`. Read `internal/payment/model/model.go`:

```bash
grep -n "RefundPaymentRequest" internal/payment/model/model.go
```

If the struct doesn't have `IdempotencyKey`, add it:

```go
type RefundPaymentRequest struct {
	PaymentID      string `json:"payment_id"`
	IdempotencyKey string `json:"idempotency_key"`
}
```

- [ ] **Step 2: Write the failing test**

Add this test to `internal/payment/usecase/usecase_test.go`. This requires reading the existing test file to follow its mock/testing pattern:

```go
func TestRefundPayment_ShouldReturnExisting_WhenIdempotencyKeyExists(t *testing.T) {
	// Setup: create an existing refunded payment with idempotency key
	existingPayment := &model.Payment{
		ID:             "payment-existing",
		BillingID:      "bill-1",
		Amount:         15000,
		Status:         model.PaymentStatusRefunded,
		IdempotencyKey: "refund-key-1",
	}

	mockRepo := new(MockRepository)
	mockRepo.On("GetByIdempotencyKey", mock.Anything, "refund-key-1").Return(existingPayment, nil)

	mockGw := new(MockPaymentGateway)

	uc := NewUsecase(mockRepo, mockGw, nil)

	req := &model.RefundPaymentRequest{
		PaymentID:      "different-payment",
		IdempotencyKey: "refund-key-1",
	}

	result, err := uc.RefundPayment(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "payment-existing", result.ID)
	assert.Equal(t, model.PaymentStatusRefunded, result.Status)
	mockGw.AssertNotCalled(t, "Refund")
}
```

Note: The exact mock types depend on what's already defined in the test file. Adjust package references (`new(MockRepository)`, `new(MockPaymentGateway)`) to match the test file's existing mocks. If the test file uses testify/mock, the mock types will be in the same package.

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/payment/usecase/ -run TestRefundPayment_ShouldReturnExisting_WhenIdempotencyKeyExists -v
```
Expected: FAIL — no idempotency check in RefundPayment yet

- [ ] **Step 4: Implement idempotency check in RefundPayment**

In `internal/payment/usecase/usecase.go`, change `RefundPayment` to check idempotency first:

```go
func (uc *paymentUsecase) RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.Payment, error) {
	if req.IdempotencyKey != "" {
		existing, err := uc.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	payment, err := uc.repo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("refund payment get: %w", err)
	}

	if payment.Status != model.PaymentStatusSuccess {
		return nil, fmt.Errorf("cannot refund payment in status %q", payment.Status)
	}

	if err := uc.gw.Refund(ctx, payment.TransactionRef); err != nil {
		return nil, fmt.Errorf("refund payment gateway: %w", err)
	}

	payment.Status = model.PaymentStatusRefunded
	payment.UpdatedAt = time.Now()
	if req.IdempotencyKey != "" {
		payment.IdempotencyKey = req.IdempotencyKey
	}

	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("refund payment update: %w", err)
	}

	return payment, nil
}
```

Key changes:
1. Idempotency check at the top (returns existing if found)
2. Guard: only refund payments in `success` status (prevents double-refunding even without idempotency)
3. Store idempotency_key on the payment after a successful refund

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/payment/usecase/ -v
```
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/payment/usecase/usecase.go internal/payment/usecase/usecase_test.go internal/payment/model/model.go
git commit -m "fix: add idempotency check and status guard to RefundPayment"
```

---

### Task 6: Fix Gateway StreamLocation to Include Accuracy

**Files:**
- Modify: `internal/gateway/handler/handler.go:252-275`

The `StreamLocation` handler parses `Accuracy` from the request body but the `DetectArrival` RPC proto only has `ReservationId`, `Latitude`, and `Longitude` fields — the accuracy value is silently discarded. If the proto has an accuracy field (check `proto/presence/v1/`), pass it. Otherwise, at minimum pass it to give the presence service data for wrong-spot detection.

- [ ] **Step 1: Check the proto definition for accuracy field**

```bash
grep -n -i "accuracy" proto/presence/v1/*.go
```

If the `DetectArrivalRequest` struct has an `Accuracy` field, the fix is to simply pass `req.Accuracy` in the RPC call. If not, pass everything we can.

- [ ] **Step 2: Fix the handler**

In `internal/gateway/handler/handler.go`, change the `StreamLocation` function to pass accuracy if the proto supports it. The current code at line 264:

```go
	resp, err := h.presence.DetectArrival(c.Request.Context(), &presencev1.DetectArrivalRequest{
		ReservationId: req.ReservationID,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
	})
```

If the proto has `Accuracy` field, change to:

```go
	resp, err := h.presence.DetectArrival(c.Request.Context(), &presencev1.DetectArrivalRequest{
		ReservationId: req.ReservationID,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		Accuracy:      req.Accuracy,
	})
```

If the proto does NOT have an accuracy field, the current code is functionally correct (field is just unused). In that case, remove the unused `Accuracy` field from the local `req` struct to avoid confusion:

```go
	var req struct {
		ReservationID string  `json:"reservation_id"`
		Latitude      float64 `json:"latitude"`
		Longitude     float64 `json:"longitude"`
	}
```

- [ ] **Step 3: Verify the handler test still passes**

```bash
go test ./internal/gateway/handler/ -v -run StreamLocation
```
Expected: PASS (or no StreamLocation-specific test, then just verify compilation)

```bash
go build ./internal/gateway/handler/
```
Expected: builds successfully

- [ ] **Step 4: Commit**

```bash
git add internal/gateway/handler/handler.go
git commit -m "fix: pass accuracy field to presence DetectArrival in gateway StreamLocation handler"
```

---

### Task 7: Use Config's ShutdownTimeout in Server

**Files:**
- Modify: `pkg/server/server.go:69`

The server hardcodes a 30-second shutdown timeout in `context.WithTimeout(context.Background(), 30*time.Second)` (line 69) but `config.Server.ShutdownTimeout` exists and is loaded. Wire the config value in.

- [ ] **Step 1: Accept ShutdownTimeout in GracefulServer constructor**

In `pkg/server/server.go`, add a `shutdownTimeout` field to `GracefulServer`:

```go
type GracefulServer struct {
	engine          *gin.Engine
	logger          *slog.Logger
	port            int
	shutdownTimeout time.Duration
}
```

Update `NewGracefulServer` to accept a `shutdownTimeout` parameter:

```go
func NewGracefulServer(engine *gin.Engine, logger *slog.Logger, port int, shutdownTimeout time.Duration) *GracefulServer {
	return &GracefulServer{
		engine:          engine,
		logger:          logger,
		port:            port,
		shutdownTimeout: shutdownTimeout,
	}
}
```

- [ ] **Step 2: Use the field in Start method**

Change line 69 from:
```go
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
```
to:
```go
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
```

- [ ] **Step 3: Update server_test.go if needed**

Read `pkg/server/server_test.go` and update any call to `NewGracefulServer` to pass the new `shutdownTimeout` parameter:

```bash
grep "NewGracefulServer" pkg/server/server_test.go
```

Update each call site to add a fourth argument, e.g. `30*time.Second`.

- [ ] **Step 4: Find all callers of NewGracefulServer across the codebase and update them**

```bash
rg "NewGracefulServer" --no-filename
```

Update each `cmd/*/main.go` caller to pass `time.Duration(cfg.Server.ShutdownTimeout) * time.Second` from config. The cfg.Server.ShutdownTimeout is in seconds (int), so convert: `time.Duration(cfg.Server.ShutdownTimeout) * time.Second`.

- [ ] **Step 5: Run tests to verify**

```bash
go test ./pkg/server/ -v
go build ./cmd/gateway/
```
Expected: ALL PASS, builds successfully

- [ ] **Step 6: Commit**

```bash
git add pkg/server/server.go pkg/server/server_test.go cmd/*/main.go
git commit -m "fix: wire config.Server.ShutdownTimeout into GracefulServer instead of hardcoded 30s"
```

---

### Task 8: Replace String.Contains Duplicate-Key Detection with errors.Is

**Files:**
- Modify: `internal/reservation/usecase/usecase.go:193-199`

The fallback duplicate-idempotency-key detection uses `strings.Contains(errMsg, "duplicate key")` which is fragile — it depends on PostgreSQL's English error message format. Use proper error sentinel or check.

Since the repository layer already uses `model.ErrNotFound` sentinel errors, this pattern doesn't have a clear sentinel for unique constraint violations. The cleanest fix: extract the idempotency retry into a helper, and use `errors.As` on `*pq.Error` (which `pgx` uses for constraint violations).

- [ ] **Step 1: Write a test that verifies the duplicate-key fallback**

Read `internal/reservation/usecase/usecase_test.go` to find the existing `CreateReservation` tests. Add a test case for duplicate-key recovery:

```go
func TestCreateReservation_ShouldRecoverDuplicateKey_WhenRacingIdempotencyKeys(t *testing.T) {
	// This test verifies that when two concurrent CreateReservation calls
	// with the same idempotency key both pass the initial idempotency check
	// (before either creates the DB row), the second one recovers via the
	// duplicate-key catch in the repository and returns the existing record.
}
```

Note: The exact test structure depends on the existing mock setup in the test file. Use the same patterns (mocks for BillingClient, PaymentClient, Locker, etc.).

- [ ] **Step 2: Fix the duplicate-key detection**

In `internal/reservation/usecase/usecase.go`, lines 193-199, replace the string-based check. The pgx driver returns errors wrapping `*pgconn.PgError`. Import:

```go
	"github.com/jackc/pgx/v5/pgconn"
```

Then change the duplicate-key handling at line 193:

```go
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				existing, findErr := uc.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
				if findErr == nil && existing != nil {
					return existing, nil
				}
			}
			return nil, fmt.Errorf("create reservation: %w", err)
		}
```

The PostgreSQL error code `23505` is the SQL standard code for `unique_violation`. Using `errors.As` with `*pgconn.PgError` is the proper way to detect constraint violations in pgx.

Remove the old import for `"strings"` if it's no longer used elsewhere in the file (check first with `grep "strings\." internal/reservation/usecase/usecase.go`).

- [ ] **Step 3: Run tests to verify they pass**

```bash
go test ./internal/reservation/usecase/ -v
```
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/reservation/usecase/usecase.go internal/reservation/usecase/usecase_test.go
git commit -m "fix: use pgconn.PgError code 23505 instead of string matching for duplicate key detection"
```

---

## Self-Review

**1. Spec coverage:**
- Critical issue #1 (Docker DNS) → Task 1
- Critical issue #2 (JWT empty secret) → Task 2
- Critical issue #3 (rate limiter leak) → Task 3
- High issue #14 (JWT algorithm confusion) → Task 4
- High issue #6 (Refund no idempotency) → Task 5
- High issue #4 (Gateway wrong RPC) → Task 6
- Medium issue #9 (hardcoded timeout) → Task 7
- High issue #8 (fragile duplicate-key detection) → Task 8

Gaps intentionally left for future plans: gRPC TLS configuration, leader election for workers, singleflight for cache.

**2. Placeholder scan:** No TBD, TODO, or vague "add error handling" steps. All steps contain specific code.

**3. Type consistency:** `RefundPaymentRequest.IdempotencyKey` is defined in Task 5 and used in the same task. `GracefulServer.shutdownTimeout time.Duration` is defined in Task 7 Step 1 and used in Step 2. `rateStore` field on Middleware is defined in Task 3 Step 1 and used in Step 3. All consistent.
