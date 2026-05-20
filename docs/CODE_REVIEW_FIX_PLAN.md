# Code Review Fix Plan

**Date:** 2026-05-19
**Based on:** docs/CODE_REVIEW.md (commit af12c49)

---

## Phase 1: Security (Critical)

### 1.1 Upgrade Go toolchain to 1.25.10
- [x] Download and install go1.25.10
- [x] Update `go.mod` directive
- [ ] Update Dockerfile base image
- [ ] Update CI workflow Go version
- [ ] Verify build passes

### 1.2 Upgrade golang.org/x/net to v0.53.0
- [x] `go get golang.org/x/net@v0.53.0`
- [x] `go mod tidy`
- [ ] Verify no breaking changes

### 1.3 Verify fixes
- [ ] Run `govulncheck` — expect 0 vulnerabilities
- [ ] Run full test suite
- [ ] Commit: `fix(security): upgrade Go 1.25.10 + x/net v0.53.0`

---

## Phase 2: Dead Code Removal (High)

### 2.1 pkg/grpcmiddleware
- [ ] Remove `UserIDFromContext`, `RoleFromContext` from `context_keys.go`
- [ ] Remove `AuthStreamInterceptor`, `LoggingStreamInterceptor`, `RecoveryStreamInterceptor`, `IdempotencyUnaryInterceptor`

### 2.2 pkg/nats
- [ ] Remove `DefaultStreamConfigs`, `DefaultConsumerConfigs` from `streams.go`

### 2.3 pkg/logger
- [ ] Remove `Bool` from `logger.go`

### 2.4 pkg/apperror
- [ ] Remove `Unauthorized` from `apperror.go`

### 2.5 pkg/asynq
- [ ] Remove `CancelTask`, `EnqueuePaymentHoldTimeout` from `client.go`

### 2.6 pkg/middleware
- [ ] Remove `APIKeyAuth`, `GenerateTransactionID`, `LogResponse`, `NormalizeMsisdn`

### 2.7 pkg/redis (traced wrappers)
- [ ] Remove unused methods: `GeoAdd`, `GeoRadius`, `SAdd`, `SIsMember`, `SRem`, `ZRem`, `Expire`, `HMSet`, `HGetAll`, `HMGet`

### 2.8 pkg/database
- [ ] Remove `GetTracer`, `Ping` from `postgres_traced.go`

### 2.9 pkg/nats/client
- [ ] Remove `Conn`, `CreateConsumer`, `CreateStream`, `JetStream`

### 2.10 internal/search
- [ ] Remove `DefaultFloorCount` from `handler/nats.go`

### 2.11 Verify
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Commit: `refactor: remove ~30 unused exported symbols`

---

## Phase 3: Deduplication (High)

### 3.1 Eliminate mapError() wrappers
- [ ] Replace `mapError(err)` calls with `grpcerror.MapToGRPCError(err)` directly in:
  - `billing/handler`
  - `payment/handler`
  - `reservation/handler`
  - `search/handler`
  - `analytics/handler`
- [ ] Delete `mapError` functions

### 3.2 Extract shared idempotency helper
- [ ] Create `pkg/idempotency/idempotency.go` with generic check-or-create pattern
- [ ] Refactor `billing/usecase` to use it
- [ ] Refactor `payment/usecase` to use it

### 3.3 Extract PG constraint error helper
- [ ] Create `pkg/database/errors.go` with `IsUniqueViolation(err) bool`
- [ ] Refactor `billing/repository` to use it
- [ ] Refactor `payment/repository` to use it

### 3.4 Generic circuit breaker wrapper (optional)
- [ ] Create `pkg/circuitbreaker/execute.go` with generic `Execute[T]` helper
- [ ] Refactor `reservation/client/` methods to use it

### 3.5 Verify
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Commit: `refactor: deduplicate shared patterns across services`

---

## Phase 4: Dependency Updates (Medium)

### 4.1 Upgrade bsm/redislock
- [ ] `go get github.com/bsm/redislock@v0.10.0`
- [ ] Check for API changes, update call sites if needed

### 4.2 Upgrade alicebob/miniredis
- [ ] `go get github.com/alicebob/miniredis/v2@v2.38.0`

### 4.3 Verify
- [ ] `go test ./...` passes
- [ ] Commit: `chore(deps): upgrade redislock v0.10.0, miniredis v2.38.0`

---

## Phase 5: Architecture Improvement (Low)

### 5.1 Decouple grpcerror mapper
- [ ] Define `type DomainError interface { GRPCCode() codes.Code }` in `pkg/apperror`
- [ ] Have service sentinel errors implement the interface
- [ ] Refactor `grpcerror/mapper.go` to use interface instead of importing service packages
- [ ] Verify no circular deps introduced

### 5.2 Verify
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Commit: `refactor(grpcerror): decouple mapper from service-specific imports`

---

## Execution Order

1. Phase 1 (security) — already in progress
2. Phase 2 (dead code) — mechanical, low risk
3. Phase 3 (dedup) — moderate risk, needs careful testing
4. Phase 4 (deps) — low risk
5. Phase 5 (architecture) — higher risk, optional

## Notes

- Each phase gets its own commit
- Run full test suite after each phase
- Push after Phase 1 immediately (security)
- Phases 2-5 can be batched into a single push
