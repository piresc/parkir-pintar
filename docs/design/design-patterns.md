# Design Patterns — ParkirPintar

This document catalogs the architectural and design patterns used across the ParkirPintar system, with rationale, implementation locations, and trade-offs.

---

## 1. Repository Pattern

### Problem
Business logic becomes tightly coupled to database implementation details (SQL queries, connection management, transaction handling), making it difficult to test and swap storage backends.

### Solution
Abstract data access behind interfaces. Each aggregate root gets a repository interface defined in the domain layer, with a concrete PostgreSQL implementation in the infrastructure layer.

### Implementation Location
- `internal/reservation/domain/repository.go` — interface definition
- `internal/reservation/infra/postgres/reservation_repo.go` — PostgreSQL implementation
- Same pattern in billing, payment, presence services

### Code Example

```go
// domain/repository.go
package domain

import "context"

type ReservationRepository interface {
    Create(ctx context.Context, r *Reservation) error
    GetByID(ctx context.Context, id string) (*Reservation, error)
    ListByDriver(ctx context.Context, driverID string, opts ListOpts) ([]*Reservation, error)
    Update(ctx context.Context, r *Reservation) error
}

// infra/postgres/reservation_repo.go
package postgres

type reservationRepo struct {
    db *pgxpool.Pool
}

func NewReservationRepo(db *pgxpool.Pool) domain.ReservationRepository {
    return &reservationRepo{db: db}
}

func (r *reservationRepo) Create(ctx context.Context, res *domain.Reservation) error {
    query := `INSERT INTO reservations (id, driver_id, spot_id, start_time, end_time, status, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
    _, err := r.db.Exec(ctx, query,
        res.ID, res.DriverID, res.SpotID, res.StartTime, res.EndTime, res.Status, res.CreatedAt)
    return err
}
```

### Trade-offs
- ✅ Testable — mock repository in unit tests without DB
- ✅ Swappable — can add Redis caching layer via decorator
- ❌ Abstraction overhead — simple CRUD still needs interface + impl
- ❌ Complex queries may leak through abstraction boundaries

---

## 2. Use Case / Interactor Pattern

### Problem
Business logic scattered across HTTP handlers makes it untestable, unreusable, and tightly coupled to transport layer.

### Solution
Encapsulate each business operation in a dedicated use case struct with a single `Execute` method. Use cases orchestrate repositories, external services, and domain logic.

### Implementation Location
- `internal/reservation/usecase/create_reservation.go`
- `internal/reservation/usecase/cancel_reservation.go`
- `internal/billing/usecase/calculate_charge.go`
- `internal/payment/usecase/process_payment.go`

### Code Example

```go
// usecase/create_reservation.go
package usecase

type CreateReservation struct {
    repo   domain.ReservationRepository
    locker domain.SpotLocker
    events domain.EventPublisher
}

func NewCreateReservation(
    repo domain.ReservationRepository,
    locker domain.SpotLocker,
    events domain.EventPublisher,
) *CreateReservation {
    return &CreateReservation{repo: repo, locker: locker, events: events}
}

type CreateReservationInput struct {
    DriverID  string
    SpotID    string
    StartTime time.Time
    EndTime   time.Time
    IdempKey  string
}

func (uc *CreateReservation) Execute(ctx context.Context, in CreateReservationInput) (*domain.Reservation, error) {
    // 1. Acquire distributed lock on spot+time
    lock, err := uc.locker.AcquireSpotLock(ctx, in.SpotID, in.StartTime, in.EndTime)
    if err != nil {
        return nil, domain.ErrSpotUnavailable
    }
    defer lock.Release(ctx)

    // 2. Verify no overlapping reservation
    if err := uc.repo.CheckOverlap(ctx, in.SpotID, in.StartTime, in.EndTime); err != nil {
        return nil, domain.ErrSpotUnavailable
    }

    // 3. Create reservation
    res := domain.NewReservation(in.DriverID, in.SpotID, in.StartTime, in.EndTime)
    if err := uc.repo.Create(ctx, res); err != nil {
        return nil, err
    }

    // 4. Publish event
    uc.events.Publish(ctx, domain.ReservationCreatedEvent{ReservationID: res.ID})

    return res, nil
}
```

### Trade-offs
- ✅ Single Responsibility — one use case, one business operation
- ✅ Testable — inject mocks for all dependencies
- ✅ Reusable — same use case callable from gRPC, REST, or CLI
- ❌ More files/structs than putting logic in handlers
- ❌ Cross-cutting concerns (logging, metrics) need middleware or explicit calls

---

## 3. Dependency Injection (Constructor Injection)

### Problem
Hard-coded dependencies make testing difficult and create tight coupling between layers.

### Solution
All dependencies are injected via constructor functions. No DI framework — the composition root (`cmd/*/main.go`) wires everything manually. This keeps the dependency graph explicit and compile-time verified.

### Implementation Location
- `cmd/reservation/main.go` — composition root for reservation service
- `cmd/billing/main.go` — composition root for billing service
- All service `main.go` files follow the same pattern

### Code Example

```go
// cmd/reservation/main.go
func main() {
    cfg := config.Load()

    // Infrastructure
    db := postgres.NewPool(cfg.DatabaseURL)
    redisClient := redis.NewClient(cfg.RedisURL)
    natsConn := nats.Connect(cfg.NatsURL)

    // Repositories
    reservationRepo := postgres.NewReservationRepo(db)

    // Domain services
    spotLocker := redislock.NewSpotLocker(redisClient)
    eventPub := natsevents.NewPublisher(natsConn)

    // Use cases
    createReservation := usecase.NewCreateReservation(reservationRepo, spotLocker, eventPub)
    cancelReservation := usecase.NewCancelReservation(reservationRepo, eventPub)

    // Transport
    grpcServer := grpc.NewServer(createReservation, cancelReservation)
    grpcServer.Serve(cfg.GRPCPort)
}
```

### Trade-offs
- ✅ Explicit — all wiring visible in one place
- ✅ Compile-time safety — missing dependency = compile error
- ✅ No magic — no reflection, no annotations, no framework lock-in
- ❌ Verbose composition root for many dependencies
- ❌ Adding a dependency requires updating constructor + all call sites

---

## 4. Circuit Breaker

### Problem
External service failures (payment gateway, third-party APIs) can cascade through the system, exhausting connection pools and causing timeouts across all requests.

### Solution
Wrap external calls in a circuit breaker that tracks failures and opens the circuit (fast-fails) when error rate exceeds threshold. After a cooldown period, it allows a probe request to test recovery.

### Implementation Location
- `pkg/circuitbreaker/breaker.go` — generic circuit breaker implementation
- `internal/payment/infra/midtrans/client.go` — wraps Midtrans API calls
- `internal/search/infra/external/geocoding.go` — wraps geocoding API

### Code Example

```go
// pkg/circuitbreaker/breaker.go
package circuitbreaker

type State int

const (
    Closed State = iota
    Open
    HalfOpen
)

type Breaker struct {
    mu            sync.Mutex
    state         State
    failCount     int
    threshold     int
    cooldown      time.Duration
    lastFailure   time.Time
    onStateChange func(from, to State)
}

func New(threshold int, cooldown time.Duration) *Breaker {
    return &Breaker{threshold: threshold, cooldown: cooldown, state: Closed}
}

func (b *Breaker) Execute(fn func() error) error {
    b.mu.Lock()
    if b.state == Open {
        if time.Since(b.lastFailure) > b.cooldown {
            b.state = HalfOpen
        } else {
            b.mu.Unlock()
            return ErrCircuitOpen
        }
    }
    b.mu.Unlock()

    err := fn()

    b.mu.Lock()
    defer b.mu.Unlock()
    if err != nil {
        b.failCount++
        b.lastFailure = time.Now()
        if b.failCount >= b.threshold {
            b.state = Open
        }
        return err
    }

    b.failCount = 0
    b.state = Closed
    return nil
}

// Usage in payment client:
// breaker := circuitbreaker.New(5, 30*time.Second)
// err := breaker.Execute(func() error {
//     return midtransClient.ChargeCard(ctx, req)
// })
```

### Trade-offs
- ✅ Prevents cascade failures
- ✅ Fast-fail gives immediate feedback to callers
- ✅ Self-healing — automatically retries after cooldown
- ❌ Adds latency tracking overhead
- ❌ Threshold tuning requires production observation
- ❌ Half-open state can cause intermittent failures during recovery

---

## 5. Distributed Lock (Redis SETNX)

### Problem
Multiple concurrent reservation requests for the same parking spot and time window can create double-bookings if only relying on database constraints.

### Solution
Use Redis `SET NX EX` (atomic set-if-not-exists with expiry) to acquire a distributed lock on the spot+time combination before checking availability and creating the reservation. The lock has a TTL to prevent deadlocks if the holder crashes.

### Implementation Location
- `pkg/redislock/spot_lock.go` — distributed lock implementation
- `internal/reservation/usecase/create_reservation.go` — lock acquisition before DB write

### Code Example

```go
// pkg/redislock/spot_lock.go
package redislock

type SpotLocker struct {
    client *redis.Client
    ttl    time.Duration
}

func NewSpotLocker(client *redis.Client) *SpotLocker {
    return &SpotLocker{client: client, ttl: 10 * time.Second}
}

type Lock struct {
    client *redis.Client
    key    string
    value  string
}

func (sl *SpotLocker) AcquireSpotLock(ctx context.Context, spotID string, start, end time.Time) (*Lock, error) {
    // Key encodes spot + time window (hourly granularity)
    key := fmt.Sprintf("lock:spot:%s:%s:%s", spotID, start.Format("2006010215"), end.Format("2006010215"))
    value := uuid.NewString() // unique owner ID for safe release

    ok, err := sl.client.SetNX(ctx, key, value, sl.ttl).Result()
    if err != nil {
        return nil, fmt.Errorf("redis lock: %w", err)
    }
    if !ok {
        return nil, domain.ErrSpotUnavailable
    }

    return &Lock{client: sl.client, key: key, value: value}, nil
}

func (l *Lock) Release(ctx context.Context) error {
    // Lua script ensures we only delete our own lock
    script := `if redis.call("get", KEYS[1]) == ARGV[1] then
                   return redis.call("del", KEYS[1])
               else
                   return 0
               end`
    return l.client.Eval(ctx, script, []string{l.key}, l.value).Err()
}
```

### Trade-offs
- ✅ Prevents double-booking at the application layer
- ✅ Sub-millisecond lock acquisition
- ✅ TTL prevents permanent deadlocks
- ❌ Redis SPOF — need Redis Sentinel/Cluster for HA
- ❌ Clock skew in distributed Redis can cause issues (Redlock debate)
- ❌ Lock TTL must be longer than max operation time

---

## 6. Event-Driven Architecture (NATS Pub/Sub)

### Problem
Synchronous inter-service calls create tight coupling, cascade failures, and make it impossible to add new consumers without modifying producers.

### Solution
Services communicate asynchronously via NATS JetStream. Producers publish domain events; consumers subscribe independently. JetStream provides at-least-once delivery with consumer acknowledgment.

### Implementation Location
- `pkg/events/publisher.go` — NATS event publisher
- `pkg/events/subscriber.go` — NATS event subscriber with retry
- `internal/reservation/infra/nats/publisher.go` — reservation events
- `internal/search/infra/nats/subscriber.go` — search service consumer
- `internal/billing/infra/nats/subscriber.go` — billing service consumer

### Code Example

```go
// pkg/events/publisher.go
package events

type Publisher struct {
    js nats.JetStreamContext
}

func NewPublisher(js nats.JetStreamContext) *Publisher {
    return &Publisher{js: js}
}

func (p *Publisher) Publish(ctx context.Context, event Event) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    subject := fmt.Sprintf("parkir.%s.%s", event.AggregateType(), event.EventType())
    _, err = p.js.Publish(subject, data, nats.MsgId(event.ID())) // dedup by event ID
    return err
}

// pkg/events/subscriber.go
func (s *Subscriber) Subscribe(subject, consumer string, handler Handler) error {
    sub, err := s.js.PullSubscribe(subject, consumer, nats.AckWait(30*time.Second))
    if err != nil {
        return err
    }
    go func() {
        for {
            msgs, _ := sub.Fetch(10, nats.MaxWait(5*time.Second))
            for _, msg := range msgs {
                if err := handler(context.Background(), msg.Data); err != nil {
                    msg.Nak() // will be redelivered
                    continue
                }
                msg.Ack()
            }
        }
    }()
    return nil
}
```

### Trade-offs
- ✅ Loose coupling — producers don't know about consumers
- ✅ Scalable — add consumers without modifying producers
- ✅ Resilient — JetStream persists messages during consumer downtime
- ❌ Eventual consistency — consumers may lag behind
- ❌ Debugging distributed flows is harder than synchronous calls
- ❌ Message ordering guarantees require careful subject design

---

## 7. CQRS-lite (Separate Read Model)

### Problem
The search service needs to query across multiple aggregates (spots, reservations, pricing) with low latency, but the source-of-truth data lives in separate service databases.

### Solution
Maintain a denormalized read model (`spot_read_model` table) in the search service, updated asynchronously via NATS events. Writes go to source services; reads come from the optimized read model.

### Implementation Location
- `internal/search/domain/read_model.go` — read model entity
- `internal/search/infra/postgres/read_model_repo.go` — read model persistence
- `internal/search/infra/nats/projector.go` — event projector that updates read model
- `internal/search/usecase/search_spots.go` — queries read model

### Code Example

```go
// internal/search/infra/nats/projector.go
package nats

type Projector struct {
    repo domain.ReadModelRepository
}

func (p *Projector) HandleReservationCreated(ctx context.Context, data []byte) error {
    var event events.ReservationCreated
    if err := json.Unmarshal(data, &event); err != nil {
        return err
    }
    return p.repo.MarkSpotReserved(ctx, event.SpotID, event.StartTime, event.EndTime)
}

func (p *Projector) HandleReservationCancelled(ctx context.Context, data []byte) error {
    var event events.ReservationCancelled
    if err := json.Unmarshal(data, &event); err != nil {
        return err
    }
    return p.repo.MarkSpotAvailable(ctx, event.SpotID, event.StartTime, event.EndTime)
}

// internal/search/usecase/search_spots.go
func (uc *SearchSpots) Execute(ctx context.Context, in SearchInput) ([]domain.SpotView, error) {
    return uc.readRepo.FindAvailable(ctx, in.Location, in.Radius, in.StartTime, in.EndTime)
}
```

### Trade-offs
- ✅ Optimized read performance — denormalized, indexed for search queries
- ✅ Decoupled — search service doesn't call reservation service synchronously
- ✅ Scalable — read model can be replicated independently
- ❌ Eventual consistency — stale window of ~1-3 seconds
- ❌ Projector logic must handle all relevant events correctly
- ❌ Read model rebuild required if projector has bugs

---

## 8. Singleflight (Concurrent Cache Miss Deduplication)

### Problem
When a popular parking spot's cache entry expires, hundreds of concurrent requests simultaneously hit the database to reload it, causing a "thundering herd" / cache stampede.

### Solution
Use `golang.org/x/sync/singleflight` to deduplicate concurrent calls for the same key. Only one goroutine fetches from DB; others wait and share the result.

### Implementation Location
- `internal/search/infra/cache/spot_cache.go` — singleflight wrapper around cache reads
- `internal/reservation/infra/cache/availability_cache.go` — availability lookups

### Code Example

```go
// internal/search/infra/cache/spot_cache.go
package cache

import "golang.org/x/sync/singleflight"

type SpotCache struct {
    redis  *redis.Client
    repo   domain.ReadModelRepository
    group  singleflight.Group
    ttl    time.Duration
}

func (c *SpotCache) GetSpot(ctx context.Context, spotID string) (*domain.SpotView, error) {
    // Try cache first
    cached, err := c.redis.Get(ctx, "spot:"+spotID).Bytes()
    if err == nil {
        var spot domain.SpotView
        json.Unmarshal(cached, &spot)
        return &spot, nil
    }

    // Singleflight: deduplicate concurrent DB calls for same spot
    result, err, _ := c.group.Do(spotID, func() (interface{}, error) {
        spot, err := c.repo.GetByID(ctx, spotID)
        if err != nil {
            return nil, err
        }
        // Repopulate cache
        data, _ := json.Marshal(spot)
        c.redis.Set(ctx, "spot:"+spotID, data, c.ttl)
        return spot, nil
    })
    if err != nil {
        return nil, err
    }
    return result.(*domain.SpotView), nil
}
```

### Trade-offs
- ✅ Eliminates thundering herd on cache miss
- ✅ Zero additional infrastructure — stdlib extension
- ✅ Transparent to callers
- ❌ All waiters get same error if the single call fails
- ❌ Slight latency increase for waiters (blocked on leader)
- ❌ Only helps for concurrent requests; sequential misses still hit DB

---

## 9. Idempotency Pattern

### Problem
Network retries, client timeouts, and at-least-once message delivery can cause duplicate operations (double charges, duplicate reservations).

### Solution
All mutation endpoints accept an `Idempotency-Key` header. Before processing, check if the key has been seen; if so, return the cached response. Store the key + response in Redis with a 24h TTL.

### Implementation Location
- `pkg/idempotency/middleware.go` — HTTP/gRPC middleware
- `pkg/idempotency/store.go` — Redis-backed idempotency store
- Applied to: reservation creation, payment processing, billing charge creation

### Code Example

```go
// pkg/idempotency/middleware.go
package idempotency

type Store struct {
    redis *redis.Client
    ttl   time.Duration
}

func NewStore(redis *redis.Client) *Store {
    return &Store{redis: redis, ttl: 24 * time.Hour}
}

type CachedResponse struct {
    StatusCode int    `json:"status_code"`
    Body       []byte `json:"body"`
}

func (s *Store) Check(ctx context.Context, key string) (*CachedResponse, error) {
    data, err := s.redis.Get(ctx, "idemp:"+key).Bytes()
    if err == redis.Nil {
        return nil, nil // not seen before
    }
    if err != nil {
        return nil, err
    }
    var resp CachedResponse
    json.Unmarshal(data, &resp)
    return &resp, nil
}

func (s *Store) Save(ctx context.Context, key string, resp CachedResponse) error {
    data, _ := json.Marshal(resp)
    return s.redis.Set(ctx, "idemp:"+key, data, s.ttl).Err()
}

// gRPC interceptor usage
func IdempotencyInterceptor(store *Store) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        key := extractIdempotencyKey(ctx)
        if key == "" {
            return handler(ctx, req)
        }
        if cached, _ := store.Check(ctx, key); cached != nil {
            return cached, nil
        }
        resp, err := handler(ctx, req)
        if err == nil {
            store.Save(ctx, key, resp)
        }
        return resp, err
    }
}
```

### Trade-offs
- ✅ Safe retries — clients can retry without side effects
- ✅ Works across network boundaries (load balancers, proxies)
- ✅ Simple implementation with Redis
- ❌ Storage cost — one Redis key per mutation for 24h
- ❌ Key generation responsibility on client
- ❌ Doesn't help if client generates new key on each retry (client bug)

---

## 10. Graceful Degradation

### Problem
Redis cache failure should not bring down the entire system. The application must continue serving requests (with higher latency) when non-critical components fail.

### Solution
Cache operations are wrapped in error handlers that log the failure and fall through to the database. The system operates in degraded mode rather than failing completely.

### Implementation Location
- `internal/search/infra/cache/spot_cache.go` — cache miss falls through to DB
- `internal/reservation/infra/cache/availability_cache.go` — same pattern
- `pkg/circuitbreaker/` — circuit breaker on external services

### Code Example

```go
// internal/search/infra/cache/spot_cache.go
func (c *SpotCache) GetAvailableSpots(ctx context.Context, query SearchQuery) ([]domain.SpotView, error) {
    cacheKey := query.CacheKey()

    // Attempt cache read — non-fatal on failure
    cached, err := c.redis.Get(ctx, cacheKey).Bytes()
    if err == nil {
        var spots []domain.SpotView
        if json.Unmarshal(cached, &spots) == nil {
            cacheHits.Inc()
            return spots, nil
        }
    }
    if err != nil && err != redis.Nil {
        // Redis is down — log and continue to DB
        c.logger.Warn("cache read failed, falling through to DB",
            "error", err, "key", cacheKey)
        cacheFallbacks.Inc()
    }

    // Fall through to database
    spots, err := c.repo.Search(ctx, query)
    if err != nil {
        return nil, err
    }

    // Attempt cache write — non-fatal on failure
    if data, err := json.Marshal(spots); err == nil {
        if err := c.redis.Set(ctx, cacheKey, data, c.ttl).Err(); err != nil {
            c.logger.Warn("cache write failed", "error", err, "key", cacheKey)
        }
    }

    return spots, nil
}
```

### Trade-offs
- ✅ System stays available during partial failures
- ✅ Users experience higher latency instead of errors
- ✅ Self-healing — cache recovers automatically when Redis returns
- ❌ Database load spikes during cache outage
- ❌ Must ensure DB can handle full load without cache
- ❌ Degraded mode may not meet SLA latency targets

---

## 11. Health Check Pattern

### Problem
Kubernetes needs to know if a pod is alive (not deadlocked) and ready to serve traffic (dependencies connected). Without proper health checks, traffic routes to unhealthy pods.

### Solution
Implement two health endpoints: `/healthz` (liveness — is the process alive?) and `/readyz` (readiness — can it serve traffic?). Readiness checks verify database and Redis connectivity.

### Implementation Location
- `pkg/health/handler.go` — health check HTTP handler
- `pkg/health/checker.go` — dependency checkers (DB, Redis, NATS)
- `deploy/helm/templates/deployment.yaml` — Kubernetes probe configuration

### Code Example

```go
// pkg/health/handler.go
package health

type Checker interface {
    Check(ctx context.Context) error
    Name() string
}

type Handler struct {
    checkers []Checker
}

func NewHandler(checkers ...Checker) *Handler {
    return &Handler{checkers: checkers}
}

// Liveness: is the process alive and not deadlocked?
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"alive"}`))
}

// Readiness: can this instance serve traffic?
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
    defer cancel()

    results := make(map[string]string)
    healthy := true

    for _, c := range h.checkers {
        if err := c.Check(ctx); err != nil {
            results[c.Name()] = err.Error()
            healthy = false
        } else {
            results[c.Name()] = "ok"
        }
    }

    status := http.StatusOK
    if !healthy {
        status = http.StatusServiceUnavailable
    }

    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": healthy,
        "checks": results,
    })
}

// pkg/health/postgres_checker.go
type PostgresChecker struct{ db *pgxpool.Pool }

func (c *PostgresChecker) Check(ctx context.Context) error {
    return c.db.Ping(ctx)
}
func (c *PostgresChecker) Name() string { return "postgres" }
```

### Trade-offs
- ✅ Kubernetes routes traffic only to healthy pods
- ✅ Automatic restart of deadlocked pods (liveness failure)
- ✅ Graceful rolling deploys (readiness gates)
- ❌ Overly aggressive probes can cause restart loops
- ❌ Dependency checks add slight overhead per probe interval
- ❌ Must tune timeouts carefully (probe timeout < check timeout)

---

## 12. Strangler Fig Pattern

### Problem
Migrating from a monolithic shared database to per-service schemas cannot happen in a single big-bang release. The system must continue operating during the migration.

### Solution
Incrementally migrate tables to service-owned schemas. During transition, services read from both old and new schemas (with feature flags). Once migration is verified, cut over writes and drop the old tables.

### Implementation Location
- `migrations/` — per-service migration directories
- `internal/*/infra/postgres/dual_reader.go` — reads from both schemas during migration
- `internal/config/features.go` — feature flags controlling migration phases

### Code Example

```go
// internal/reservation/infra/postgres/dual_reader.go
package postgres

type DualReservationRepo struct {
    newRepo    *reservationRepo  // service-owned schema
    legacyRepo *legacyRepo       // shared monolith schema
    features   *config.Features
}

func (d *DualReservationRepo) GetByID(ctx context.Context, id string) (*domain.Reservation, error) {
    if d.features.IsEnabled("reservation.use_new_schema") {
        res, err := d.newRepo.GetByID(ctx, id)
        if err == nil {
            return res, nil
        }
        // Fallback to legacy during migration
        if d.features.IsEnabled("reservation.legacy_fallback") {
            return d.legacyRepo.GetByID(ctx, id)
        }
        return nil, err
    }
    return d.legacyRepo.GetByID(ctx, id)
}

// Migration phases:
// Phase 1: Read legacy, write both (dual-write)
// Phase 2: Read new, fallback legacy (dual-read)
// Phase 3: Read new only, legacy deprecated
// Phase 4: Drop legacy tables
```

### Trade-offs
- ✅ Zero-downtime migration
- ✅ Rollback possible at each phase
- ✅ Incremental verification reduces risk
- ❌ Dual-write complexity and potential inconsistency
- ❌ Feature flag management overhead
- ❌ Temporary code that must be cleaned up after migration

---

## Pattern Interaction Map

```
Client Request
    │
    ▼
[Gateway] ──── Health Check Pattern
    │
    ▼
[Idempotency Middleware] ──── Idempotency Pattern
    │
    ▼
[Use Case / Interactor] ──── Use Case Pattern + DI
    │
    ├──► [Repository] ──── Repository Pattern
    │        │
    │        ▼
    │    [Cache Layer] ──── Graceful Degradation + Singleflight
    │        │
    │        ▼
    │    [PostgreSQL]
    │
    ├──► [Distributed Lock] ──── Redis SETNX
    │
    ├──► [External Service] ──── Circuit Breaker
    │
    └──► [Event Publisher] ──── Event-Driven Architecture
             │
             ▼
         [NATS JetStream]
             │
             ▼
         [Search Projector] ──── CQRS-lite
             │
             ▼
         [Read Model DB]
```
