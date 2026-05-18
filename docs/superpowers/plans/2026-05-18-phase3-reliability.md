# Phase 3: Reliability, Infrastructure & Quality Fixes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix NATS handler reliability (timeouts, poison pills), service entrypoint issues (signal handling, exit codes), infrastructure gaps (CI pipeline, healthchecks, image pinning), broken tests, and shared library bugs.

**Architecture:** Add context timeouts to all NATS handlers, fix poison-pill handling, add signal handling to services, fix CI dependencies, fix broken test assertions, fix gRPC idempotency interceptor serialization, add nil guard to NATS publisher.

**Tech Stack:** Go, NATS JetStream, Docker, GitHub Actions, gRPC, protobuf

---

### Task 1: Fix NATS Handlers — Add Timeouts and Poison-Pill Handling

**Files:**
- Modify: `internal/search/handler/nats.go`
- Modify: `internal/analytics/handler/nats.go`
- Modify: `internal/reservation/handler/nats.go`

- [ ] **Step 1: Fix search NATS handler — add context timeout**

In `internal/search/handler/nats.go`, replace `context.Background()` (around line 50):

```go
import "time"

// Inside the message handler function:
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
```

- [ ] **Step 2: Fix analytics NATS handler — add timeout and fix poison pill**

In `internal/analytics/handler/nats.go`, replace the handler logic (around lines 35-55):

```go
func (h *NATSHandler) handleMessage(msg jetstream.Msg) {
    var event model.ReservationEvent
    if err := json.Unmarshal(msg.Data(), &event); err != nil {
        slog.Error("analytics: failed to unmarshal event",
            slog.String("subject", msg.Subject()),
            slog.Any("error", err))
        // Terminate poison messages — they will never become valid
        _ = msg.Term()
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := h.usecase.RecordEvent(ctx, &event); err != nil {
        slog.Error("analytics: failed to record event",
            slog.String("subject", msg.Subject()),
            slog.Any("error", err))
        _ = msg.Nak()
        return
    }

    if err := msg.Ack(); err != nil {
        slog.Error("analytics: failed to ack message",
            slog.String("subject", msg.Subject()),
            slog.Any("error", err))
    }
}
```

- [ ] **Step 3: Fix reservation NATS handler — add timeout**

In `internal/reservation/handler/nats.go`, replace `context.Background()` (around line 61):

```go
func (h *NATSHandler) handleMessage(msg jetstream.Msg) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // ... rest of handler uses ctx ...
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/search/handler/ ./internal/analytics/handler/ ./internal/reservation/handler/ -v`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/search/handler/nats.go internal/analytics/handler/nats.go internal/reservation/handler/nats.go
git commit -m "fix(reliability): add context timeouts to NATS handlers, terminate poison pills"
```

---

### Task 2: Fix NATS Publisher Nil Guard

**Files:**
- Modify: `pkg/nats/publisher.go`
- Modify: `pkg/nats/nats_test.go`

- [ ] **Step 1: Write the failing test**

Add to `pkg/nats/nats_test.go`:

```go
func TestPublisher_NilClient_ReturnsError(t *testing.T) {
    pub := NewPublisher(nil)
    err := pub.Publish(context.Background(), "test.subject", []byte("data"), "msg-1")
    require.Error(t, err)
    assert.Contains(t, err.Error(), "publisher client is nil")
}
```

- [ ] **Step 2: Run test to verify it fails (panics)**

Run: `go test ./pkg/nats/ -run TestPublisher_NilClient -v`
Expected: FAIL (panic: nil pointer dereference)

- [ ] **Step 3: Add nil guard**

In `pkg/nats/publisher.go`:

```go
import (
    "context"
    "errors"
    "fmt"
)

// ErrNilClient is returned when Publish is called on a Publisher with a nil client.
var ErrNilClient = errors.New("publisher client is nil")

// Publish sends a message to the given subject with deduplication via msgID.
func (p *Publisher) Publish(ctx context.Context, subject string, data []byte, msgID string) error {
    if p.client == nil {
        return ErrNilClient
    }
    _, err := p.client.Publish(ctx, subject, data, msgID)
    if err != nil {
        return fmt.Errorf("publisher: %w", err)
    }
    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/nats/ -run TestPublisher_NilClient -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/nats/
git commit -m "fix(nats): add nil guard to Publisher.Publish to prevent panic"
```

---

### Task 3: Fix gRPC Idempotency Interceptor Response Serialization

**Files:**
- Modify: `pkg/grpcmiddleware/idempotency.go`

The current implementation serializes gRPC responses as JSON and deserializes to `interface{}` (becomes `map[string]interface{}`). The gRPC framework expects the registered proto message type. Fix by storing the proto wire format.

- [ ] **Step 1: Write the failing test**

Add to `pkg/grpcmiddleware/interceptors_test.go`:

```go
func TestIdempotencyInterceptor_ReturnsCachedProtoResponse(t *testing.T) {
    // Setup: Redis with a pre-cached protobuf response
    mr := miniredis.RunT(t)
    rClient := setupRedisClient(t, mr)
    interceptors := NewInterceptors(rClient, slog.Default())

    cfg := IdempotencyConfig{
        TTL:     time.Minute,
        Methods: []string{"/test.Service/Method"},
    }
    interceptor := interceptors.IdempotencyUnaryInterceptor(cfg)

    // First call — handler executes and caches
    callCount := 0
    handler := func(ctx context.Context, req interface{}) (interface{}, error) {
        callCount++
        return &testProtoResponse{Value: "hello"}, nil
    }

    ctx := metadata.NewIncomingContext(context.Background(),
        metadata.Pairs("x-idempotency-key", "test-key-1"))
    info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

    resp1, err := interceptor(ctx, nil, info, handler)
    require.NoError(t, err)
    assert.Equal(t, 1, callCount)
    assert.NotNil(t, resp1)

    // Second call — should return cached response without calling handler
    resp2, err := interceptor(ctx, nil, info, handler)
    require.NoError(t, err)
    assert.Equal(t, 1, callCount) // handler NOT called again
    assert.NotNil(t, resp2)
}
```

- [ ] **Step 2: Fix serialization to use proto wire format**

Replace the JSON marshal/unmarshal in `pkg/grpcmiddleware/idempotency.go`:

```go
import (
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/encoding/protojson"
)

// In the acquired branch (around line 96), replace json.Marshal:
// Serialize response. Try proto first, fall back to JSON for non-proto types.
var data []byte
var marshalErr error
if pm, ok := resp.(proto.Message); ok {
    data, marshalErr = protojson.Marshal(pm)
} else {
    data, marshalErr = json.Marshal(resp)
}
if marshalErr != nil {
    i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: failed to marshal response",
        slog.String("key", redisKey),
        slog.String("error", marshalErr.Error()),
    )
    return resp, nil
}
```

For the deserialization in `pollForCachedResponse` (around line 150), we cannot reconstruct the exact proto type without knowing it. The pragmatic fix is to store the response as-is and return it directly. However, since we don't know the type at poll time, the best approach is to **not use the poll path for proto responses** — instead, return an Aborted status so the client retries and hits the idempotency check in the handler itself:

```go
// In pollForCachedResponse, replace the unmarshal block:
// Real response available — but we cannot reconstruct the proto type here.
// Return a signal that the operation completed successfully.
// The client should retry, and the handler's own idempotency check will return the result.
return nil, status.Errorf(codes.Aborted, "concurrent request completed, please retry for cached result")
```

Actually, a better approach: store both the serialized bytes AND the gRPC method name, then use a type registry. But the simplest correct fix is to store protojson and return it as a `structpb.Struct`:

```go
import "google.golang.org/protobuf/types/known/structpb"

// In pollForCachedResponse, replace the unmarshal:
var respMap map[string]interface{}
if unmarshalErr := json.Unmarshal([]byte(cached), &respMap); unmarshalErr != nil {
    i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: failed to unmarshal cached response",
        slog.String("key", redisKey),
        slog.String("error", unmarshalErr.Error()),
    )
    return nil, status.Errorf(codes.Internal, "internal server error")
}
structResp, convErr := structpb.NewStruct(respMap)
if convErr != nil {
    return nil, status.Errorf(codes.Internal, "internal server error")
}
return structResp, nil
```

- [ ] **Step 3: Run tests**

Run: `go test ./pkg/grpcmiddleware/ -v`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add pkg/grpcmiddleware/idempotency.go
git commit -m "fix(grpc): use protojson for idempotency response serialization"
```

---

### Task 4: Fix Service Entrypoints — Signal Handling and Exit Codes

**Files:**
- Modify: `cmd/billing/main.go`
- Modify: `cmd/payment/main.go`
- Modify: `cmd/presence/main.go`
- Modify: `cmd/search/main.go`
- Modify: `cmd/reservation/main.go`

- [ ] **Step 1: Add signal handling pattern to all gRPC services**

Apply this pattern to each `cmd/*/main.go` (example for billing):

```go
import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // ... existing setup code ...

    // Create signal-aware context
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // Start gRPC server in a goroutine
    serverErr := make(chan error, 1)
    go func() {
        serverErr <- grpcSrv.Start()
    }()

    // Wait for signal or server error
    select {
    case <-ctx.Done():
        log.Info("shutdown signal received")
    case err := <-serverErr:
        if err != nil {
            log.Error("gRPC server error", slog.Any("error", err))
        }
    }

    // Run shutdown manager
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()
    sm.Shutdown(shutdownCtx)

    // Exit with appropriate code
    select {
    case err := <-serverErr:
        if err != nil {
            os.Exit(1)
        }
    default:
    }
}
```

- [ ] **Step 2: Apply to all 5 gRPC services**

Apply the same pattern to:
- `cmd/billing/main.go`
- `cmd/payment/main.go`
- `cmd/presence/main.go`
- `cmd/search/main.go`
- `cmd/reservation/main.go`

- [ ] **Step 3: Verify all services compile**

Run: `go build ./cmd/...`
Expected: All compile successfully.

- [ ] **Step 4: Commit**

```bash
git add cmd/
git commit -m "fix(services): add OS signal handling and non-zero exit on startup failure"
```

---

### Task 5: Fix CI Pipeline — Tests Before Image Push

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Update build-push-staging dependencies**

In `.github/workflows/ci.yml`, change the `needs` field of `build-push-staging` (around line 205):

```yaml
build-push-staging:
  needs: [secret-scan, test, lint, security]
  if: github.ref == 'refs/heads/main' && github.event_name == 'push'
```

- [ ] **Step 2: Add health check wait before DAST**

After the `build-push-staging` job, add a wait step to the `dast` job:

```yaml
dast:
  needs: build-push-staging
  steps:
    - name: Wait for deployment to propagate
      run: |
        for i in $(seq 1 30); do
          if curl -sf "${{ secrets.STAGING_URL }}/health" > /dev/null 2>&1; then
            echo "Staging is healthy"
            exit 0
          fi
          echo "Waiting for staging... attempt $i/30"
          sleep 10
        done
        echo "Staging health check timed out"
        exit 1
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "fix(ci): require tests to pass before pushing staging images"
```

---

### Task 6: Fix Docker Healthchecks

**Files:**
- Modify: `docker-compose.yml`
- Modify: `deploy/coolify/app/docker-compose.yml`

- [ ] **Step 1: Replace process-grep healthchecks with proper checks**

For the gateway (HTTP service), use curl:

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 10s
  timeout: 5s
  retries: 3
  start_period: 10s
```

For gRPC services, use grpc_health_probe (add to Dockerfiles) or a simple TCP check:

```yaml
healthcheck:
  test: ["CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:${HEALTH_PORT}/health || exit 1"]
  interval: 10s
  timeout: 5s
  retries: 3
  start_period: 10s
```

- [ ] **Step 2: Add health HTTP endpoint to gRPC services**

Each gRPC service already has a health package. Ensure the health HTTP server is started in each `cmd/*/main.go`:

```go
// Already exists in most services via:
healthSvc := health.NewService(log)
healthSvc.AddChecker("postgres", health.NewPostgresChecker(db))
go health.ServeHTTP(healthSvc, cfg.Health.Port, log)
```

Verify this is present in all services. If not, add it.

- [ ] **Step 3: Update docker-compose healthchecks**

Replace all `ps | grep` healthchecks in `docker-compose.yml`:

```yaml
# For each gRPC service:
healthcheck:
  test: ["CMD-SHELL", "wget -qO- http://localhost:8081/health || exit 1"]
  interval: 10s
  timeout: 5s
  retries: 3
  start_period: 15s
```

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml deploy/coolify/app/docker-compose.yml
git commit -m "fix(infra): replace process-grep healthchecks with HTTP health endpoint checks"
```

---

### Task 7: Pin Image Versions and Fix Go Version Mismatch

**Files:**
- Modify: `Dockerfile`
- Modify: `build/billing.Dockerfile`
- Modify: `build/gateway.Dockerfile`
- Modify: `build/payment.Dockerfile`
- Modify: `build/presence.Dockerfile`
- Modify: `build/reservation.Dockerfile`
- Modify: `build/search.Dockerfile`
- Modify: `infra/terraform/environments/staging.tfvars`
- Modify: `infra/terraform/environments/production.tfvars`

- [ ] **Step 1: Standardize Go version to 1.25 across all Dockerfiles**

In all `build/*.Dockerfile`, change line 4:

```dockerfile
# Before: FROM golang:1.24-alpine AS builder
# After:
FROM golang:1.25-alpine AS builder
```

- [ ] **Step 2: Standardize Alpine version to 3.21**

In `Dockerfile` (root), change the runtime stage:

```dockerfile
# Before: FROM alpine:3.19
# After:
FROM alpine:3.21
```

- [ ] **Step 3: Replace :latest in Terraform with placeholder for CI**

In `infra/terraform/environments/staging.tfvars`:

```hcl
# Before: container_image = "ghcr.io/piresc/parkir-pintar:latest"
# After:
container_image = "ghcr.io/piresc/parkir-pintar:staging"  # Override with SHA in CI: -var="container_image=ghcr.io/piresc/parkir-pintar:sha-abc123"
```

In `infra/terraform/environments/production.tfvars`:

```hcl
container_image = "ghcr.io/piresc/parkir-pintar:production"  # Override with tagged version in CD
```

- [ ] **Step 4: Verify Docker builds**

Run: `docker build -f build/gateway.Dockerfile -t test .`
Expected: Builds with Go 1.25.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile build/ infra/terraform/environments/
git commit -m "fix(infra): standardize Go 1.25 and Alpine 3.21, remove :latest from Terraform"
```

---

### Task 8: Fix Broken Tests

**Files:**
- Modify: `tests/e2e_docker/double_book_test.go`
- Modify: `tests/e2e/race_test.go`
- Modify: `tests/load/k6_load_test.js`

- [ ] **Step 1: Fix Docker E2E double-book test**

In `tests/e2e_docker/double_book_test.go`, change to use `user_selected` mode targeting the same spot:

```go
func TestDoubleBooking_SameSpot(t *testing.T) {
    // First, find an available spot
    availResp := testhelpers.Get(t, baseURL+"/api/v1/availability?vehicle_type=motorcycle")
    require.Equal(t, http.StatusOK, availResp.StatusCode)
    // ... parse to get a specific spot_id ...

    // Both requests target the same spot with user_selected mode
    body1 := fmt.Sprintf(`{
        "driver_id": "%s",
        "vehicle_type": "motorcycle",
        "assignment_mode": "user_selected",
        "spot_id": "%s",
        "idempotency_key": "double-book-1"
    }`, driverID, spotID)

    body2 := fmt.Sprintf(`{
        "driver_id": "driver-other",
        "vehicle_type": "motorcycle",
        "assignment_mode": "user_selected",
        "spot_id": "%s",
        "idempotency_key": "double-book-2"
    }`, spotID)

    // Send concurrently
    var wg sync.WaitGroup
    statusCodes := make([]int, 2)
    // ... concurrent requests ...

    // Exactly one should succeed, one should get 409
    successCount := 0
    conflictCount := 0
    for _, code := range statusCodes {
        if code == http.StatusCreated {
            successCount++
        }
        if code == http.StatusConflict {
            conflictCount++
        }
    }
    assert.Equal(t, 1, successCount, "exactly one reservation should succeed")
    assert.Equal(t, 1, conflictCount, "exactly one should get conflict")
}
```

- [ ] **Step 2: Fix race test assertion**

In `tests/e2e/race_test.go`, around line 283:

```go
// Before:
// assert.GreaterOrEqual(t, successCount, 1, "at least one concurrent operation should succeed")

// After:
assert.Equal(t, 1, successCount, "exactly one concurrent lifecycle operation should succeed")
```

- [ ] **Step 3: Fix k6 load test — correct paths and add auth**

In `tests/load/k6_load_test.js`:

```javascript
// Fix vehicle types (remove 'truck')
const VEHICLE_TYPES = ['car', 'motorcycle'];

// Fix API paths
// Before: /api/v1/search/availability
// After:
const availabilityURL = `${BASE_URL}/api/v1/availability`;

// Before: /api/v1/search/floors/:id
// After:
const floorURL = `${BASE_URL}/api/v1/floors/${floorNumber}`;

// Add JWT auth header
const params = {
    headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${__ENV.TEST_JWT_TOKEN}`,
    },
};
```

- [ ] **Step 4: Run e2e tests (unit-level, not Docker)**

Run: `go test ./tests/e2e/ -run TestDoubleBooking -v -count=1`
Expected: Compiles (may skip if no infra available).

- [ ] **Step 5: Commit**

```bash
git add tests/
git commit -m "fix(tests): fix double-book test logic, strengthen race assertions, fix k6 paths"
```

---

### Task 9: Fix Migration Issues

**Files:**
- Move: `migrations/000007_placeholder.up.sql` → `db/migrations/000007_placeholder.up.sql`
- Move: `migrations/000007_placeholder.down.sql` → `db/migrations/000007_placeholder.down.sql`
- Modify: `db/migrations/000011_remove_penalties.down.sql`

- [ ] **Step 1: Move migration 000007 to correct directory**

```bash
mv migrations/000007_placeholder.up.sql db/migrations/000007_placeholder.up.sql
mv migrations/000007_placeholder.down.sql db/migrations/000007_placeholder.down.sql
rmdir migrations/  # Remove empty directory
```

- [ ] **Step 2: Fix FK reference in migration 000011 down**

In `db/migrations/000011_remove_penalties.down.sql`, fix the foreign key reference:

```sql
-- Before:
-- FOREIGN KEY (reservation_id) REFERENCES reservations(id)

-- After:
FOREIGN KEY (reservation_id) REFERENCES reservation.reservations(id)
```

- [ ] **Step 3: Commit**

```bash
git add db/migrations/ migrations/
git commit -m "fix(migrations): move 000007 to correct dir, fix FK reference in 000011"
```

---

### Task 10: Fix gRPC Rate Limit Store Leak

**Files:**
- Modify: `pkg/grpcmiddleware/interceptors.go`

- [ ] **Step 1: Store rate limit store reference in Interceptors struct**

In `pkg/grpcmiddleware/interceptors.go`, add a field:

```go
type Interceptors struct {
    redisClient RedisClient
    logger      *slog.Logger
    rateLimitStore *ratelimit.Store  // Add this field
}
```

- [ ] **Step 2: Update RateLimitUnaryInterceptor to store the reference**

In `pkg/grpcmiddleware/ratelimit.go`:

```go
func (i *Interceptors) RateLimitUnaryInterceptor(cfg RateLimitConfig) grpc.UnaryServerInterceptor {
    store := ratelimit.NewStore(cfg.ToStoreConfig())
    i.rateLimitStore = store  // Store reference for cleanup
    // ... rest of implementation ...
}
```

- [ ] **Step 3: Add Shutdown method**

```go
// Shutdown stops background goroutines (rate limit cleanup).
func (i *Interceptors) Shutdown() {
    if i.rateLimitStore != nil {
        i.rateLimitStore.Stop()
    }
}
```

- [ ] **Step 4: Call Shutdown in service entrypoints**

In each `cmd/*/main.go`, register the interceptors shutdown:

```go
sm.Register("grpc-interceptors", func(ctx context.Context) error {
    interceptors.Shutdown()
    return nil
})
```

- [ ] **Step 5: Run tests**

Run: `go test ./pkg/grpcmiddleware/ -v`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/grpcmiddleware/ cmd/
git commit -m "fix(grpc): stop rate limit cleanup goroutine on shutdown"
```

---

### Task 11: Remove os.Exit from pkg/server

**Files:**
- Modify: `pkg/server/server.go`

- [ ] **Step 1: Replace os.Exit with error return**

In `pkg/server/server.go` (around line 64):

```go
// Before:
// slog.Error("server failed to start", slog.String("error", err.Error()))
// os.Exit(1)

// After:
if err != nil {
    return fmt.Errorf("server failed to start: %w", err)
}
```

Ensure the `Start()` method signature returns `error` (it likely already does).

- [ ] **Step 2: Run tests**

Run: `go test ./pkg/server/ -v`
Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add pkg/server/
git commit -m "fix(server): return error instead of calling os.Exit from library code"
```

---

### Task 12: Fix Search Singleflight Context Sharing

**Files:**
- Modify: `internal/search/usecase/usecase.go`

- [ ] **Step 1: Use detached context inside singleflight**

In `internal/search/usecase/usecase.go` (around line 76):

```go
import "context"

// In GetAvailability, inside the singleflight Do callback:
result, err, _ := uc.sf.Do(cacheKey, func() (interface{}, error) {
    // Use a detached context with timeout to prevent first-caller cancellation
    // from affecting coalesced callers.
    sfCtx, sfCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
    defer sfCancel()

    floors, err := uc.repo.GetAvailabilityByVehicleType(sfCtx, req.VehicleType)
    if err != nil {
        return nil, err
    }
    // ... cache and return ...
})
```

Note: `context.WithoutCancel` requires Go 1.21+. Since this project uses Go 1.25, this is available.

- [ ] **Step 2: Run tests**

Run: `go test ./internal/search/usecase/ -v`
Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/search/usecase/usecase.go
git commit -m "fix(search): use detached context in singleflight to prevent cross-caller cancellation"
```

---

## Final Verification

After all tasks in all 3 phases are complete:

- [ ] **Full build check**

```bash
go build ./...
```

- [ ] **Full test suite**

```bash
go test ./... -count=1
```

- [ ] **Lint check**

```bash
golangci-lint run ./...
```

- [ ] **Frontend build**

```bash
cd frontend && npm run build
```

- [ ] **Docker build (gateway as smoke test)**

```bash
docker build -f build/gateway.Dockerfile -t parkir-pintar-gateway .
```
