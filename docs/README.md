# ParkirPintar Documentation

Comprehensive documentation for the ParkirPintar smart parking reservation system.

---

## Architecture

| Document | Description |
|----------|-------------|
| [System Overview](architecture/overview.md) | High-level architecture, service communication, middleware chains, resilience patterns |
| [ER Diagram](architecture/er-diagram.md) | Database schema relationships across 5 service schemas |
| [Sequence Diagrams](architecture/sequence-diagrams.md) | Key flow interactions (reservation, check-in, payment, check-out) |

## Services

| Document | Description |
|----------|-------------|
| [Gateway](services/gateway.md) | REST→gRPC transcoding, JWT auth, rate limiting, CORS |
| [Reservation](services/reservation.md) | Full reservation lifecycle, spot locking, state machine |
| [Billing](services/billing.md) | Fee calculation (hourly + overnight + booking fee), invoicing |
| [Payment](services/payment.md) | QRIS payment processing, refunds, circuit breaker |
| [Search](services/search.md) | Spot availability queries, CQRS read model, caching |
| [Analytics](services/analytics.md) | Peak hours, occupancy patterns, prediction |
| [Presence](services/presence.md) | Sensor-based occupancy verification |

## API Reference

| Document | Description |
|----------|-------------|
| [API Flows](api/api-flows/index.html) | Interactive documentation for all 16 REST endpoints |
| [Swagger UI](api/swagger-ui/index.html) | OpenAPI spec with interactive explorer |
| [OpenAPI Spec](api/swagger.yaml) | Raw OpenAPI 3.0 specification |

## Design

| Document | Description |
|----------|-------------|
| [Design Patterns](design/design-patterns.md) | 10 patterns used across services (Repository, Circuit Breaker, CQRS-lite, etc.) |
| [ADRs](adr/) | Architecture Decision Records (6 records) |

## Requirements

| Document | Description |
|----------|-------------|
| [Clarification Specs](requirements/clarification-specs.md) | Requirement analysis and architectural decisions |

## Operations

| Document | Description |
|----------|-------------|
| [Deployment](operations/deployment.md) | Coolify staging, AWS EKS production, CI/CD pipeline (4 workflows) |
| [Configuration](operations/configuration.md) | YAML config system, environment hierarchy, secret management |
| [Observability](operations/observability.md) | OpenTelemetry pipeline, Grafana stack, custom metrics |
| [SLO/SLI](operations/slo-sli.md) | Service level objectives, indicators, and alerting rules |
| [Profiling](operations/profiling.md) | Runtime profiling guide (CPU, heap, goroutine, trace) |

## Development

| Document | Description |
|----------|-------------|
| [Getting Started](development/getting-started.md) | Prerequisites, local setup, running services, Makefile targets |
| [Testing Strategy](development/testing.md) | Test pyramid, frameworks, patterns, CI integration |
| [Frontend](development/frontend.md) | React SPA architecture, components, build & deploy |

## Shared Packages

| Document | Description |
|----------|-------------|
| [Package Reference](development/shared-packages.md) | All 24 pkg/ packages — purpose, exports, usage examples |
