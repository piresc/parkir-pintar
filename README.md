# ParkirPintar — Smart Parking Reservation System

> **Solution Development Assessment 2026**

A production-grade smart parking backend with 7 Go microservices communicating via gRPC and NATS JetStream. REST gateway, automated billing, QRIS payment, and full observability.

**Live Demo:** https://parkir-pintar.piresc.dev

---

## Services

| Service | gRPC Port | Responsibility |
|---------|-----------|----------------|
| Gateway | 8082 (REST) | REST→gRPC transcoding, JWT auth, rate limiting |
| Search | 9092 | Spot availability queries, filtering |
| Reservation | 9091 | Full reservation lifecycle, spot locking |
| Billing | 9093 | Fee calculation (hourly + overnight) |
| Payment | 9094 | QRIS payment processing, refunds |
| Analytics | 9095 | Peak hours, occupancy patterns |
| Presence | 9096 | Sensor-based occupancy verification |

---

## Quick Start

```bash
git clone https://github.com/piresc/parkir-pintar.git
cd parkir-pintar
cp config/.env.example config/.env   # fill in secrets
docker compose up -d                  # starts infra + all services
```

---

## Project Structure

```
cmd/            7 service entrypoints + Dockerfiles
config/         local/ and staging/ YAML configs + .env (secrets only)
db/migrations/  Single init migration (full schema)
deploy/         Coolify compose (app, infra, observability) + deploy.sh
docs/           Architecture, design, ADRs, API docs
frontend/       React SPA
internal/       Domain logic per service
pkg/            Shared libraries (config, tracing, grpc, redis, nats, etc.)
proto/          Protocol Buffer definitions
tests/          E2E + load tests
```

---

## Documentation Index

| Document | Description |
|----------|-------------|
| [API Flows](docs/api-flows/index.html) | Interactive docs for all 16 REST endpoints |
| [Swagger](docs/swagger-ui/index.html) | OpenAPI spec + Swagger UI |
| [Architecture](docs/architecture.md) | System overview, HLD/LLD, state machines, ERD |
| [ER Diagram](docs/design/er-diagram.md) | Database schema relationships |
| [Sequence Diagrams](docs/design/sequence-diagrams.md) | Key flow interactions |
| [Design Patterns](docs/design/design-patterns.md) | Patterns used across services |
| [Clarification Specs](docs/clarification-specs.md) | Requirement analysis and decisions |
| [SLO/SLI](docs/slo-sli.md) | Service level objectives and indicators |
| [Testing Strategy](docs/testing/testing-strategy.md) | Test categories and approach |
| [ADRs](docs/adr/) | Architecture Decision Records |
| [Deploy README](deploy/coolify/README.md) | Coolify stacks, networks, CI/CD flow |

---

## Tech Stack

| Category | Technology |
|----------|------------|
| Language | Go 1.25 |
| RPC | gRPC / Protocol Buffers v3 |
| HTTP | Gin (gateway REST layer) |
| Database | PostgreSQL 14 (schema-per-service) |
| Cache & Locks | Redis 7.0 |
| Task Queue | Asynq (Redis-backed) |
| Event Streaming | NATS JetStream 2.10 |
| Observability | OpenTelemetry → Grafana (Tempo + Prometheus + Loki) |
| Containerization | Docker & Docker Compose |
| Deployment | Coolify (self-hosted) |
| Reverse Proxy | Traefik |
| CI/CD | GitHub Actions → GHCR → Coolify API |
| Code Quality | SonarCloud, golangci-lint, OWASP ZAP |

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| gRPC inter-service | Type-safe contracts, HTTP/2 multiplexing, codegen |
| REST gateway | Client compatibility, simpler mobile/web integration |
| NATS JetStream events | Decouples services; one stream per producer→consumer pair |
| Redis distributed locks | Prevents double-booking race conditions |
| Asynq + polling fallback | Precise delayed tasks + edge case coverage |
| Two-moment payment | Booking fee on confirm, session fee on checkout |
| Per-midnight overnight fee | Fair billing for multi-night stays (20k/midnight) |
| YAML config (no code defaults) | Single source of truth, missing config = hard error |
| Schema-per-service | Logical isolation without separate DB instances |

---

## Configuration

YAML is the source of truth: `config/<env>/<service>.yaml`

Env vars are secrets only: `DB_PASSWORD`, `JWT_SECRET`, `REDIS_PASSWORD`, `NATS_URL`, `OTEL_EXPORTER_OTLP_ENDPOINT`

---

## License

MIT
