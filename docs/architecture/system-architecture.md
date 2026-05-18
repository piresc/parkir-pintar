# ParkirPintar System Architecture

## Overview

ParkirPintar is a smart parking backend system built as a collection of Go microservices following clean architecture principles. The system manages a single centralized parking area (5 floors, 400 spots) and handles the full reservation lifecycle from spot search through billing and payment.

The architecture prioritizes:
- **Consistency** — distributed locking and database constraints prevent double-booking
- **Observability** — full OpenTelemetry instrumentation (traces, metrics, logs)
- **Resilience** — circuit breakers, retries, graceful degradation
- **Scalability** — stateless services, event-driven communication

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Client Layer                                  │
│                                                                       │
│   Super App / Mobile Client ──── REST/JSON over HTTPS ────┐          │
└───────────────────────────────────────────────────────────┼──────────┘
                                                            │
┌───────────────────────────────────────────────────────────▼──────────┐
│                      Edge / Ingress Layer                              │
│                                                                       │
│   Traefik (TLS termination, routing, load balancing)                  │
└───────────────────────────────────────────────────────────┬──────────┘
                                                            │
┌───────────────────────────────────────────────────────────▼──────────┐
│                      API Gateway (:8080)                               │
│                                                                       │
│   REST → gRPC transcoding │ JWT Auth │ Rate Limiting │ CORS          │
│   Recovery │ Tracing │ Request Logging                                │
└──────┬────────────┬────────────┬────────────┬────────────────────────┘
       │            │            │            │
       │ gRPC       │ gRPC       │ gRPC       │ gRPC
       ▼            ▼            ▼            ▼
┌──────────┐ ┌────────────┐ ┌─────────┐ ┌──────────┐
│  Search  │ │Reservation │ │ Payment │ │ Presence │
│  :9092   │ │   :9091    │ │  :9094  │ │  :9095   │
└────┬─────┘ └──┬───┬─────┘ └────┬────┘ └────┬─────┘
     │          │   │             │            │
     │          │   │ gRPC        │            │
     │          │   ▼             │            │
     │          │ ┌─────────┐    │            │
     │          │ │ Billing │    │            │
     │          │ │  :9093  │    │            │
     │          │ └────┬────┘    │            │
     │          │      │         │            │
     ▼          ▼      ▼         ▼            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Data Layer                                     │
│                                                                       │
│   ┌────────────┐    ┌─────────────┐    ┌──────────────────┐         │
│   │ PostgreSQL │    │    Redis    │    │  NATS JetStream  │         │
│   │   :5432    │    │    :6379    │    │      :4222       │         │
│   └────────────┘    └─────────────┘    └──────────────────┘         │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Service Responsibilities

| Service | Port | Responsibility | Data Owned |
|---------|------|----------------|------------|
| **Gateway** | 8080 | REST entry point, JWT auth, rate limiting, gRPC transcoding | None (stateless) |
| **Search** | 9092 | Real-time availability queries, floor maps, spot details | Read: parking_spots |
| **Reservation** | 9091 | Spot reservation lifecycle, distributed locking, expiry worker | reservations, parking_spots (write) |
| **Billing** | 9093 | Fee calculation (pricing engine), invoice generation, penalties | billing_records, penalties |
| **Payment** | 9094 | Payment processing, QRIS integration, refunds | payments |
| **Presence** | 9095 | Sensor-based spot occupancy verification, wrong-spot detection | presence_logs |
| **Notification** | 9096 | Push/SMS/Email notifications (stub — logs payloads) | None |

### Service Boundaries

Each service follows clean architecture internally:

```
gRPC Handler → Usecase (business logic) → Repository (data access) → PostgreSQL
                    ↕
              Domain Model (entities, errors, interfaces)
```

- Services own their data and expose it only through gRPC interfaces
- No direct database access across service boundaries
- Shared schema (single PostgreSQL instance) with logical ownership boundaries

---

## Communication Patterns

### Synchronous: gRPC over HTTP/2

Used for request-response flows where the caller needs an immediate result.

```
Client → Gateway (REST)
Gateway → Search (gRPC)         — availability queries
Gateway → Reservation (gRPC)    — create, cancel, check-in, check-out
Gateway → Payment (gRPC)        — payment status
Gateway → Presence (gRPC)       — location streaming

Reservation → Billing (gRPC)    — fee calculation, invoice generation
Reservation → Payment (gRPC)    — payment processing
Billing → Payment (gRPC)        — payment for penalties
```

**Configuration:**
- Keepalive: 30s interval, 10s timeout
- Max message size: 4MB
- Connection pooling via gRPC's built-in HTTP/2 multiplexing
- Client-side interceptors: tracing, logging, retry

### Asynchronous: NATS JetStream

Used for event-driven communication where the publisher doesn't need an immediate response.

```
Reservation → NATS:
  - reservation.confirmed
  - reservation.checked_in
  - reservation.checked_out
  - reservation.expired
  - reservation.cancelled

Billing → NATS:
  - billing.calculated
  - billing.invoiced

Payment → NATS:
  - payment.success
  - payment.failed

Presence → NATS:
  - presence.arrival
  - presence.wrong_spot
```

**Subscribers:**
- Search Service: listens for reservation events to invalidate availability cache
- Notification Service: listens for all events to trigger notifications

**JetStream Configuration:**
- Retention: Limits-based
- Storage: File
- Max Age: 72 hours
- Delivery: At-least-once with consumer acknowledgment

---

## Data Flow Diagrams

### Reservation Creation Flow

```
Client                Gateway              Reservation           Redis              PostgreSQL           Billing             NATS
  │                     │                      │                   │                    │                  │                  │
  │── POST /reservations ──▶│                  │                   │                    │                  │                  │
  │                     │── CreateReservation ──▶│                  │                    │                  │                  │
  │                     │                      │── SETNX lock ────▶│                    │                  │                  │
  │                     │                      │◀── OK ────────────│                    │                  │                  │
  │                     │                      │── SELECT FOR UPDATE ──────────────────▶│                  │                  │
  │                     │                      │◀── spot available ─────────────────────│                  │                  │
  │                     │                      │── BEGIN + INSERT + UPDATE ────────────▶│                  │                  │
  │                     │                      │◀── COMMIT ────────────────────────────│                  │                  │
  │                     │                      │── StartBilling ───────────────────────────────────────────▶│                  │
  │                     │                      │── Publish event ──────────────────────────────────────────────────────────────▶│
  │                     │                      │── DEL lock ──────▶│                    │                  │                  │
  │                     │◀── Response ─────────│                   │                    │                  │                  │
  │◀── 201 Created ────│                      │                   │                    │                  │                  │
```

### Check-Out & Payment Flow

```
Client                Gateway              Reservation           PostgreSQL           Billing             Payment            NATS
  │                     │                      │                    │                  │                   │                  │
  │── POST /checkout ───▶│                     │                    │                  │                   │                  │
  │                     │── CheckOut ──────────▶│                   │                  │                   │                  │
  │                     │                      │── SELECT FOR UPDATE ▶│                │                   │                  │
  │                     │                      │── UPDATE status ────▶│                │                   │                  │
  │                     │                      │── CalculateFee ──────────────────────▶│                   │                  │
  │                     │                      │◀── BillingRecord ────────────────────│                   │                  │
  │                     │                      │── GenerateInvoice ───────────────────▶│                   │                  │
  │                     │                      │◀── Invoice ─────────────────────────│                   │                  │
  │                     │                      │── ProcessPayment ────────────────────────────────────────▶│                  │
  │                     │                      │◀── PaymentResponse ──────────────────────────────────────│                  │
  │                     │                      │── Publish event ──────────────────────────────────────────────────────────────▶│
  │                     │◀── Response ─────────│                    │                  │                   │                  │
  │◀── 200 OK ─────────│                      │                    │                  │                   │                  │
```

### Event-Driven Cache Invalidation

```
Reservation Service          NATS JetStream           Search Service            Redis
       │                           │                        │                     │
       │── reservation.confirmed ──▶│                       │                     │
       │                           │── deliver ────────────▶│                     │
       │                           │                        │── DEL cache key ───▶│
       │                           │                        │── ACK ────────────▶ │
       │                           │◀── ACK ───────────────│                     │
```

---

## Technology Stack

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| Language | Go | 1.25+ | Service implementation |
| HTTP Framework | Gin | 1.12.0 | Gateway REST API |
| RPC | gRPC | 1.80.0 | Inter-service communication |
| Serialization | Protocol Buffers | 3 | Message format |
| Database | PostgreSQL | 14 | Primary data store |
| Cache/Lock | Redis | 7.0 | Caching, distributed locks, rate limiting |
| Messaging | NATS JetStream | Latest | Async event bus |
| Observability | OpenTelemetry | 1.43.0 | Traces, metrics, logs |
| Tracing Backend | Tempo | — | Distributed tracing |
| Metrics Backend | Prometheus | — | Metrics storage |
| Logs Backend | Loki | — | Log aggregation |
| Dashboard | Grafana | — | Unified observability UI |
| Collector | Grafana Alloy | — | OTel signal routing |
| Containerization | Docker | — | Service packaging |
| Orchestration | Docker Compose | — | Local development |
| Reverse Proxy | Traefik | — | Edge routing, TLS |
| CI/CD | GitHub Actions / GitLab CI | — | Automated pipeline |

---

## Infrastructure Topology

### Local Development (Docker Compose)

```
┌─────────────────────────────────────────────────────────┐
│                  Docker Compose Network                   │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌────────────────┐        │
│  │ postgres │  │  redis   │  │ nats (jetstream)│        │
│  │  :5432   │  │  :6379   │  │  :4222 / :8222 │        │
│  └──────────┘  └──────────┘  └────────────────┘        │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ gateway  │  │  search  │  │reservation│              │
│  │  :8080   │  │  :9092   │  │   :9091   │              │
│  └──────────┘  └──────────┘  └──────────┘              │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ billing  │  │ payment  │  │ presence │              │
│  │  :9093   │  │  :9094   │  │  :9095   │              │
│  └──────────┘  └──────────┘  └──────────┘              │
│                                                          │
│  ┌──────────────┐                                        │
│  │ notification │                                        │
│  │    :9096     │                                        │
│  └──────────────┘                                        │
└─────────────────────────────────────────────────────────┘
```

All services share a single Docker network with DNS-based service discovery.

### Staging Environment

```
┌─────────────────────────────────────────────────────────────────┐
│                     Staging (Single Node)                         │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Traefik (TLS via Let's Encrypt, routing)                 │    │
│  │ parkir-pintar.staging.piresc.dev                         │    │
│  └──────────────────────────┬──────────────────────────────┘    │
│                              │                                    │
│  ┌───────────────────────────▼──────────────────────────────┐   │
│  │ Docker Compose (all services)                             │   │
│  │ + Grafana Alloy → Tempo/Prometheus/Loki                   │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐      │
│  │ PostgreSQL   │  │    Redis     │  │  NATS JetStream  │      │
│  │ (container)  │  │ (container)  │  │   (container)    │      │
│  └──────────────┘  └──────────────┘  └──────────────────┘      │
└─────────────────────────────────────────────────────────────────┘
```

### Production Environment (Reference)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Production (Kubernetes)                            │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Ingress Controller / ALB (TLS termination)                   │    │
│  │ parkir-pintar.piresc.dev                                     │    │
│  └──────────────────────────┬──────────────────────────────────┘    │
│                              │                                        │
│  ┌───────────────────────────▼──────────────────────────────────┐   │
│  │ Gateway Pods (HPA: 2-10 replicas)                             │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                              │ gRPC (client-side LB / service mesh)   │
│  ┌───────────────────────────▼──────────────────────────────────┐   │
│  │ Service Pods (per-service HPA)                                │   │
│  │ search(2-5) │ reservation(2-5) │ billing(2-3) │ payment(2-5) │   │
│  │ presence(2-3) │ notification(1-2)                             │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  ┌──────────────────┐  ┌────────────────┐  ┌────────────────────┐  │
│  │ RDS/Aurora (PG)  │  │ ElastiCache    │  │ NATS Cluster       │  │
│  │ Multi-AZ, r/w    │  │ Redis Cluster  │  │ 3-node, R=3        │  │
│  │ split            │  │ (failover)     │  │ (JetStream)        │  │
│  └──────────────────┘  └────────────────┘  └────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Network Flow (Request Path)

```
Internet
    │
    ▼
┌──────────┐
│ Traefik  │  TLS termination, path-based routing
└────┬─────┘
     │ HTTP (internal)
     ▼
┌──────────┐
│ Gateway  │  JWT validation, rate limiting, REST→gRPC transcoding
└────┬─────┘
     │ gRPC over HTTP/2
     ▼
┌──────────────────────────────────────────┐
│ gRPC Services                             │
│ (Search, Reservation, Billing,            │
│  Payment, Presence, Notification)         │
└────┬──────────────┬──────────────┬───────┘
     │              │              │
     ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────────┐
│PostgreSQL│  │  Redis   │  │NATS JetStream│
│          │  │          │  │              │
│• Primary │  │• Cache   │  │• Events      │
│  data    │  │• Locks   │  │• Pub/Sub     │
│• ACID    │  │• Rate    │  │• At-least-   │
│  txns    │  │  limits  │  │  once        │
└──────────┘  └──────────┘  └──────────────┘
```

---

## Resilience & Fault Tolerance

| Pattern | Implementation | Scope |
|---------|---------------|-------|
| Circuit Breaker | `pkg/circuitbreaker/` | External service calls |
| Retry with Exponential Backoff | `pkg/httpclient/` | Transient failures |
| Distributed Lock | `pkg/redislock/` (SETNX + Lua) | Spot reservation |
| Idempotency | gRPC middleware + DB unique constraint | All write operations |
| Graceful Degradation | Non-critical failures logged, not propagated | Billing, notifications |
| Singleflight | `golang.org/x/sync/singleflight` | Cache miss coalescing |
| Context-Aware Retry | Respects `ctx.Done()` during backoff | Payment processing |
| Health Checks | Liveness + readiness probes | All services |

---

## Observability Stack

```
┌─────────────────────────────────────────────────────────┐
│                   Go Services (OTel SDK)                  │
│                                                          │
│  TracerProvider ─┐                                       │
│  MeterProvider ──┼── OTLP gRPC ──▶ Grafana Alloy        │
│  LoggerProvider ─┘                      │                │
└─────────────────────────────────────────┼────────────────┘
                                          │
                    ┌─────────────────────┼─────────────────┐
                    │                     ▼                  │
                    │  ┌──────────────────────────────────┐ │
                    │  │         Grafana Alloy            │ │
                    │  │  (routing, batching, filtering)  │ │
                    │  └──────┬──────────┬──────────┬─────┘ │
                    │         │          │          │        │
                    │         ▼          ▼          ▼        │
                    │  ┌──────────┐ ┌────────┐ ┌──────┐    │
                    │  │  Tempo   │ │Promethe│ │ Loki │    │
                    │  │ (traces) │ │us(metr)│ │(logs)│    │
                    │  └──────────┘ └────────┘ └──────┘    │
                    │         │          │          │        │
                    │         └──────────┼──────────┘        │
                    │                    ▼                   │
                    │            ┌──────────────┐            │
                    │            │   Grafana    │            │
                    │            │ (dashboards) │            │
                    │            └──────────────┘            │
                    └───────────────────────────────────────┘
```

### Key Metrics Collected

- **HTTP**: request count, latency (p50/p95/p99), error rate
- **gRPC**: RPC count, latency, status codes
- **Database**: query count, latency, connection pool utilization
- **NATS**: publish/subscribe count, latency, consumer lag
- **Business**: reservations created, check-ins, check-outs, revenue

---

## Security

| Layer | Mechanism |
|-------|-----------|
| Transport | TLS 1.3 (Traefik terminates) |
| Authentication | JWT (HMAC-SHA256) validated at Gateway |
| Authorization | Driver ID extracted from token claims |
| Rate Limiting | Per-IP token bucket (100 req/min) |
| Input Validation | Proto schema + handler-level validation |
| SQL Injection | Parameterized queries via sqlx |
| CORS | Explicit origin allowlist (no wildcard) |
| Secrets | Environment variables, never in code |
| SSRF Protection | `pkg/httpclient/` blocks internal network ranges |
