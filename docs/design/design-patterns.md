# Design Patterns — ParkirPintar

Architectural and design patterns used across the system.

---

## 1. Repository Pattern

Abstract data access behind interfaces. Each service defines a `Repository` interface at the service package root, with a PostgreSQL implementation in `repository/`.

```
internal/reservation/interfaces.go         → Repository interface
internal/reservation/repository/repository.go → PostgreSQL implementation
```

**Trade-offs:** Testable (mock in unit tests), swappable storage. Adds interface overhead for simple CRUD.

---

## 2. Use Case / Interactor Pattern

Business logic lives in `usecase/usecase.go`. Orchestrates repositories, gRPC clients, and event publishing. Transport layer (gRPC handlers) delegates to use cases.

```
internal/reservation/usecase/usecase.go    → Business logic
internal/reservation/handler/grpc/handler.go → gRPC transport (thin)
```

**Trade-offs:** Single responsibility, testable with mocks. More files than putting logic in handlers.

---

## 3. Constructor Injection (No DI Framework)

All dependencies injected via constructors. Composition root in `cmd/<service>/bootstrap/bootstrap.go` wires everything. No reflection, no annotations.

```
cmd/reservation/bootstrap/bootstrap.go → Wires repo, usecase, handler, NATS
cmd/reservation/main.go                → ~30 lines, calls bootstrap
```

**Trade-offs:** Explicit, compile-time safe. Verbose for many dependencies.

---

## 4. Circuit Breaker

External calls (payment gateway) wrapped in circuit breaker. Uses `sony/gobreaker` via `pkg/circuitbreaker/`.

```
pkg/circuitbreaker/circuitbreaker.go → Generic wrapper around gobreaker
internal/payment/gateway/stub.go     → Payment gateway with circuit breaker
```

**Trade-offs:** Prevents cascade failures, self-healing. Threshold tuning needed.

---

## 5. Distributed Lock (Redis SETNX)

Prevents double-booking. Redis `SET NX EX` acquires lock on spot before reservation creation. Lua script for safe release.

```
pkg/redislock/redislock.go                    → Lock acquire/release
internal/reservation/usecase/usecase.go       → Lock before DB write
```

**Trade-offs:** Sub-ms acquisition, TTL prevents deadlocks. Redis is SPOF without Sentinel.

---

## 6. Event-Driven (NATS JetStream)

Services communicate asynchronously. Producers publish domain events; consumers subscribe independently. One consumer per stream/topic.

```
pkg/nats/client.go                              → NATS client wrapper
internal/reservation/gateway/nats/publisher.go  → Publishes spot/reservation events
internal/search/handler/nats/handler.go         → Consumes spot updates
internal/analytics/handler/nats/handler.go      → Consumes reservation lifecycle
```

**Trade-offs:** Loose coupling, resilient (persists during downtime). Eventual consistency, harder debugging.

---

## 7. CQRS-lite (Separate Read Model)

Search service maintains denormalized `spot_cache` table, updated via NATS events from reservation service. Writes go to reservation; reads from search.

```
internal/search/repository/repository.go       → spot_cache read/write
internal/search/handler/nats/handler.go        → Event projector
```

**Trade-offs:** Optimized reads, decoupled. Stale window ~1-3s, projector bugs require rebuild.

---

## 8. Idempotency Keys

All state-changing operations accept an idempotency key. Duplicate requests return the original result without side effects.

```
pkg/idempotency/idempotency.go                → Redis-based idempotency check
internal/reservation/usecase/usecase.go       → Check before create
internal/payment/usecase/usecase.go           → Check before process
```

**Trade-offs:** Safe retries, prevents duplicate charges. Redis dependency, key expiry tuning.

---

## 9. Optimistic Concurrency (Version Column)

Billing records use a `version` column. Updates include `WHERE version = $current` to detect concurrent modifications.

```
internal/billing/repository/repository.go → UPDATE ... WHERE version = $1
internal/billing/usecase/usecase.go       → Retry on version mismatch
```

**Trade-offs:** No locks held during business logic. Requires retry logic on conflict.

---

## 10. Constants Layer

Each service has `constants/` below model level. Status enums, error sentinels, and timeouts live here. No re-aliasing — use constants directly.

```
internal/reservation/constants/status.go   → ReservationStatus type + values
internal/reservation/constants/error.go    → Sentinel errors (ErrNotFound, etc.)
internal/reservation/constants/timeouts.go → Lock TTL, expiry durations
```
