# Gap Analysis — Kompetensi 2: Software Construction (Level 4)

**Project:** Parkir Pintar  
**Assessment Level:** 4 (Advanced/Lead)  
**Date:** 2026-05-13  
**Status:** 🟡 Partially Complete

---

## Overview

Level 4 Software Construction requires the ability to **define workflows, establish standards, create reusable libraries, implement distributed systems, and lead error handling practices** across projects. This document analyzes the parkir-pintar project against each sub-competency.

---

## Sub 1 — Collaboration Tools (Version Control & CI/CD)

### Level 4 Requirements

- Mampu mendefinisikan workflow version control (Trunk-based, Feature/Topic branch)
- Mampu merumuskan kebijakan CI/CD
- **Assessment:** Mendesain dan mengimplementasikan workflow version control + CI/CD

### ✅ What EXISTS

| Evidence | Location |
|----------|----------|
| Feature branch workflow | `.github/` workflows, branch naming conventions |
| GitHub Actions CI/CD | `.github/workflows/` |
| GitLab CI/CD (dual pipeline) | `.gitlab-ci.yml` |
| Automated testing in CI | CI pipeline stages |
| Linting in CI (golangci-lint) | CI configuration |
| Race detection in CI | `-race` flag in test stages |
| SonarCloud quality gate | CI integration |

### ❌ What's MISSING

1. **No CONTRIBUTING.md** — No explicit documentation of the branching strategy, commit conventions, or PR workflow
2. **No branch protection rules documentation** — Rules may exist but aren't documented as policy
3. **No CI/CD policy document** — Pipeline stages, deployment gates, and rollback procedures aren't formalized

### 🎯 ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Create CONTRIBUTING.md with branching strategy, commit conventions, PR process | `CONTRIBUTING.md` | High |
| 2 | Document CI/CD policy (stages, gates, rollback, environments) | `docs/ci-cd-policy.md` | High |
| 3 | Add branch protection rules documentation | `docs/branching-strategy.md` | Medium |
| 4 | Document deployment pipeline flow diagram | `docs/diagrams/ci-cd-flow.md` | Low |

### Assessment Criteria

- [ ] Documented version control workflow with rationale
- [ ] CI/CD policy document covering build, test, deploy stages
- [ ] Evidence of pipeline enforcement (quality gates, branch protection)
- [ ] Dual CI/CD implementation (GitHub Actions + GitLab CI) demonstrated

---

## Sub 2 — Writing Source Code (Design Patterns, Review, Standards)

### Level 4 Requirements

- Menguasai berbagai design pattern
- Dapat melakukan review dan feedback terhadap kode program yang kurang optimal
- Mampu mendefinisikan coding convention, pattern, dan aturan penulisan kode
- Mampu membuat library atau framework internal
- Mampu melakukan riset best-practices
- Mampu melakukan analisa performance dan integritas data
- Menguasai distributed programming
- **Assessment:** Design patterns, code review reports, coding guideline, library/framework, performance/integrity analysis, distributed app

### ✅ What EXISTS

| Evidence | Location |
|----------|----------|
| Repository Pattern | `internal/*/repository/` across services |
| Strategy Pattern | Rate limiting strategies, auth strategies |
| Circuit Breaker Pattern | `pkg/circuitbreaker/` |
| Adapter Pattern | `pkg/` adapters for external services |
| Middleware Pattern | `pkg/middleware/`, gRPC interceptors |
| Observer Pattern | NATS event subscribers |
| Factory Pattern | Service/repository factories |
| Token Bucket Pattern | Rate limiter implementation |
| Distributed Lock Pattern | `pkg/distlock/` or Redis-based locks |
| Idempotency Pattern | Idempotency key handling |
| 14+ reusable pkg/ libraries | `pkg/` directory (all tested) |
| Code review process | `REVIEW.md`, code review skill |
| golangci-lint in CI | `.github/workflows/`, `.gitlab-ci.yml` |
| SonarCloud quality gate | CI integration |
| Load tests (6 scenarios) | `tests/load/` or similar |
| Race condition tests (6 scenarios) | `tests/race/` or `-race` flag tests |
| Property-based tests | Test files with property-based approach |
| gRPC + NATS distributed system | Service-to-service communication |
| Redis distributed locks | Distributed coordination |

### ❌ What's MISSING

1. **No explicit coding style guide** — No `docs/coding-guidelines.md` defining conventions, naming rules, error handling patterns
2. **No `.golangci.yml` with custom rules** — Linter runs with defaults, no project-specific rule customization
3. **No Go benchmarks (`func BenchmarkXxx`)** — Performance analysis lacks micro-benchmarks
4. **No pprof profiling** — No CPU/memory profiling integration or reports
5. **No formal design patterns documentation** — Patterns exist but aren't cataloged with rationale

### 🎯 ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Create coding guidelines document | `docs/coding-guidelines.md` | High |
| 2 | Create custom `.golangci.yml` with project-specific rules | `.golangci.yml` | High |
| 3 | Add Go benchmarks for critical paths (parking entry/exit, payment) | `internal/*/benchmark_test.go` | High |
| 4 | Add pprof endpoints and profiling documentation | `docs/profiling.md`, handler registration | Medium |
| 5 | Document design patterns catalog with rationale | `docs/design-patterns.md` | Medium |
| 6 | Create performance analysis report from load tests | `docs/performance-analysis.md` | Medium |
| 7 | Add data integrity analysis document | `docs/data-integrity-analysis.md` | Medium |

### Assessment Criteria

- [ ] 10+ design patterns identified and documented with rationale
- [ ] Code review reports/evidence (REVIEW.md + PR history)
- [ ] Coding guideline document published
- [ ] Internal libraries created and tested (`pkg/` with tests)
- [ ] Performance analysis with benchmarks and profiling
- [ ] Data integrity analysis (race conditions, consistency)
- [ ] Distributed programming demonstrated (gRPC + NATS + locks)

---

## Sub 3 — Manage Data in Application (Distributed Data, Events, Messaging)

### Level 4 Requirements

- Memahami prinsip dan implementasi akses data pada database terdistribusi (eventual consistency, CAP theorem)
- Memahami prinsip dan implementasi aliran data dalam aplikasi berbasis event (event sourcing)
- Memahami prinsip dan implementasi aliran data dalam aplikasi berbasis messaging
- **Assessment:** Distributed app, event-based app, messaging in app

### ✅ What EXISTS

| Evidence | Location |
|----------|----------|
| Schema-per-service (distributed DB) | Each service has own PostgreSQL schema |
| CQRS read model | Read-optimized projections |
| Eventual consistency via NATS | Event-driven state propagation |
| NATS JetStream (4 streams, 12 subjects) | `configs/nats/` or stream setup code |
| Durable consumers | JetStream consumer configuration |
| Redis caching layer | Cache-aside pattern implementation |
| Distributed locks (Redis) | Cross-service coordination |
| Event-driven architecture | NATS pub/sub across services |

### ❌ What's MISSING

1. **No full event sourcing** — No event store with append-only log and event replay capability
2. **No outbox pattern** — No transactional outbox for guaranteed event publishing
3. **No saga orchestrator** — No explicit saga pattern for distributed transactions
4. **No CAP theorem analysis document** — Trade-offs not formally documented
5. **No event schema registry** — No versioned event schemas

### 🎯 ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Implement event sourcing for parking session lifecycle | `internal/parking/eventsource/` | High |
| 2 | Implement outbox pattern for reliable event publishing | `pkg/outbox/` | High |
| 3 | Implement saga orchestrator for payment flow | `internal/payment/saga/` | Medium |
| 4 | Document CAP theorem analysis for the system | `docs/cap-theorem-analysis.md` | Medium |
| 5 | Create event schema documentation with versioning | `docs/event-schemas.md` | Medium |
| 6 | Document data flow diagrams (event-based) | `docs/diagrams/data-flow.md` | Low |

### Assessment Criteria

- [ ] Distributed database with eventual consistency demonstrated
- [ ] Event sourcing implementation (event store + replay)
- [ ] Messaging-based data flow (NATS JetStream with durable consumers)
- [ ] CAP theorem trade-offs documented
- [ ] Outbox pattern or equivalent reliability mechanism

---

## Sub 4 — Integrate and Implement API (Review, Alternatives, Evolution)

### Level 4 Requirements

- Mampu melakukan review dan memberikan feedback terhadap desain API
- Dapat mengajukan berbagai alternatif desain API
- Dapat membuat roadmap evolusi API
- Memahami berbagai metode integrasi aplikasi (messaging, file transfer, dsb)
- **Assessment:** Review API design, present alternatives, integrate apps

### ✅ What EXISTS

| Evidence | Location |
|----------|----------|
| REST API (HTTP) | Gateway service, route handlers |
| gRPC API (inter-service) | `.proto` files, gRPC servers |
| NATS messaging integration | Event-driven communication |
| API versioning (v1) | Route prefixes `/api/v1/` |
| Rate limiting | Token bucket implementation |
| Health check endpoints | `/health`, `/ready` endpoints |
| OpenTelemetry tracing | Distributed tracing across services |
| Dual protocol (REST + gRPC) | Multiple integration methods |

### ❌ What's MISSING

1. **No API evolution roadmap** — No documented plan for v2, deprecation policy, or migration strategy
2. **No OpenAPI/Swagger specification** — REST APIs lack formal specification
3. **No API design review document** — No formal review with alternatives analysis
4. **No API style guide** — No documented conventions for endpoint naming, pagination, error responses
5. **No gRPC service reflection** — No runtime API discovery

### 🎯 ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Create API evolution roadmap (v1 → v2, deprecation policy) | `docs/api-roadmap.md` | High |
| 2 | Generate OpenAPI/Swagger spec for REST endpoints | `docs/openapi/swagger.yaml` | High |
| 3 | Write API design review with alternatives analysis | `docs/api-design-review.md` | High |
| 4 | Create API style guide (naming, pagination, errors, versioning) | `docs/api-style-guide.md` | Medium |
| 5 | Document integration methods comparison (REST vs gRPC vs messaging) | `docs/integration-methods.md` | Medium |
| 6 | Add gRPC reflection for service discovery | Service configuration | Low |

### Assessment Criteria

- [ ] API design review document with feedback
- [ ] Alternative API designs presented with trade-offs
- [ ] API evolution roadmap with versioning strategy
- [ ] Multiple integration methods demonstrated (REST, gRPC, messaging)
- [ ] OpenAPI specification published

---

## Sub 5 — Handling Error/Bug (Aggregation, RCA, Prevention)

### Level 4 Requirements

- Mampu mengumpulkan data bug/error dari banyak aplikasi ke dalam satu database
- Mampu membuat rekapitulasi dan klasifikasi data error
- Mampu melakukan root cause analysis
- Mampu merumuskan metode instrumentasi/pengukuran untuk mengotomasi pengumpulan data error
- Mampu menghasilkan solusi generik untuk mengurangi bug/error
- Mampu merumuskan perubahan dalam metode development
- Mampu menentukan langkah preventif
- Mampu merumuskan prosedur untuk memperpendek response time penanganan error
- Mampu membuat laporan post mortem
- **Assessment:** Log aggregator DB, log analysis, RCA, fix procedures, reduce errors across projects

### ✅ What EXISTS

| Evidence | Location |
|----------|----------|
| Structured logging (JSON) | `pkg/logger/` slog usage |
| OpenTelemetry tracing | Distributed trace correlation |
| OTel log bridge | `pkg/logger/` — slog → OTel log bridge → OTLP to Alloy |
| Centralized log aggregation (Loki) | All 7 services → OTel → Alloy → Loki (7d retention) |
| Prometheus metrics | `pkg/metrics/` — OTel meter provider with OTLP exporter |
| Alerting rules (8 rules) | `deploy/monitoring/prometheus/alerts.yml` — HighErrorRate, HighLatency, HighGRPCErrorRate, HighSpanErrorRate, HighSpanLatency, NoTrafficDetected, NATSConsumerLag, DatabaseSlowQueries |
| Alertmanager | `deploy/monitoring/alertmanager/` — automated error notification |
| Grafana dashboards | 3 datasources (Prometheus, Tempo, Loki) with cross-linking/correlation |
| Distributed tracing (Tempo) | Cross-service request flow for root cause investigation |
| Health check endpoints | Service health monitoring |
| Circuit breaker (error threshold) | `pkg/circuitbreaker/` |
| Graceful error handling | Error wrapping, sentinel errors |
| Race detection in CI | `-race` flag in test pipeline |
| Property-based tests | Fuzzing/property tests for edge cases |

### ❌ What's MISSING

1. **No error classification system** — No taxonomy of error types, severity levels, or categorization
2. **No RCA documentation** — No root cause analysis reports or templates
3. **No post-mortem template** — No incident post-mortem process
4. **No incident response procedure** — No documented runbook for error handling
5. **No preventive measures document** — No documented strategies for error reduction

### 🎯 ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Create error classification taxonomy | `docs/error-classification.md` | High |
| 2 | Create RCA template and sample report | `docs/templates/rca-template.md`, `docs/rca/sample-rca.md` | High |
| 3 | Create post-mortem template | `docs/templates/post-mortem-template.md` | High |
| 4 | Create incident response procedure/runbook | `docs/incident-response.md` | High |
| 5 | Document preventive measures and development process changes | `docs/error-prevention.md` | Medium |
| 6 | Create error reduction strategy across services | `docs/error-reduction-strategy.md` | Medium |

### Assessment Criteria

- [ ] Centralized log aggregation from all services into one database
- [ ] Error classification and recap report
- [ ] Root cause analysis documentation
- [ ] Automated instrumentation for error collection (OpenTelemetry + metrics)
- [ ] Generic solutions to reduce bugs (patterns, linting, testing strategies)
- [ ] Development process improvements documented
- [ ] Preventive measures defined
- [ ] Response time reduction procedures (runbooks, alerting)
- [ ] Post-mortem report template and sample

---

## Summary Matrix

| Sub-Competency | Status | Completion | Priority Items |
|----------------|--------|------------|----------------|
| 1. Collaboration Tools | 🟡 Partial | 70% | CONTRIBUTING.md, CI/CD policy |
| 2. Writing Source Code | 🟢 Strong | 80% | Coding guidelines, benchmarks, .golangci.yml |
| 3. Manage Data | 🟡 Partial | 60% | Event sourcing, outbox pattern, saga |
| 4. Integrate API | 🟡 Partial | 65% | API roadmap, OpenAPI spec, design review |
| 5. Handling Error/Bug | 🟡 Partial | 60% | RCA, post-mortem, incident response (log aggregation + alerting ✅) |

### Overall Kompetensi 2 Readiness: **~67%**

---

## Priority Roadmap

### Phase 1 — Quick Wins (1-2 days)
- [ ] `CONTRIBUTING.md` with branching strategy
- [ ] `docs/coding-guidelines.md`
- [ ] `.golangci.yml` with custom rules
- [ ] `docs/templates/post-mortem-template.md`
- [ ] `docs/templates/rca-template.md`
- [ ] `docs/error-classification.md`

### Phase 2 — Core Implementation (3-5 days)
- [ ] Go benchmarks for critical paths
- [ ] `docs/api-roadmap.md`
- [ ] `docs/api-design-review.md`
- [ ] OpenAPI/Swagger specification
- [ ] `docs/incident-response.md`
- [ ] `docs/ci-cd-policy.md`
- [x] ~~Centralized logging setup (docker-compose)~~ ✅ Loki + OTel + Alloy deployed

### Phase 3 — Advanced Patterns (5-10 days)
- [ ] Event sourcing implementation
- [ ] Outbox pattern implementation
- [ ] Saga orchestrator
- [ ] pprof profiling integration
- [ ] Performance analysis report
- [x] ~~Error metrics dashboard~~ ✅ Grafana + Prometheus + alerting deployed

---

## References

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
- [12 Factor App](https://12factor.net/)
- [Microsoft REST API Guidelines](https://github.com/microsoft/api-guidelines)
- [Event Sourcing Pattern](https://docs.microsoft.com/en-us/azure/architecture/patterns/event-sourcing)
- [Saga Pattern](https://microservices.io/patterns/data/saga.html)
- [Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html)
- [SRE Book - Postmortem Culture](https://sre.google/sre-book/postmortem-culture/)
