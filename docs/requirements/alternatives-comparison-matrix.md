# Alternatives Comparison Matrix

**Project:** ParkirPintar — Smart Parking Backend System  
**Version:** 1.0  
**Date:** 2026-05-13  
**Status:** Approved

---

## Scoring Methodology

Each alternative is scored on weighted criteria using a 1-5 scale:

| Score | Meaning |
|-------|---------|
| 1 | Poor — significant drawbacks, high risk |
| 2 | Below average — notable limitations |
| 3 | Adequate — meets basic needs |
| 4 | Good — strong fit with minor trade-offs |
| 5 | Excellent — optimal fit for requirements |

**Final Score** = Σ (criterion_weight × score) / Σ weights

---

## 1. Architecture Style: Monolith vs Microservices vs Serverless

### Context

ParkirPintar manages a parking facility with 7 distinct domains (search, reservation, billing, payment, presence, notification, analytics). The system requires real-time availability, distributed locking, and event-driven communication.

### Criteria & Scoring

| Criterion | Weight | Monolith | Microservices | Serverless |
|-----------|--------|----------|---------------|------------|
| Independent deployability | 5 | 1 | 5 | 5 |
| Team scalability | 4 | 2 | 5 | 4 |
| Operational complexity | 4 | 5 | 2 | 3 |
| Performance (latency) | 5 | 5 | 4 | 2 |
| Fault isolation | 5 | 1 | 5 | 4 |
| Technology flexibility | 3 | 2 | 5 | 3 |
| Development speed (initial) | 3 | 5 | 3 | 4 |
| Cost at low scale | 3 | 5 | 3 | 5 |
| Cost at high scale | 4 | 3 | 4 | 2 |
| Real-time streaming support | 5 | 4 | 5 | 1 |
| Distributed locking support | 5 | 3 | 5 | 2 |
| Observability | 4 | 4 | 4 | 3 |

### Weighted Scores

| Alternative | Weighted Score | Normalized (1-5) |
|-------------|---------------|-------------------|
| Monolith | 155/250 | 3.10 |
| **Microservices** | **210/250** | **4.20** |
| Serverless | 155/250 | 3.10 |

### Recommendation: **Microservices** ✅

**Justification:**
- ParkirPintar has clear domain boundaries (7 services with distinct responsibilities)
- Real-time presence streaming requires long-lived connections (incompatible with serverless cold starts)
- Distributed locking for reservation requires persistent Redis connections
- Fault isolation critical: payment gateway failures must not affect search availability
- Event-driven architecture (NATS) naturally fits microservices pub/sub pattern
- Team can deploy billing fixes without risking reservation stability

**Trade-offs accepted:**
- Higher operational complexity (mitigated by Docker Compose locally, Kubernetes in production)
- Network latency between services (mitigated by gRPC + HTTP/2 multiplexing)
- Distributed debugging (mitigated by OpenTelemetry distributed tracing)

**ADR Reference:** [ADR-0001: Microservices Architecture](../adr/0001-microservices-architecture.md)

---

## 2. Communication Protocol: REST vs gRPC vs GraphQL

### Context

Services need to communicate synchronously for request-response flows (reservation creation, fee calculation) and the gateway needs to expose APIs to mobile clients.

### Criteria & Scoring

| Criterion | Weight | REST (JSON) | gRPC (Protobuf) | GraphQL |
|-----------|--------|-------------|-----------------|---------|
| Performance (serialization) | 5 | 2 | 5 | 3 |
| Performance (transport) | 5 | 3 | 5 | 3 |
| Streaming support | 5 | 2 | 5 | 2 |
| Type safety | 5 | 2 | 5 | 4 |
| Code generation | 4 | 2 | 5 | 4 |
| Mobile client compatibility | 4 | 5 | 2 | 4 |
| Browser compatibility | 3 | 5 | 2 | 5 |
| Learning curve | 3 | 5 | 3 | 3 |
| Tooling ecosystem | 3 | 5 | 4 | 4 |
| Contract enforcement | 5 | 2 | 5 | 4 |
| Backward compatibility | 4 | 3 | 5 | 4 |
| Observability integration | 4 | 4 | 5 | 3 |

### Weighted Scores

| Alternative | Weighted Score | Normalized (1-5) |
|-------------|---------------|-------------------|
| REST (JSON) | 160/250 | 3.20 |
| **gRPC (Protobuf)** | **225/250** | **4.50** |
| GraphQL | 175/250 | 3.50 |

### Recommendation: **gRPC for internal, REST for external** ✅

**Justification:**
- Internal (service-to-service): gRPC provides 5-10x better serialization performance, native streaming for Presence service, strong contract enforcement via `.proto` files
- External (client-facing): REST via Gateway for mobile compatibility (Gin framework handles REST→gRPC transcoding)
- Presence service requires bidirectional streaming (`StreamLocation`) — only gRPC supports this natively
- Proto files serve as living documentation and enable automated code generation

**Trade-offs accepted:**
- Dual protocol complexity (REST at edge, gRPC internal) — mitigated by Gateway transcoding layer
- gRPC debugging harder than REST — mitigated by gRPC reflection and OTel tracing

**ADR Reference:** [ADR-0002: gRPC Internal Communication](../adr/0002-grpc-internal-communication.md)

---

## 3. Database Strategy: Single Shared vs Schema-per-Service vs DB-per-Service

### Context

7 microservices need data persistence. Data includes reservations, billing records, payments, presence logs, and parking spot inventory.

### Criteria & Scoring

| Criterion | Weight | Single Shared DB | Schema-per-Service | DB-per-Service |
|-----------|--------|-----------------|-------------------|----------------|
| Data isolation | 5 | 1 | 4 | 5 |
| Operational simplicity | 5 | 5 | 4 | 2 |
| Cross-service queries | 3 | 5 | 3 | 1 |
| Independent scaling | 4 | 1 | 2 | 5 |
| Cost (infrastructure) | 4 | 5 | 5 | 2 |
| Migration complexity | 4 | 5 | 4 | 2 |
| Backup/restore granularity | 3 | 3 | 3 | 5 |
| Connection management | 4 | 3 | 3 | 4 |
| Transaction support (cross-service) | 3 | 5 | 3 | 1 |
| Team autonomy | 4 | 2 | 4 | 5 |
| Deployment independence | 4 | 2 | 3 | 5 |
| Suitable for current scale | 5 | 5 | 5 | 2 |

### Weighted Scores

| Alternative | Weighted Score | Normalized (1-5) |
|-------------|---------------|-------------------|
| Single Shared DB | 165/250 | 3.30 |
| **Schema-per-Service** | **185/250** | **3.70** |
| DB-per-Service | 170/250 | 3.40 |

### Recommendation: **Schema-per-Service** ✅ (current implementation)

**Justification:**
- Balances isolation (each service owns its schema) with operational simplicity (single PostgreSQL instance)
- At current scale (400 spots, ~500 reservations/day), a single PostgreSQL instance is sufficient
- Schema boundaries enforce ownership without the cost of multiple database instances
- Migration path to DB-per-service exists when scale demands it (documented in `docs/superpowers/plans/2026-05-07-database-per-service-separation.md`)
- Cross-service data needs handled via gRPC calls, not direct DB access

**Trade-offs accepted:**
- Single point of failure (mitigated by Phase 2 read replicas)
- Shared connection pool pressure (mitigated by PgBouncer in Phase 2)
- Cannot independently scale storage per service (acceptable at current scale)

---

## 4. Messaging System: NATS vs Kafka vs RabbitMQ

### Context

Services need asynchronous event-driven communication for cache invalidation, notifications, and decoupling. Events include reservation state changes, billing events, payment results, and presence detections.

### Criteria & Scoring

| Criterion | Weight | NATS JetStream | Apache Kafka | RabbitMQ |
|-----------|--------|---------------|--------------|----------|
| Operational simplicity | 5 | 5 | 2 | 3 |
| Performance (latency) | 5 | 5 | 4 | 4 |
| Performance (throughput) | 4 | 4 | 5 | 3 |
| At-least-once delivery | 5 | 5 | 5 | 5 |
| Exactly-once semantics | 3 | 3 | 5 | 3 |
| Message replay | 4 | 4 | 5 | 2 |
| Resource footprint | 5 | 5 | 1 | 3 |
| Go client quality | 4 | 5 | 4 | 4 |
| Clustering (HA) | 4 | 4 | 5 | 4 |
| Learning curve | 4 | 5 | 2 | 4 |
| Consumer groups | 4 | 4 | 5 | 4 |
| Cloud-native (single binary) | 4 | 5 | 1 | 3 |

### Weighted Scores

| Alternative | Weighted Score | Normalized (1-5) |
|-------------|---------------|-------------------|
| **NATS JetStream** | **225/255** | **4.41** |
| Apache Kafka | 185/255 | 3.63 |
| RabbitMQ | 175/255 | 3.43 |

### Recommendation: **NATS JetStream** ✅

**Justification:**
- Single binary deployment (~20 MB) vs Kafka's ZooKeeper + broker + schema registry
- Sub-millisecond latency for cache invalidation (Search service needs fast updates)
- At-least-once delivery with durable consumers meets all ParkirPintar requirements
- Native Go client (`nats.go`) with excellent ergonomics
- JetStream provides persistence, replay, and consumer groups without Kafka's operational burden
- Resource footprint: NATS uses ~128 MB RAM vs Kafka's 1+ GB minimum
- Current event volume (~1000 events/day) doesn't justify Kafka's complexity

**Trade-offs accepted:**
- No exactly-once semantics (mitigated by idempotent consumers)
- Smaller ecosystem than Kafka (acceptable — no need for Kafka Connect, ksqlDB)
- Less mature stream processing (not needed — events are simple pub/sub)

**ADR Reference:** [ADR-0003: NATS Event-Driven Architecture](../adr/0003-nats-event-driven.md)

---

## 5. Deployment Platform: Kubernetes vs Docker Compose vs Serverless

### Context

ParkirPintar needs a deployment platform that supports 7 microservices + infrastructure (PostgreSQL, Redis, NATS) with observability stack (Grafana, Tempo, Prometheus, Loki).

### Criteria & Scoring

| Criterion | Weight | Kubernetes | Docker Compose | Serverless (ECS/Lambda) |
|-----------|--------|-----------|----------------|------------------------|
| Auto-scaling | 5 | 5 | 1 | 4 |
| Self-healing | 5 | 5 | 2 | 4 |
| Operational complexity | 4 | 2 | 5 | 3 |
| Local development | 5 | 3 | 5 | 2 |
| Production readiness | 5 | 5 | 2 | 4 |
| Cost (small scale) | 4 | 2 | 5 | 4 |
| Cost (large scale) | 4 | 4 | 2 | 3 |
| Service discovery | 4 | 5 | 4 | 4 |
| Secret management | 4 | 5 | 2 | 4 |
| Rolling deployments | 5 | 5 | 2 | 4 |
| Resource efficiency | 4 | 4 | 3 | 4 |
| Team expertise required | 3 | 2 | 5 | 3 |

### Weighted Scores

| Alternative | Weighted Score | Normalized (1-5) |
|-------------|---------------|-------------------|
| Kubernetes | 205/260 | 3.94 |
| Docker Compose | 175/260 | 3.37 |
| Serverless | 180/260 | 3.46 |

### Recommendation: **Docker Compose (dev/staging) + Kubernetes (production)** ✅

**Justification:**
- **Development/Staging:** Docker Compose provides zero-friction local development (single `docker compose up` starts all 7 services + infra)
- **Production:** Kubernetes provides auto-scaling, self-healing, rolling deployments, and secret management needed for 99.9% availability target
- Phased approach: start with Docker Compose (current), migrate to Kubernetes when scaling demands it (Phase 2)
- Long-lived gRPC connections and streaming (Presence service) incompatible with serverless cold starts
- Observability stack (Grafana, Tempo, Loki, Prometheus) has native Kubernetes operators

**Trade-offs accepted:**
- Kubernetes operational complexity (mitigated by managed K8s: EKS/GKE)
- Dual deployment configuration (Compose + K8s manifests) — mitigated by Helm charts
- Higher cost at small scale for K8s (acceptable for production reliability)

---

## 6. Decision Log

| # | Decision | Date | ADR | Chosen Alternative | Score |
|---|----------|------|-----|-------------------|-------|
| 1 | Architecture style | 2025-01-01 | [ADR-0001](../adr/0001-microservices-architecture.md) | Microservices | 4.20/5 |
| 2 | Communication protocol | 2025-01-01 | [ADR-0002](../adr/0002-grpc-internal-communication.md) | gRPC (internal) + REST (external) | 4.50/5 |
| 3 | Messaging system | 2025-01-01 | [ADR-0003](../adr/0003-nats-event-driven.md) | NATS JetStream | 4.41/5 |
| 4 | Distributed locking | 2025-01-01 | [ADR-0004](../adr/0004-distributed-locking-redis.md) | Redis (SETNX + Lua) | N/A |
| 5 | Observability | 2025-01-01 | [ADR-0005](../adr/0005-opentelemetry-observability.md) | OpenTelemetry + Grafana stack | N/A |
| 6 | Infrastructure as Code | 2025-01-01 | [ADR-0006](../adr/0006-terraform-iac.md) | Terraform | N/A |
| 7 | Database strategy | 2026-05-07 | — | Schema-per-service | 3.70/5 |
| 8 | Deployment platform | 2026-05-07 | — | Docker Compose + Kubernetes | 3.94/5 |

---

## Summary

| Category | Chosen | Runner-Up | Key Differentiator |
|----------|--------|-----------|-------------------|
| Architecture | Microservices | Monolith/Serverless (tied) | Fault isolation + streaming support |
| Communication | gRPC | GraphQL | Native streaming + performance |
| Database | Schema-per-Service | DB-per-Service | Operational simplicity at current scale |
| Messaging | NATS JetStream | Kafka | Resource footprint + operational simplicity |
| Deployment | Docker Compose + K8s | Serverless | Long-lived connections + auto-scaling |

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-05-13 | Engineering Team | Initial alternatives comparison matrix |
