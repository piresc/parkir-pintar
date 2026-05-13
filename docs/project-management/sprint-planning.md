# Sprint Planning — ParkirPintar

## Document Information

| Field | Value |
|-------|-------|
| Project | ParkirPintar — Smart Parking Reservation System |
| Sprint Duration | 2 weeks |
| Team Size | 3 (1 Backend, 1 DevOps/Infra, 1 Full-stack) |
| Methodology | Scrum with Kanban elements |
| Story Point Scale | Fibonacci (1, 2, 3, 5, 8, 13) |

---

## Sprint 1: Core Domain Foundation

| Field | Value |
|-------|-------|
| Sprint Goal | Establish core domain models, repositories, and basic reservation flow |
| Duration | 2026-03-03 → 2026-03-16 |
| Status | ✅ Complete |
| Planned Points | 34 |
| Completed Points | 31 |

### Sprint Backlog

| ID | Story | Points | Status |
|----|-------|:------:|:------:|
| PP-001 | Define parking lot domain model with capacity, location, pricing | 3 | ✅ Done |
| PP-002 | Define reservation domain model with state machine (pending→confirmed→active→completed/cancelled) | 5 | ✅ Done |
| PP-003 | Define billing domain model with line items and payment status | 3 | ✅ Done |
| PP-004 | Implement PostgreSQL repository for parking lots (CRUD + spatial query) | 5 | ✅ Done |
| PP-005 | Implement PostgreSQL repository for reservations with optimistic locking | 5 | ✅ Done |
| PP-006 | Implement billing repository with transaction support | 3 | ✅ Done |
| PP-007 | Set up database migrations with golang-migrate | 2 | ✅ Done |
| PP-008 | Implement parking spot search by location (PostGIS) with radius filter | 5 | ✅ Done |
| PP-009 | Add unit tests for domain models and state transitions | 3 | ✅ Done |
| PP-010 | Set up project structure (cmd/, internal/, pkg/, proto/) | 2 | ⚠️ Partial |

### Retrospective

| Category | Notes |
|----------|-------|
| What went well | Domain modeling was solid, PostGIS integration smoother than expected |
| What didn't | Project structure took longer due to monorepo vs polyrepo debate (PP-010 partial) |
| Action items | Finalize project layout in Sprint 2, adopt hexagonal architecture pattern |

---

## Sprint 2: Service Integration

| Field | Value |
|-------|-------|
| Sprint Goal | Connect services via gRPC, implement event-driven communication with NATS, add distributed locking |
| Duration | 2026-03-17 → 2026-03-30 |
| Status | ✅ Complete |
| Planned Points | 37 |
| Completed Points | 37 |

### Sprint Backlog

| ID | Story | Points | Status |
|----|-------|:------:|:------:|
| PP-011 | Define protobuf schemas for reservation, search, and billing services | 5 | ✅ Done |
| PP-012 | Implement gRPC server for reservation service with interceptors | 5 | ✅ Done |
| PP-013 | Implement gRPC server for search service with streaming results | 5 | ✅ Done |
| PP-014 | Implement gRPC server for billing service | 3 | ✅ Done |
| PP-015 | Set up NATS JetStream for event publishing (reservation.created, reservation.cancelled) | 5 | ✅ Done |
| PP-016 | Implement event consumers for billing (on reservation.confirmed → create invoice) | 3 | ✅ Done |
| PP-017 | Implement Redis-based distributed locking for concurrent reservation prevention | 5 | ✅ Done |
| PP-018 | Add gRPC health checks and reflection for all services | 2 | ✅ Done |
| PP-019 | Implement graceful shutdown with signal handling | 2 | ✅ Done |
| PP-020 | Integration tests for full reservation→billing flow | 2 | ✅ Done |

### Retrospective

| Category | Notes |
|----------|-------|
| What went well | Full velocity achieved, NATS JetStream simpler than Kafka, distributed lock works reliably |
| What didn't | gRPC streaming for search added complexity; may simplify to unary with pagination |
| Action items | Add deadline/timeout configuration per RPC, document event schema contracts |

---

## Sprint 3: Observability & Security

| Field | Value |
|-------|-------|
| Sprint Goal | Implement full observability pipeline (traces, metrics, logs) and secure all endpoints |
| Duration | 2026-03-31 → 2026-04-13 |
| Status | ✅ Complete |
| Planned Points | 39 |
| Completed Points | 36 |

### Sprint Backlog

| ID | Story | Points | Status |
|----|-------|:------:|:------:|
| PP-021 | Integrate OpenTelemetry SDK with trace propagation across gRPC calls | 5 | ✅ Done |
| PP-022 | Set up OTel Collector with OTLP receiver, export to Tempo (traces) and Prometheus (metrics) | 5 | ✅ Done |
| PP-023 | Add structured logging with slog, correlate with trace IDs | 3 | ✅ Done |
| PP-024 | Deploy Grafana with pre-configured dashboards (RED metrics, resource usage) | 3 | ✅ Done |
| PP-025 | Implement JWT-based authentication with RS256 signing | 5 | ✅ Done |
| PP-026 | Add RBAC middleware (roles: driver, operator, admin) | 3 | ✅ Done |
| PP-027 | Implement rate limiting per IP and per authenticated user (token bucket) | 5 | ✅ Done |
| PP-028 | Add input validation middleware with custom error responses | 3 | ✅ Done |
| PP-029 | Set up Loki for log aggregation with Grafana integration | 3 | ✅ Done |
| PP-030 | Implement request/response logging interceptor with PII redaction | 2 | ✅ Done |
| PP-031 | Add Prometheus alerting rules for SLO burn rate | 2 | ❌ Moved |

### Retrospective

| Category | Notes |
|----------|-------|
| What went well | OTel integration excellent, trace propagation works end-to-end, auth solid |
| What didn't | PP-031 (alerting rules) moved to Sprint 5 — needed SLO definitions first |
| Action items | Define SLOs before creating alerts, add security headers to HTTP gateway |

---

## Sprint 4: CI/CD & Infrastructure

| Field | Value |
|-------|-------|
| Sprint Goal | Automate build/test/deploy pipeline, containerize services, provision infrastructure |
| Duration | 2026-04-14 → 2026-04-27 |
| Status | ✅ Complete |
| Planned Points | 40 |
| Completed Points | 40 |

### Sprint Backlog

| ID | Story | Points | Status |
|----|-------|:------:|:------:|
| PP-032 | Create multi-stage Dockerfiles for all services (build + distroless runtime) | 3 | ✅ Done |
| PP-033 | Set up GitHub Actions CI: lint (golangci-lint), test, build | 5 | ✅ Done |
| PP-034 | Add Trivy container scanning to CI pipeline (fail on HIGH/CRITICAL) | 3 | ✅ Done |
| PP-035 | Configure GitHub Actions CD: build image → push GHCR → deploy via Coolify webhook | 5 | ✅ Done |
| PP-036 | Write docker-compose.yml for full local development stack | 3 | ✅ Done |
| PP-037 | Deploy to Coolify on VPS with Caddy reverse proxy and auto-TLS | 5 | ✅ Done |
| PP-038 | Configure Watchtower for automated image pulls with health-check validation | 3 | ✅ Done |
| PP-039 | Write Terraform modules for GCP Cloud Run (future production) | 5 | ✅ Done |
| PP-040 | Set up branch protection rules and PR template | 2 | ✅ Done |
| PP-041 | Implement database migration in CI (test against real PostgreSQL) | 3 | ✅ Done |
| PP-042 | Add Go test coverage reporting with threshold (80%) | 3 | ✅ Done |

### Retrospective

| Category | Notes |
|----------|-------|
| What went well | Full CI/CD pipeline operational, Coolify deployment smooth, Terraform modules ready |
| What didn't | Watchtower initially pulled untagged images; fixed with label filtering |
| Action items | Add smoke tests post-deploy, implement blue-green for production |

---

## Sprint 5: Quality & Documentation (Current)

| Field | Value |
|-------|-------|
| Sprint Goal | Validate system quality through load testing and DAST, formalize SLOs, complete documentation |
| Duration | 2026-04-28 → 2026-05-11 (extended to 2026-05-18 due to scope) |
| Status | 🔄 In Progress |
| Planned Points | 42 |
| Completed Points | 28 |

### Sprint Backlog

| ID | Story | Points | Status |
|----|-------|:------:|:------:|
| PP-043 | Write k6 load test scripts for search, reservation, and billing flows | 5 | ✅ Done |
| PP-044 | Run load tests: establish baseline (target: 500 concurrent users, P99 < 1s) | 3 | ✅ Done |
| PP-045 | Integrate OWASP ZAP DAST scan into CI pipeline | 5 | ✅ Done |
| PP-046 | Define SLOs: availability (99.9%), latency (P99 < 500ms search, < 1s reserve), error rate (<0.1%) | 3 | ✅ Done |
| PP-047 | Create SLO dashboard in Grafana with burn rate alerts | 3 | ✅ Done |
| PP-048 | Write Architecture Decision Records (ADRs) for key decisions | 5 | 🔄 In Progress |
| PP-049 | Create project management artifacts (risk register, WBS, roadmap) | 5 | 🔄 In Progress |
| PP-050 | Write API documentation with protobuf-generated docs | 3 | ✅ Done |
| PP-051 | Create runbooks for common operational scenarios | 3 | 🔄 In Progress |
| PP-052 | Implement error budget policy and escalation procedure | 2 | 📋 To Do |
| PP-053 | Final integration test suite with contract testing | 5 | ✅ Done |

### Retrospective (Preliminary)

| Category | Notes |
|----------|-------|
| What went well | Load tests revealed connection pool issue (fixed), DAST found no critical vulns |
| What didn't | Documentation scope larger than estimated, sprint extended by 1 week |
| Action items | Complete remaining docs, prepare for assessment demo |

---

## Velocity Chart

| Sprint | Planned | Completed | Velocity |
|--------|:-------:|:---------:|:--------:|
| Sprint 1 | 34 | 31 | 31 |
| Sprint 2 | 37 | 37 | 37 |
| Sprint 3 | 39 | 36 | 36 |
| Sprint 4 | 40 | 40 | 40 |
| Sprint 5 | 42 | 28* | — |

*Sprint 5 in progress*

```
Velocity Trend
45 │
40 │              ┌───┐       ┌───┐
35 │       ┌───┐  │   │  ┌───┐│   │
30 │  ┌───┐│   │  │   │  │   ││   │
25 │  │   ││   │  │   │  │   ││   │
20 │  │   ││   │  │   │  │   ││   │
15 │  │   ││   │  │   │  │   ││   │
10 │  │   ││   │  │   │  │   ││   │
 5 │  │   ││   │  │   │  │   ││   │
 0 └──┴───┴┴───┴──┴───┴──┴───┴┴───┴──
    SP1    SP2    SP3    SP4
    (31)   (37)   (36)   (40)
```

**Average Velocity:** 36 points/sprint
**Trend:** Increasing (team ramping up, better estimation accuracy)

## Definition of Done

A story is considered Done when:

1. Code is written and follows project conventions (golangci-lint passes)
2. Unit tests written with ≥80% coverage on new code
3. Integration tests pass (where applicable)
4. Code reviewed and approved by at least 1 team member
5. CI pipeline passes (lint, test, build, scan)
6. Documentation updated (API docs, ADRs if architectural)
7. Deployed to staging environment successfully
8. Product Owner accepts the deliverable
