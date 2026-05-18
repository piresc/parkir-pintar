# ParkirPintar — Smart Parking Reservation System

> **Solution Development Assessment 2026**

A production-grade smart parking backend managing a centralized parking area with real-time spot reservation, automated billing, and QRIS payment processing. Built with Go microservices communicating via gRPC, backed by PostgreSQL, Redis, and Asynq task queues.

**Live Demo:** https://parkir-pintar.piresc.dev

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [High-Level Design (HLD)](#high-level-design-hld)
- [Low-Level Design (LLD)](#low-level-design-lld)
- [Entity Relationship Diagram](#entity-relationship-diagram)
- [State Machines](#state-machines)
- [Assumptions](#assumptions)
- [Tech Stack](#tech-stack)
- [3rd-Party Libraries & Justification](#3rd-party-libraries--justification)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Testing](#testing)
- [CI/CD Pipeline](#cicd-pipeline)
- [Monitoring & Observability](#monitoring--observability)
- [Design Decisions](#design-decisions)
- [License](#license)

---

## Architecture Overview

ParkirPintar uses a microservices architecture with 5 Go services communicating via gRPC (synchronous) and NATS JetStream (asynchronous events). The API Gateway exposes REST endpoints and transcodes to internal gRPC calls.

```mermaid
graph TB
    Client[React SPA / Mobile Client] -->|REST| GW[API Gateway<br/>:8080]

    subgraph ParkirPintar Microservices
        GW -->|gRPC| SEARCH[Search Service<br/>:9092]
        GW -->|gRPC| RESERVE[Reservation Service<br/>:9091]
        GW -->|gRPC| PAYMENT[Payment Service<br/>:9094]

        RESERVE -->|gRPC| BILLING[Billing Service<br/>:9093]
        RESERVE -->|gRPC| PAYMENT
    end

    subgraph Event Streaming
        NATS[NATS JetStream]
    end

    RESERVE -->|publish spot-updated| NATS
    NATS -->|spot-updated events| SEARCH
    PAYMENT -->|publish payment results| NATS
    NATS -->|payment success/failed| RESERVE
    RESERVE -->|publish lifecycle events| NATS

    subgraph Data Layer
        PG[(PostgreSQL)]
        RD[(Redis)]
    end

    subgraph Background Workers
        ASYNQ[Asynq Workers<br/>Reservation Expiry<br/>Payment Hold Timeout]
    end

    SEARCH --> PG
    SEARCH --> RD
    RESERVE --> PG
    RESERVE --> RD
    RESERVE --> ASYNQ
    ASYNQ --> RD
    BILLING --> PG
    PAYMENT --> PG

    subgraph Observability
        ALLOY[Grafana Alloy] --> TEMPO[Tempo<br/>Traces]
        ALLOY --> PROM[Prometheus<br/>Metrics]
        ALLOY --> LOKI[Loki<br/>Logs]
        GRAFANA[Grafana Dashboard]
        GRAFANA --> TEMPO
        GRAFANA --> PROM
        GRAFANA --> LOKI
    end
```

### Service Responsibilities

| Service | gRPC Port | Responsibility |
|---------|-----------|----------------|
| **Gateway** | 8080 (REST) | REST→gRPC transcoding, JWT auth, rate limiting, request routing |
| **Search** | 9092 | Spot availability queries, filtering by type/zone/floor |
| **Reservation** | 9091 | Full reservation lifecycle, spot locking, state transitions |
| **Billing** | 9093 | Fee calculation (hourly + overnight + penalties) |
| **Payment** | 9094 | QRIS payment processing, refunds, payment status |

---

## High-Level Design (HLD)

### System Context

```mermaid
C4Context
    title ParkirPintar System Context

    Person(driver, "Driver", "Parks vehicle via mobile app")
    System(parkir, "ParkirPintar", "Smart parking reservation & billing system")
    System_Ext(qris, "QRIS Provider", "Payment gateway (stub)")
    SystemDb(pg, "PostgreSQL", "Persistent storage")
    SystemDb(redis, "Redis", "Cache, locks, task queue")

    Rel(driver, parkir, "Reserves spots, pays fees", "HTTPS/REST")
    Rel(parkir, qris, "Processes payments", "QRIS API")
    Rel(parkir, pg, "Reads/writes data", "TCP")
    Rel(parkir, redis, "Caching, locking, tasks", "TCP")
```

### Key Flows

**Reservation Flow (Happy Path):**

```mermaid
sequenceDiagram
    participant D as Driver
    participant GW as Gateway
    participant R as Reservation
    participant B as Billing
    participant P as Payment
    participant Q as Asynq Queue

    D->>GW: POST /api/reservations (spot_id, vehicle_type)
    GW->>R: CreateReservation (gRPC)
    R->>R: Acquire distributed lock on spot
    R->>R: Validate spot available + state transition
    R->>R: Create reservation (status: waiting_payment)
    R->>Q: Enqueue payment hold timeout (10 min)
    R-->>GW: Reservation created
    GW-->>D: 201 + reservation_id

    D->>GW: POST /api/reservations/:id/confirm
    GW->>R: ConfirmReservation (gRPC)
    R->>B: CalculateFee (booking fee)
    B-->>R: Fee: 5000 IDR
    R->>P: ProcessQRIS (booking fee)
    P-->>R: Payment success
    R->>R: Status → confirmed
    R->>Q: Enqueue reservation expiry (60 min)
    R-->>GW: Confirmed
    GW-->>D: 200 OK

    D->>GW: POST /api/reservations/:id/checkin
    GW->>R: CheckIn (gRPC)
    R->>R: Status → checked_in, record check_in_time
    R-->>D: 200 OK

    D->>GW: POST /api/reservations/:id/checkout
    GW->>R: CheckOut (gRPC)
    R->>R: Status → checked_out, record check_out_time
    R-->>D: 200 OK

    D->>GW: POST /api/reservations/:id/complete
    GW->>R: CompleteCheckout (gRPC)
    R->>B: CalculateFee (session: hourly + overnight)
    B-->>R: Total session fee
    R->>P: ProcessQRIS (session charges)
    P-->>R: Payment success
    R->>R: Status → completed, release spot
    R-->>D: 200 OK + final bill
```

### Payment Two-Moment Pattern

Payments are split into two moments to minimize driver risk:

1. **Booking Fee** (at confirmation): Fixed 5,000 IDR — confirms intent, reserves the spot
2. **Session Charges** (at checkout completion): Hourly rate × duration + overnight fees — charged only for actual usage

This ensures drivers aren't overcharged if they cancel early, and the system collects a non-refundable booking fee to prevent frivolous reservations.

---

## Low-Level Design (LLD)

### Package Architecture

```
parkir-pintar/
├── cmd/                          # Service entry points
│   ├── gateway/main.go           # REST API gateway
│   ├── search/main.go            # Search service
│   ├── reservation/main.go       # Reservation + Asynq workers
│   ├── billing/main.go           # Billing service
│   └── payment/main.go           # Payment service
├── internal/                     # Domain logic (not importable externally)
│   ├── gateway/handler/          # REST handlers, routing
│   ├── search/                   # handler, usecase, repository, model, sync
│   ├── reservation/              # handler, usecase, repository, model, worker, client
│   ├── billing/                  # handler, usecase, repository, model
│   ├── payment/                  # handler, usecase, gateway (stub)
│   └── analytics/                # usecase, repository (peak hours, occupancy)
├── pkg/                          # Shared libraries
│   ├── asynq/                    # Task queue (client, server, handlers, tasks)
│   ├── nats/                     # JetStream client, publisher, streams, constants
│   ├── pricing/                  # Pricing engine (hourly, overnight, penalties)
│   ├── redislock/                # Distributed locking
│   ├── circuitbreaker/           # Circuit breaker for gRPC calls
│   ├── telemetry/                # Unified OTLP (traces, metrics, logs)
│   ├── tracing/                  # OpenTelemetry tracer
│   ├── metrics/                  # Prometheus + OTLP metrics
│   ├── grpcserver/               # gRPC server factory
│   ├── grpcclient/               # gRPC client factory with interceptors
│   ├── grpcmiddleware/           # Auth, rate limit, logging, tracing, recovery
│   ├── config/                   # Env-based configuration
│   ├── database/                 # PostgreSQL client with tracing
│   ├── redis/                    # Redis client wrapper
│   ├── auth/                     # JWT validation
│   ├── health/                   # Health check endpoints
│   ├── apperror/                 # Domain error types
│   └── server/                   # Graceful shutdown manager
├── proto/                        # Protocol Buffer definitions
│   ├── reservation/v1/
│   ├── billing/v1/
│   ├── payment/v1/
│   └── search/v1/
├── db/migrations/                # PostgreSQL migrations (golang-migrate)
├── deploy/                       # Docker Compose, monitoring configs
├── tests/                        # E2E and integration tests
└── config/                       # Environment files
```

### Pricing Engine (`pkg/pricing`)

```go
// Constants
BookingFee    = 5000  // IDR, fixed per reservation
HourlyRate    = 5000  // IDR per hour (prorated per minute)
OvernightFee  = 20000 // IDR per midnight crossed

// CalculateSessionFee computes: hourly_rate × hours + overnight_fee × midnights_crossed
// CalculateCancellationFee returns booking fee as non-refundable penalty
```

Overnight calculation counts actual midnight crossings (00:00 boundaries in WIB timezone) between check-in and check-out, charging 20,000 IDR per midnight crossed. Multi-night stays are charged proportionally (e.g., 2 midnights = 40,000 IDR).

### Distributed Locking Strategy

Spot reservation uses Redis-based distributed locks to prevent double-booking:

```mermaid
sequenceDiagram
    participant R1 as Request 1
    participant R2 as Request 2
    participant Redis as Redis Lock
    participant DB as PostgreSQL

    R1->>Redis: LOCK spot-123 (TTL: 12min)
    Redis-->>R1: OK (acquired)
    R2->>Redis: LOCK spot-123
    Redis-->>R2: FAIL (already locked)
    R2-->>R2: Return "spot unavailable"
    R1->>DB: UPDATE spot SET status='reserved'
    R1->>DB: INSERT reservation
    R1->>Redis: UNLOCK spot-123
```

### Background Task Processing

```mermaid
graph LR
    subgraph Asynq Task Queue
        RC[Reservation Created] -->|10 min delay| PHT[Payment Hold Timeout]
        CONF[Reservation Confirmed] -->|60 min delay| RE[Reservation Expiry]
    end

    PHT -->|If still waiting_payment| FAIL[FailReservation<br/>Release spot]
    RE -->|If still confirmed<br/>not checked in| EXPIRE[ExpireReservation<br/>Release spot]

    subgraph Fallback Polling Workers
        EW[Expiry Worker<br/>every 30s] -->|Scan DB| EXPIRE
        PW[Payment Timeout Worker<br/>every 30s] -->|Scan DB| FAIL
    end
```

### Event-Driven Messaging (NATS JetStream)

```mermaid
graph LR
    subgraph RESERVATION_SEARCH Stream
        RS_PUB[Reservation Service] -->|reservation.search.spot-updated| RS_STREAM[RESERVATION_SEARCH<br/>InterestPolicy]
        RS_STREAM -->|spot_updated_search| RS_CON[Search Service]
    end

    subgraph RESERVATION_ANALYTICS Stream
        RA_PUB[Reservation Service] -->|reservation.analytics.*| RA_STREAM[RESERVATION_ANALYTICS<br/>LimitsPolicy, 7d retention]
        RA_STREAM -->|lifecycle_analytics| RA_CON[Analytics]
    end

    subgraph PAYMENT_RESERVATION Stream
        PR_PUB[Payment Service] -->|payment.reservation.success<br/>payment.reservation.failed| PR_STREAM[PAYMENT_RESERVATION<br/>InterestPolicy]
        PR_STREAM -->|payment_result_reservation| PR_CON[Reservation Service]
    end
```

**Retention policies:**
- **InterestPolicy** (RESERVATION_SEARCH, PAYMENT_RESERVATION): Messages are kept only while there are active consumers. Once all consumers acknowledge, messages are discarded. Ideal for real-time event delivery where historical replay isn't needed.
- **LimitsPolicy** (RESERVATION_ANALYTICS): Messages are retained up to configured limits (7 days). Allows late-joining consumers or replay for analytics reprocessing.

**Key design choices:**
- Consumer naming: `{subject_short}_{consuming_service}` (e.g., `spot_updated_search`)
- Deduplication via MsgID: `{event}-{id}-{timestamp_nano}`
- `NATS_ENABLED` opt-in — services degrade gracefully without NATS (fall back to synchronous paths)
- gRPC remains for request-response (queries, fee calculation, payment intent); NATS handles "something happened, react when you can"

---

## Entity Relationship Diagram

```mermaid
erDiagram
    PARKING_SPOTS {
        uuid id PK
        string spot_number
        int floor
        string zone
        enum type "car | motorcycle"
        enum status "available | reserved | occupied"
        timestamp created_at
        timestamp updated_at
    }

    DRIVERS {
        uuid id PK
        string name
        string phone
        string email
        timestamp created_at
        timestamp updated_at
    }

    RESERVATIONS {
        uuid id PK
        uuid driver_id FK
        uuid spot_id FK
        enum status "waiting_payment | confirmed | checked_in | checked_out | completed | cancelled | expired | failed"
        enum vehicle_type "car | motorcycle"
        string license_plate
        timestamp check_in_time
        timestamp check_out_time
        bigint booking_fee
        bigint hourly_rate
        bigint total_amount
        bigint overnight_fee
        bigint penalty_amount
        timestamp created_at
        timestamp updated_at
    }

    DRIVERS ||--o{ RESERVATIONS : "makes"
    PARKING_SPOTS ||--o{ RESERVATIONS : "assigned to"
```

---

## State Machines

### Reservation State Machine

```mermaid
stateDiagram-v2
    [*] --> waiting_payment: CreateReservation
    waiting_payment --> confirmed: ConfirmReservation<br/>(booking fee paid)
    waiting_payment --> failed: Payment timeout (10 min)
    waiting_payment --> cancelled: CancelReservation

    confirmed --> checked_in: CheckIn
    confirmed --> expired: Expiry timeout (60 min)
    confirmed --> cancelled: CancelReservation

    checked_in --> checked_out: CheckOut
    checked_out --> completed: CompleteCheckout<br/>(session fee paid)

    failed --> [*]
    cancelled --> [*]
    expired --> [*]
    completed --> [*]
```

### Parking Spot State Machine

```mermaid
stateDiagram-v2
    [*] --> available
    available --> reserved: CreateReservation
    reserved --> occupied: CheckIn
    reserved --> available: Cancel / Expire / Fail
    occupied --> available: CompleteCheckout
```

---

## Assumptions

The following assumptions scope the MVP implementation:

- **Single parking area** — Centralized inventory with no multi-area or multi-tenant support
- **Booking fee** — 5,000 IDR charged on reservation confirmation; non-refundable
- **No cancellation fee** — Driver simply forfeits the booking fee already charged
- **Overnight fee** — 20,000 IDR per midnight crossed (not a flat one-time fee), justified by fairness for multi-night stays
- **No overstay penalty** — Additional time beyond checkout is billed at the standard hourly rate (5,000 IDR/hour)
- **Payment gateway is stubbed** — Interface-ready for Midtrans/Xendit QRIS integration
- **Presence service** — Uses GPS + Redis Geo for wrong-spot detection (50m threshold)
- **Authentication is BYO-JWT** — Tokens issued externally by super-app or standalone auth service
- **Wrong-spot detection** — Warning/flag only, not a blocker (driver can still park)
- **Notification service** — Out of scope for MVP; NATS events provide the foundation for future implementation
- **Arrival detection** — Location-based arrival detection replaced by manual check-in for MVP simplicity

---

## Tech Stack

| Category | Technology |
|----------|-----------|
| Language | Go 1.25 |
| RPC | gRPC / Protocol Buffers v3 |
| HTTP | Gin (gateway REST layer) |
| Database | PostgreSQL 14 |
| Cache & Locks | Redis 7.0 |
| Task Queue | Asynq (Redis-backed) |
| Event Streaming | NATS JetStream 2.10 |
| Observability | OpenTelemetry → Grafana (Tempo + Prometheus + Loki) |
| Containerization | Docker & Docker Compose |
| Reverse Proxy | Traefik |
| CI/CD | GitHub Actions → GHCR → Watchtower |
| Code Quality | SonarCloud |

---

## 3rd-Party Libraries & Justification

| Library | Justification |
|---------|---------------|
| `google.golang.org/grpc` | gRPC framework for service-to-service communication (assessment requirement) |
| `google.golang.org/protobuf` | Protocol Buffers serialization for gRPC |
| `github.com/jmoiron/sqlx` | Lightweight SQL extension over database/sql, struct scanning without full ORM overhead |
| `github.com/jackc/pgx/v5` | High-performance PostgreSQL driver with native Go types |
| `github.com/redis/go-redis/v9` | Redis client for distributed locking, caching, and geo operations |
| `github.com/bsm/redislock` | Production-grade Redis distributed lock (Redlock algorithm) |
| `github.com/sony/gobreaker` | Circuit breaker pattern for resilient inter-service calls |
| `github.com/hibiken/asynq` | Redis-based async task queue for background workers (expiry, billing) |
| `github.com/nats-io/nats.go` | NATS JetStream client for event-driven messaging between services |
| `github.com/joho/godotenv` | Environment variable loading from .env files |
| `github.com/google/uuid` | RFC 4122 UUID generation for entity IDs and idempotency keys |
| `go.opentelemetry.io/otel` | OpenTelemetry SDK for distributed tracing and metrics |
| `github.com/stretchr/testify` | Assertion library for readable, maintainable tests |
| `github.com/testcontainers/testcontainers-go` | Disposable Docker containers for integration/E2E tests |
| `pgregory.net/rapid` | Property-based testing for invariant verification |
| `golang.org/x/time/rate` | Token bucket rate limiter for API protection |

---

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- PostgreSQL 14+
- Redis 7+

### Local Development

```bash
# Clone
git clone https://github.com/piresc/parkir-pintar.git
cd parkir-pintar

# Copy environment config
cp config/.env.example config/.env
# Edit config/.env with your database credentials and JWT secret

# Run migrations
migrate -path db/migrations -database "postgres://user:pass@localhost:5432/parkir_pintar?sslmode=disable&search_path=reservation" up

# Start all services via Docker Compose
docker compose -f deploy/docker-compose.local.yml up -d
```

### Docker Compose (Local Ports)

| Service | Port |
|---------|------|
| Gateway | 11000 |
| Search | 11001 |
| Reservation | 11002 |
| Billing | 11003 |
| Payment | 11004 |

---

## Configuration

All configuration is via environment variables (12-factor). See `pkg/config/config.go` for the full struct.

### Key Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `local` | Environment (local/staging/production) |
| `SERVER_PORT` | `8080` | HTTP server port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_DATABASE` | — | Database name |
| `DB_SCHEMA` | `public` | PostgreSQL schema |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `JWT_SECRET` | — | **Required.** JWT signing secret |
| `PAYMENT_TIMEOUT_MINUTES` | `10` | Time to complete payment before reservation fails |
| `RESERVATION_EXPIRY_MINUTES` | `60` | Time to check in after confirmation before expiry |
| `ASYNQ_CONCURRENCY` | `10` | Asynq worker concurrency |
| `GRPC_SERVER_PORT` | `9090` | gRPC listen port |
| `GRPC_BILLING_TARGET` | `localhost:9093` | Billing service gRPC address |
| `GRPC_PAYMENT_TARGET` | `localhost:9094` | Payment service gRPC address |
| `TRACING_ENABLED` | `false` | Enable OpenTelemetry tracing |
| `TRACING_EXPORTER` | `noop` | Exporter type (noop/otlp) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTLP collector endpoint |
| `NATS_URL` | `nats://localhost:4222` | NATS server connection URL |
| `NATS_ENABLED` | `false` | Enable NATS JetStream event streaming |

---

## API Reference

### REST Endpoints (via Gateway)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/spots` | List available spots (filter by type, zone, floor) |
| GET | `/api/spots/:id` | Get spot details |
| POST | `/api/reservations` | Create reservation |
| POST | `/api/reservations/:id/confirm` | Confirm + pay booking fee |
| POST | `/api/reservations/:id/checkin` | Check in to spot |
| POST | `/api/reservations/:id/checkout` | Check out of spot |
| POST | `/api/reservations/:id/complete` | Complete + pay session fee |
| POST | `/api/reservations/:id/cancel` | Cancel reservation |
| GET | `/api/reservations/:id` | Get reservation details |
| GET | `/api/reservations` | List driver's reservations |
| GET | `/api/v1/analytics/peak-hours` | Peak hour statistics |
| GET | `/api/v1/analytics/occupancy` | Daily occupancy & usage patterns |

### Authentication

All `/api/*` endpoints require a valid JWT in the `Authorization: Bearer <token>` header. The system uses BYO-JWT — tokens are issued externally and validated against the configured `JWT_SECRET`.

### gRPC Services

Proto definitions in `proto/*/v1/*.proto`:

- `ReservationService` — Full reservation lifecycle
- `BillingService` — Fee calculation
- `PaymentService` — QRIS payment processing
- `SearchService` — Spot search and filtering

---

## Testing

```bash
# Run all tests
go test ./...

# Unit tests only (fast)
go test ./internal/... ./pkg/...

# E2E tests (requires running services)
go test ./tests/e2e/...

# Integration tests
go test ./tests/integration/...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Categories

| Category | Path | Description |
|----------|------|-------------|
| Unit | `internal/*/usecase/*_test.go` | Business logic, mocked dependencies |
| Unit | `internal/*/handler/*_test.go` | gRPC handler request/response mapping |
| Unit | `pkg/*_test.go` | Shared library tests |
| Property | `internal/reservation/usecase/spot_inventory_property_test.go` | Spot inventory invariants |
| Integration | `tests/integration/` | Database + Redis integration |
| E2E | `tests/e2e/` | Full flow including extended stay + overnight |

---

## CI/CD Pipeline

```mermaid
graph LR
    PUSH[Push to main] --> GHA[GitHub Actions]
    GHA --> LINT[golangci-lint]
    GHA --> TEST[go test ./...]
    GHA --> SONAR[SonarCloud Analysis]
    GHA --> BUILD[Docker Build]
    BUILD --> GHCR[Push to GHCR]
    GHCR --> WT[Watchtower<br/>Auto-pull]
    WT --> DEPLOY[Staging Server]
```

**Pipeline stages:**
1. **Lint** — `golangci-lint` for code quality
2. **Test** — Full test suite with coverage report
3. **SonarCloud** — Static analysis, code smells, security hotspots
4. **Build** — Multi-stage Docker build per service
5. **Push** — Container images to GitHub Container Registry
6. **Deploy** — Watchtower detects new images and restarts containers

---

## Monitoring & Observability

| Component | Port | Purpose |
|-----------|------|---------|
| Grafana | 3000 | Dashboards (metrics, traces, logs) |
| Prometheus | 9090 | Metrics collection & alerting |
| Tempo | 3200 | Distributed trace storage |
| Loki | 3100 | Log aggregation |
| Grafana Alloy | 4319 (OTLP) | Telemetry collector |

### Instrumentation

- **Traces**: All gRPC calls instrumented with OpenTelemetry spans
- **Metrics**: Request count, latency histograms, error rates per service
- **Logs**: Structured JSON logging with trace correlation (trace_id in log entries)
- **Alerts**: Prometheus alerting rules for high error rates and latency

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **gRPC for inter-service** | Type-safe contracts, HTTP/2 multiplexing, code generation |
| **REST gateway** | Client compatibility, simpler mobile/web integration |
| **BYO-JWT (no user DB)** | Assessment scope — auth is external, system validates tokens |
| **QRIS-only payment (stub)** | Indonesian market standard; stub allows testing without real provider |
| **Redis distributed locks** | Prevents double-booking race conditions at scale |
| **Asynq + polling fallback** | Asynq for precise delayed tasks; polling catches edge cases |
| **NATS JetStream for event-driven sync** | Decouples services; search reads spot-updated events, analytics consumes reservation lifecycle events, payment results flow async |
| **One stream per producer→consumer pair** | Avoids shared consumer complexity; clear ownership and independent scaling |
| **Two-moment payment** | Minimizes driver risk, ensures booking commitment |
| **Per-midnight overnight fee** | Fair billing — only charges for actual midnight crossings |
| **User-selected spot assignment** | Driver picks their preferred spot (vs. system auto-assign) |
| **Circuit breaker on gRPC** | Prevents cascade failures between services |
| **Extracted pricing engine** | Testable, reusable; 11 unit tests cover edge cases |

---

## License

MIT
# Deployment: Coolify (webhook via CI)
