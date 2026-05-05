# ParkirPintar — Implementation Progress

## Summary
| Metric | Value |
|--------|-------|
| Total Tasks | 16 (with 30+ sub-tasks) |
| Completed | 16 |
| Remaining | 0 |
| Last Updated | 2026-04-26 |

## Completed Tasks

### Task 1 — Write bug condition exploration tests
- **Files**: `pkg/nats/client_test.go`, `internal/billing/usecase/usecase_property_test.go`, `internal/gateway/handler/handler_property_test.go`, `internal/presence/handler/handler_property_test.go`, `pkg/httpclient/security_property_test.go`, `internal/payment/usecase/usecase_property_test.go`, `pkg/grpcmiddleware/idempotency_property_test.go`
- **Tests**: 7 bug condition exploration tests — all FAILED as expected on unfixed code (confirms bugs exist)
- **Status**: All 7 counterexamples documented: NATS race, billing lost update, gateway wrong RPC, missing coordinate validation, SSRF bypass, payment sleep ignoring context, idempotency GET-then-SET race
- **Notes**: Bugfix exploration phase — test failures are the success condition here.

### Task 2 — Write preservation property tests
- **Files**: `pkg/nats/client_preservation_test.go`, `pkg/nats/traced_client_preservation_test.go`, `internal/billing/usecase/usecase_preservation_test.go`, `internal/billing/usecase/usecase_billing_preservation_test.go`, `internal/gateway/handler/handler_preservation_test.go`, `internal/presence/handler/handler_preservation_test.go`, `pkg/httpclient/security_preservation_test.go`, `pkg/middleware/ratelimit_preservation_test.go`, `internal/search/usecase/usecase_preservation_test.go`
- **Tests**: 22 preservation property tests — all PASS on unfixed code
- **Status**: All passing, baseline behavior captured
- **Notes**: These tests guard against regressions when fixing bugs in subsequent tasks.

### Task 3 — Fix infrastructure: Dockerfile and docker-compose
- **Files**: `Dockerfile`, `docker-compose.yml`
- **Tests**: N/A (infrastructure config)
- **Status**: Complete
- **Notes**: 3.1: Go image updated to 1.25-alpine, all 8 service binaries built, ENTRYPOINT removed. 3.2: Gateway depends_on changed to service_healthy for reservation and search.

### Task 4 — Fix concurrency: NATS client map safety
- **Files**: `pkg/nats/client.go`, `pkg/nats/client_test.go`
- **Tests**: Bug condition test passes with `-race`, preservation tests pass
- **Status**: All NATS tests passing with race detector
- **Notes**: Added sync.RWMutex to Client struct, protected all map read/write operations in CreateOrUpdateStream, CreateConsumer, ConsumeMessages, and Close.

### Task 5 — Fix concurrency: Billing penalty atomicity
- **Files**: `internal/billing/repository/repository.go`, `internal/billing/usecase/usecase.go`, `internal/billing/usecase/usecase_test.go`, `internal/billing/usecase/usecase_property_test.go`, `internal/billing/usecase/usecase_preservation_test.go`
- **Tests**: All billing tests passing (bug condition, preservation, unit)
- **Status**: Complete
- **Notes**: Added atomic `AddPenaltyAmount` repo method with SQL `penalty_amount + $1`. Refactored `ApplyPenalty` to use it. Removed dead `isBillingNotFound` function.

### Task 6 — Fix tracing: TracedClient context propagation
- **Files**: `pkg/nats/traced_client.go`
- **Tests**: All NATS tests passing (7/7)
- **Status**: Complete
- **Notes**: Replaced `context.Background()` with `tc.client.ctx` in Publish and ConsumeMessages. No interface changes needed.

### Task 7 — Fix API correctness: Gateway StreamLocation handler
- **Files**: `internal/gateway/handler/handler.go`
- **Tests**: 18 gateway handler tests passing (bug condition + preservation + unit)
- **Status**: Complete
- **Notes**: Replaced `GetPresence()` with `DetectArrival()` in StreamLocation, passing lat/lng/accuracy from request body.

### Task 8 — Fix idempotency: Reservation concurrent duplicate handling
- **Files**: `internal/reservation/usecase/usecase.go`, `internal/reservation/usecase/usecase_test.go`
- **Tests**: 17 reservation usecase tests passing
- **Status**: Complete
- **Notes**: Added unique constraint violation handling in CreateReservation — retries FindByIdempotencyKey on duplicate key error.

### Task 9 — Fix resilience: Payment retry context awareness
- **Files**: `internal/payment/usecase/usecase.go`
- **Tests**: 7 payment usecase tests passing (including context cancellation test)
- **Status**: Complete
- **Notes**: Replaced `time.Sleep` with `select` on `time.After` and `ctx.Done()`.

### Task 10 — Fix idempotency: gRPC interceptor atomicity
- **Files**: `pkg/grpcmiddleware/idempotency.go`
- **Tests**: All grpcmiddleware tests passing
- **Status**: Complete
- **Notes**: Replaced GET-then-SET with atomic SETNX + sentinel polling pattern.

### Task 11 — Fix validation: Presence coordinate bounds checking
- **Files**: `internal/presence/handler/handler.go`
- **Tests**: All presence handler tests passing (bug condition + preservation)
- **Status**: Complete
- **Notes**: Added `validateCoordinates` helper, removed unused `count` variable.

### Task 12 — Fix performance: Search cache stampede prevention
- **Files**: `go.mod`, `go.sum`, `internal/search/usecase/usecase.go`
- **Tests**: All search usecase tests passing
- **Status**: Complete
- **Notes**: Added `golang.org/x/sync/singleflight` to coalesce concurrent cache-miss DB queries.

### Task 13 — Fix security: SSRF private IP blocking
- **Files**: `pkg/httpclient/security.go`
- **Tests**: All httpclient tests passing (bug condition + preservation)
- **Status**: Complete
- **Notes**: Added URL-decode before validation, `isPrivateIP` helper, hostname resolution + private IP blocking in `buildSafeURL`.

### Task 14 — Fix resource leaks: Rate limiter stoppable cleanup goroutines
- **Files**: `pkg/middleware/ratelimit.go`, `pkg/grpcmiddleware/ratelimit.go`
- **Tests**: All rate limiter tests passing
- **Status**: Complete
- **Notes**: Added `stopCh` channel and `Stop()` method to both HTTP and gRPC rate limiter stores.

### Task 15 — Fix documentation: REVIEW.md issue #8
- **Files**: `REVIEW.md`
- **Tests**: N/A (documentation)
- **Status**: Complete
- **Notes**: Corrected issue #8 from incorrect AES-CBC claim to RSA-OAEP with SHA-256.

### Task 16 — Checkpoint: All tests pass
- **Status**: Complete — all 10 modified packages pass, all pre-existing tests pass, `go build ./cmd/...` compiles cleanly.
