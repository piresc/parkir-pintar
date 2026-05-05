# Parkir Pintar — Codebase Review

## Overall Assessment: **High Quality** with notable issues to address

Well-architected Go microservice system (7 services) for a smart parking marketplace with strong test coverage and clean separation of concerns.

---

## Strengths

- **Excellent test strategy**: 3-layer testing (testcontainers E2E, Docker Compose E2E, integration mocks) + property-based tests (`rapid`) + race condition tests + load tests. All 11 PRD scenarios covered.
- **Clean architecture**: Handler → Usecase → Repository separation across all services with interface-driven design.
- **Security fundamentals**: JWT validation with algorithm confusion prevention, CORS with explicit origins, rate limiting, SSRF protection, parameterized SQL everywhere, secret scanning in CI.
- **Observability**: OpenTelemetry tracing, structured logging with trace context injection, health endpoints with dependency checks.
- **Database design**: Proper partial unique index for double-booking prevention, CHECK constraints, 400-spot seed data, all query patterns indexed.
- **Dual CI/CD**: GitHub Actions + GitLab CI with lint, test, security scanning, and SonarCloud integration.

---

## Critical Issues (Must Fix)

### 1. Dockerfile builds wrong binary for docker-compose

`Dockerfile:27` builds `./cmd/api` but `docker-compose.yml` runs `/app/gateway`, `/app/reservation`, etc. The Docker image only contains the monolithic API binary, not the microservice binaries. **Nothing will start in docker-compose.**

### 2. NATS client data race

`pkg/nats/client.go`: The `streams` and `consumers` maps are read/written without synchronization. Concurrent calls to `CreateOrUpdateStream` or `CreateConsumer` will cause a data race.

### 3. Billing `ApplyPenalty` race condition

`billing/usecase/usecase.go`: Reads `PenaltyAmount`, modifies in memory, then writes back. Two concurrent penalty applications will lose one increment. Needs `SELECT FOR UPDATE` or atomic `UPDATE ... SET penalty_amount = penalty_amount + $1`.

### 4. Broken trace context in NATS traced client

`pkg/nats/traced_client.go`: `Publish()` uses `context.Background()` instead of propagating the caller's context, breaking distributed trace linking.

### 5. Gateway `StreamLocation` calls wrong RPC

`gateway/handler/handler.go`: The `StreamLocation` REST handler calls `presence.GetPresence` (a simple unary RPC) instead of the actual streaming `StreamLocation` RPC. Location data in the request body is ignored.

---

## High Priority Issues

| # | Location | Issue |
|---|---|---|
| 6 | `pkg/middleware/ratelimit.go` | Cleanup goroutine has no stop mechanism → goroutine leak in tests |
| 7 | `pkg/grpcmiddleware/ratelimit.go` | Same goroutine leak as HTTP rate limiter |
| 8 | `pkg/crypto/encryption.go` | Uses RSA-OAEP with SHA-256 for encryption; contains no AES code. REVIEW.md previously misidentified this as AES-CBC. |
| 9 | `pkg/grpcserver/server.go` | TLS config fields exist but are never loaded — gRPC is always plaintext |
| 10 | `pkg/grpcclient/client.go` | TLS credentials are never configured even when `TLSEnabled=true` |
| 11 | `reservation/usecase/usecase.go` | Idempotency check happens before Redis lock — concurrent duplicate requests could both pass, causing DB unique violation that isn't caught gracefully |
| 12 | `payment/usecase/usecase.go` | Retry uses `time.Sleep` instead of context-aware delay — doesn't respect cancellation |
| 13 | `payment/usecase/usecase.go` | Refund has no idempotency — duplicate refund calls could double-refund at the gateway |
| 14 | `grpcmiddleware/idempotency.go` | GET-then-SET is not atomic — use Redis Lua script or `SETNX` |
| 15 | `docker-compose.yml:83-87` | Gateway uses `service_started` instead of `service_healthy` for reservation/search |

---

## Medium Priority Issues

| # | Location | Issue |
|---|---|---|
| 16 | `reservation/worker/expiry.go` | No leader election — all instances scan for expired reservations redundantly |
| 17 | `presence/repository/repository.go` | `CleanupPresence` defined but never called; Redis streams grow unbounded |
| 18 | `presence/handler/handler.go` | No validation on lat/lng ranges (-90 to 90, -180 to 180) |
| 19 | `presence/handler/handler.go:77` | `count` variable incremented but never used |
| 20 | `search/subscriber/subscriber.go` | Invalidates all 7 cache keys on any event instead of only the affected floor |
| 21 | `search/usecase/usecase.go` | 5s TTL with no singleflight — cache stampede on expiry |
| 22 | `pkg/server/server.go` | Hardcoded 30s shutdown timeout ignores `config.Server.ShutdownTimeout` |
| 23 | `pkg/tracing/otel.go` | OTLP exporter always uses `WithInsecure()` — no TLS option |
| 24 | `pkg/httpclient/security.go` | SSRF protection doesn't block private IPs or URL-encoded bypass attempts |
| 25 | `billing/usecase/usecase.go` | `isBillingNotFound()` defined but never called |
| 26 | `pkg/nats/client.go` | `Close()` replaces maps with empty maps while active consumers hold references |

---

## Low Priority / Quality Improvements

- **Unused mock generation**: `//go:generate mockgen` directives exist but `mocks/` directories are missing. Run `make generate-mocks`.
- **Stub implementations in production entry points**: `cmd/reservation/main.go` uses stub Billing/Payment clients; `cmd/payment/main.go` uses `StubGateway`. These need real gRPC adapters for production.
- **Hardcoded constants**: 1-hour expiry, 30s lock TTL, 30s worker interval, 5s cache TTL should be configurable.
- **`go-redis/redis/v8`** is the old module path; consider upgrading to `github.com/redis/go-redis/v9`.
- **Dockerfile uses `golang:1.23`** but `go.mod` specifies Go 1.25 — version mismatch.
- **Boilerplate leftover**: `000001_init.up.sql` creates an unused `examples` table.
- **No down migrations**: Only `.up.sql` files exist.
- **Notification service is entirely stub** — all channels just log payloads.
- **`example.env`** at root is minimal (3 lines) while `config/.env.example` is comprehensive — confusing.

---

## Code Quality Scores

| Aspect | Rating | Notes |
|---|---|---|
| Architecture | **A** | Clean microservice separation, clear boundaries |
| Test Coverage | **A+** | 3-layer E2E + property-based + race + load tests |
| Error Handling | **A** | Consistent wrapping, sentinel errors, proper gRPC→HTTP mapping |
| Security | **B+** | Good fundamentals, gaps in TLS and AES mode |
| Concurrency | **B** | Most things safe, but NATS maps and billing penalties have races |
| Observability | **A** | OTEL-native, trace-aware logging, health checks |
| DevOps | **B+** | Dual CI/CD, but Dockerfile/docker-compose mismatch is blocking |
| Documentation | **A** | Excellent README, PRD, architecture docs |

---

## Recommended Priority Order

1. **Fix the Dockerfile** to build all service binaries (or use build targets) — docker-compose is currently broken
2. **Add mutex to NATS client** maps
3. **Fix billing `ApplyPenalty`** with atomic increment
4. **Fix NATS traced client** context propagation
5. **Fix gateway StreamLocation** handler
6. **Upgrade AES-CBC to AES-GCM**
7. **Implement gRPC TLS**
8. **Add leader election** to expiry worker (or use distributed lock)
9. **Replace stubs** with real gRPC client adapters in production
