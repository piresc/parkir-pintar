# Code Review Report

**Date:** 2026-05-19
**Reviewer:** Automated (Hermes Agent)
**Commit:** af12c49

---

## Confidence Level for Prod-Readiness: 72%

**Justification:** Architecture is clean with no circular deps, but 26 Go stdlib vulnerabilities (fixable by upgrading Go) and ~30 unused exported symbols indicate technical debt. No critical business logic issues found.

---

## 1. Architecture & Design 🏗️

| Dimension | Status | Details |
|-----------|--------|---------|
| Clean Code Flow | ✅ Pass | All services follow Handler → Usecase → Repository strictly. No business logic leaks in handlers, no direct DB calls from handlers. |
| Circular Dependencies | ✅ None Found | Build is clean, dependency graph is acyclic. |
| Coupling Concern | ⚠️ Note | `internal/shared/grpcerror/mapper.go` imports 4 service-specific packages (fan-in). Not a cycle, but tight coupling — if a service adds new sentinel errors, the shared mapper must be updated. |

---

## 2. Code Quality & Optimization 🧹

### Dead Code

| Location | Symbol |
|----------|--------|
| `pkg/grpcmiddleware/context_keys.go` | `UserIDFromContext`, `RoleFromContext` |
| `pkg/nats/streams.go` | `DefaultStreamConfigs`, `DefaultConsumerConfigs` |
| `pkg/logger/logger.go` | `Bool` |
| `pkg/apperror/apperror.go` | `Unauthorized` |
| `pkg/asynq/client.go` | `CancelTask`, `EnqueuePaymentHoldTimeout` |
| `pkg/middleware/` | `APIKeyAuth`, `GenerateTransactionID`, `LogResponse`, `NormalizeMsisdn` |
| `pkg/redis/redis_traced.go` | `GeoAdd`, `GeoRadius`, `SAdd`, `SIsMember`, `SRem`, `ZRem`, `Expire`, `HMSet`, `HGetAll`, `HMGet` |
| `pkg/grpcmiddleware/` | `AuthStreamInterceptor`, `LoggingStreamInterceptor`, `RecoveryStreamInterceptor`, `IdempotencyUnaryInterceptor` |
| `internal/search/handler/nats.go` | `DefaultFloorCount` |

### Duplication Level: Medium

| Pattern | Location | Recommendation |
|---------|----------|----------------|
| `mapError()` wrapper | 5× across services | Call `grpcerror.MapToGRPCError` directly |
| Idempotency check | billing & payment usecases | Extract shared utility |
| PG `23505` constraint handling | billing & payment repos | Extract to `pkg/database/errors.go` |
| Circuit breaker wrapping | 5× in `reservation/client/` | Use generic helper function |

### Formatting & Consistency

- golangci-lint: **0 issues** (with project config, 26 active linters)
- `go vet`: **clean**
- 668 raw issues all in generated protobuf or test files (deliberately excluded)

---

## 3. Security & Dependencies 🛡️

### Deprecated Dependencies

| Package | Version | Issue |
|---------|---------|-------|
| `github.com/DATA-DOG/go-sqlmock` | v1.5.2 | v1 line, unmaintained (v2 available) |
| `go.uber.org/atomic` | v1.11.0 (indirect) | Deprecated in favor of `sync/atomic` |

### Outdated Dependencies

| Package | Current | Available |
|---------|---------|-----------|
| `bsm/redislock` | v0.9.4 | v0.10.0 |
| `alicebob/miniredis/v2` | v2.37.0 | v2.38.0 |
| `buf.build/go/protovalidate` (indirect) | v0.12.0 | v1.2.0 |
| `golang.org/x/net` (indirect) | v0.52.0 | v0.53.0 (**security fix**) |

### Vulnerabilities (26 total — all Go stdlib)

**Root cause:** Go toolchain at `go1.25.0`. Fix: upgrade to `go1.25.10+`.

| Category | Count | Fix Version |
|----------|-------|-------------|
| html/template XSS | 4 | go1.25.8–1.25.10 |
| crypto/tls DoS/bypass | 4 | go1.25.2–1.25.9 |
| crypto/x509 cert validation | 5 | go1.25.2–1.25.9 |
| net/url parsing | 3 | go1.25.2–1.25.8 |
| net/mail CPU exhaustion | 3 | go1.25.2–1.25.10 |
| net/http cookie exhaustion | 1 | go1.25.2 |
| encoding/asn1 memory exhaustion | 1 | go1.25.2 |
| encoding/pem quadratic | 1 | go1.25.2 |
| os FileInfo escape | 1 | go1.25.8 |
| net panic on Windows | 1 | go1.25.10 |
| golang.org/x/net HTTP/2 infinite loop | 1 | x/net v0.53.0 |

---

## 4. Required Actions (Priority Order)

### Critical (Security)

1. **Upgrade Go to 1.25.10+** — fixes all 26 stdlib vulnerabilities in one shot
2. **Upgrade `golang.org/x/net`** to v0.53.0 — fixes HTTP/2 infinite loop (GO-2026-4918)

### High (Code Quality)

3. **Remove dead code** — unexport or delete ~30 unused symbols in `pkg/` to reduce maintenance surface
4. **Extract shared idempotency helper** — deduplicate billing/payment usecase pattern

### Medium (Cleanup)

5. **Eliminate `mapError()` wrappers** — call `grpcerror.MapToGRPCError` directly
6. **Extract PG constraint error helper** to `pkg/database/errors.go`
7. **Generic circuit breaker wrapper** for `reservation/client/`

### Low (Nice-to-have)

8. **Upgrade `go-sqlmock`** to v2 (API changes required, test-only dep)
9. **Upgrade `bsm/redislock`** to v0.10.0
10. **Decouple `grpcerror/mapper.go`** from service-specific imports (use error interfaces)

---

## 5. What's Working Well ✅

- Clean architecture with strict layer separation
- Comprehensive test coverage (43 packages passing)
- OpenTelemetry instrumentation (traces, metrics, logs)
- Circuit breaker + distributed locking patterns
- Proper graceful shutdown across all services
- gRPC with health checks and middleware chain
- NATS JetStream for async event processing
- golangci-lint v2 with 26 active linters, zero violations
