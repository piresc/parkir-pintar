# Gap Analysis: Kompetensi 0 — Software Requirement (Level 4)

**Project:** ParkirPintar — Smart Parking Marketplace  
**Date:** 2026-05-12  
**Architecture:** 7 Go microservices, gRPC, NATS JetStream, PostgreSQL, Redis, React frontend  
**Target Level:** 4

---

## Sub-Kompetensi 1: Menggali Kebutuhan User

### Level 4 Requirements

- Mampu memprediksi dan mengklarifikasi skenario alternatif yang mungkin terjadi dalam aplikasi
- Mampu memperkirakan kebutuhan integrasi dengan aplikasi/service lain untuk diklarifikasi pada sesi diskusi

### What EXISTS

| Evidence | File Path | Description |
|----------|-----------|-------------|
| Payment flow scenarios | `docs/superpowers/specs/2026-05-06-payment-flow-design.md` | Documents before/after scenarios for payment-before-confirmation, including success/failure/timeout paths |
| Alternative flow options | `docs/superpowers/plans/2026-05-07-two-phase-payment-flow.md` | Explores Option A/B/C with tradeoffs for payment architecture |
| Integration mapping | `docs/architecture.md` | Shows gRPC sync and NATS async communication between 7 services |
| Service contracts | `proto/*/v1/*.proto` (6 files) | Defines inter-service API contracts for payment, presence, notification, reservation, billing, search |
| Database separation rationale | `docs/superpowers/plans/2026-05-07-database-per-service-separation.md` | Predicts integration needs when moving from shared DB to schema-per-service |
| Docker integration map | `docker-compose.yml` | Shows all infrastructure dependencies (PostgreSQL, Redis, NATS) |

### What's MISSING

1. **No formal requirements elicitation document** — No record of user interviews, stakeholder sessions, or discovery workshops
2. **No scenario matrix** — Alternative/exception scenarios exist in design docs but not consolidated into a structured scenario catalog
3. **No external integration analysis** — No document predicting integration needs with external systems (e.g., payment gateways like Midtrans/Xendit, maps API, government parking APIs, CCTV/IoT systems)
4. **No stakeholder communication log** — No evidence of clarification sessions or Q&A with product owner

### ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Create requirements elicitation document with user personas, interview questions, and discovery findings | `docs/requirements/01-requirements-elicitation.md` | HIGH |
| 2 | Build scenario catalog covering happy path, alternative, and exception scenarios per feature | `docs/requirements/02-scenario-catalog.md` | HIGH |
| 3 | Document external integration analysis (payment gateway, maps, IoT sensors, government APIs) | `docs/requirements/03-external-integration-analysis.md` | HIGH |
| 4 | Create stakeholder communication log template with clarification records | `docs/requirements/04-stakeholder-communication-log.md` | MEDIUM |

### Assessment Criteria

- [ ] Scenario catalog covers at least 3 alternative scenarios per major feature (reservation, payment, search)
- [ ] External integration document identifies at least 5 potential integrations with pros/cons
- [ ] Evidence of predicting edge cases before they were raised by stakeholders
- [ ] Clarification questions documented with responses and decisions made

---

## Sub-Kompetensi 2: Menganalisa Kebutuhan User

### Level 4 Requirements

- Mampu menganalisa keterkaitan aplikasi yang sedang dibangun dengan aplikasi/service lain
- Mampu menganalisa dampak kebutuhan user terhadap kebutuhan resource (RAM, Storage, Network) di server
- Menganalisa dan merekomendasikan solusi alternatif yang lebih baik daripada permintaan user
- Menjelaskan solusi alternatif, kelebihan, dan kekurangannya kepada user

### What EXISTS

| Evidence | File Path | Description |
|----------|-----------|-------------|
| Service dependency analysis | `docs/architecture.md` | Maps sync (gRPC) and async (NATS) dependencies between all 7 services |
| Alternative solutions with tradeoffs | `docs/superpowers/plans/2026-05-07-two-phase-payment-flow.md` | Option A (single-phase), Option B (two-phase interactive), Option C (webhook-based) with pros/cons |
| DB separation alternatives | `docs/superpowers/plans/2026-05-07-database-per-service-separation.md` | Analyzes schema-per-service vs full DB separation, rollback strategy |
| Payment flow design alternatives | `docs/superpowers/specs/2026-05-06-payment-flow-design.md` | Before/after analysis with rationale for chosen approach |
| Infrastructure config | `docker-compose.yml` | PostgreSQL (max 25 conns), Redis (pool 10), NATS JetStream — resource allocation visible |
| Resilience patterns | `docs/architecture.md` | Circuit breaker, retry, distributed lock, idempotency, singleflight documented |

### What's MISSING

1. **No resource impact analysis** — No document projecting RAM, Storage, Network, and CPU requirements per service under expected load
2. **No capacity planning document** — No projections for concurrent users, requests/second, storage growth over time
3. **No formal alternatives comparison matrix** — Alternatives exist in narrative form but not in a structured decision matrix format
4. **No cost-benefit analysis** — No quantified comparison of solution alternatives (development time, infrastructure cost, maintenance burden)
5. **No load/performance baseline** — No benchmarks establishing current resource consumption

### ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Create resource impact analysis with RAM/Storage/Network projections per service | `docs/requirements/05-resource-impact-analysis.md` | HIGH |
| 2 | Build capacity planning document with growth projections (users, storage, bandwidth) | `docs/requirements/06-capacity-planning.md` | HIGH |
| 3 | Create formal alternatives comparison matrix (structured table format with scoring) | `docs/requirements/07-alternatives-comparison-matrix.md` | HIGH |
| 4 | Document cost-benefit analysis for key architectural decisions | `docs/requirements/08-cost-benefit-analysis.md` | MEDIUM |
| 5 | Run load tests and document baseline resource consumption per service | `docs/requirements/09-performance-baseline.md` | MEDIUM |

### Resource Impact Analysis Template (suggested content)

```markdown
## Per-Service Resource Projections

| Service | Base RAM | Per-1K-Users RAM | Storage/Month | Network (egress) |
|---------|----------|------------------|---------------|-------------------|
| Gateway | 64 MB | +20 MB | N/A | 500 MB/day |
| Search | 128 MB | +50 MB | 100 MB | 200 MB/day |
| Reservation | 96 MB | +30 MB | 500 MB | 150 MB/day |
| Billing | 64 MB | +15 MB | 200 MB | 100 MB/day |
| Payment | 64 MB | +15 MB | 150 MB | 100 MB/day |
| Presence | 96 MB | +40 MB | 50 MB | 300 MB/day |
| Notification | 48 MB | +10 MB | 50 MB | 200 MB/day |

## Infrastructure Dependencies
| Component | Base Resource | Scaling Factor |
|-----------|--------------|----------------|
| PostgreSQL | 256 MB RAM, 1 GB disk | +100 MB RAM per 10K reservations |
| Redis | 64 MB RAM | +50 MB per 10K active sessions |
| NATS | 128 MB RAM | +20 MB per 1K msg/sec throughput |
```

### Assessment Criteria

- [ ] Resource impact document covers all 7 services + 3 infrastructure components
- [ ] Capacity planning includes 3-month, 6-month, and 12-month projections
- [ ] At least 3 architectural decisions have formal comparison matrices with scoring
- [ ] Alternatives clearly state kelebihan (pros) and kekurangan (cons) in user-facing language
- [ ] Evidence of recommending a better solution than what was originally requested

---

## Sub-Kompetensi 3: Membuat Spesifikasi Kebutuhan

### Level 4 Requirements

- Membuat spesifikasi API
- Membuat aturan access control terhadap fitur aplikasi
- Membuat spesifikasi non-functional (security, scalability, performance, configurability)

### What EXISTS

| Evidence | File Path | Description |
|----------|-----------|-------------|
| API reference (basic) | `docs/api.md` | Health endpoints + example CRUD + auth methods documented |
| gRPC API contracts | `proto/*/v1/*.proto` (6 files) | Full protobuf definitions for all inter-service RPCs |
| Auth implementation | Gateway middleware | JWT (HMAC-SHA256) + API Key (`X-API-Key`) + per-IP rate limiting |
| Access control (code) | `internal/gateway/handler/` | Route-level auth middleware applied to API routes |
| Security patterns | `docs/architecture.md` | CORS (explicit origins), rate limiting, service-to-service auth |
| Configuration | `pkg/config/` | Env-based configuration with structured config types |
| Resilience specs | `docs/architecture.md` | Circuit breaker, retry, distributed lock, idempotency, singleflight |

### What's MISSING

1. **No OpenAPI/Swagger specification** — REST API has no machine-readable spec; only a basic markdown table exists
2. **No formal access control matrix** — Auth is implemented in code but no document maps roles → endpoints → permissions
3. **No explicit scalability specification** — No document defining scaling targets, horizontal scaling strategy, or auto-scaling rules
4. **No performance specification** — No defined SLAs (response time P95/P99, throughput targets, availability %)
5. **No security specification document** — Security measures exist in code but no consolidated security spec (threat model, data classification, encryption at rest/transit)
6. **No configurability specification** — Config exists but no document listing all configurable parameters with defaults, ranges, and descriptions

### ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Generate OpenAPI 3.0 spec for all REST endpoints (Gateway) | `docs/specs/openapi.yaml` | HIGH |
| 2 | Create access control matrix (roles × endpoints × permissions) | `docs/specs/access-control-matrix.md` | HIGH |
| 3 | Write non-functional requirements spec (security, scalability, performance, configurability) | `docs/specs/non-functional-requirements.md` | HIGH |
| 4 | Create security specification with threat model | `docs/specs/security-specification.md` | HIGH |
| 5 | Document all configuration parameters with descriptions and defaults | `docs/specs/configuration-reference.md` | MEDIUM |
| 6 | Define SLA targets (latency, throughput, availability, error budget) | `docs/specs/sla-targets.md` | MEDIUM |

### Access Control Matrix Template (suggested content)

```markdown
## Role Definitions
| Role | Description |
|------|-------------|
| anonymous | Unauthenticated user |
| driver | Authenticated parking user |
| operator | Parking lot operator |
| admin | System administrator |

## Endpoint Permissions
| Endpoint | anonymous | driver | operator | admin |
|----------|-----------|--------|----------|-------|
| GET /api/v1/spots/search | ✅ | ✅ | ✅ | ✅ |
| POST /api/v1/reservations | ❌ | ✅ | ❌ | ✅ |
| POST /api/v1/reservations/:id/confirm | ❌ | ✅ (own) | ❌ | ✅ |
| POST /api/v1/reservations/:id/checkout | ❌ | ✅ (own) | ❌ | ✅ |
| GET /api/v1/billing/invoices | ❌ | ✅ (own) | ✅ (lot) | ✅ |
| POST /api/v1/presence/checkin | ❌ | ✅ | ✅ | ✅ |
```

### Non-Functional Requirements Template (suggested content)

```markdown
## Performance
- API response time: P95 < 200ms, P99 < 500ms
- Search queries: P95 < 100ms (Redis-cached)
- Throughput: 1000 req/sec per gateway instance

## Scalability
- Horizontal scaling: all services stateless, scale via replicas
- Database: read replicas for search, connection pooling (max 25)
- Message queue: NATS JetStream with consumer groups

## Security
- Transport: TLS 1.3 for all external, mTLS for internal gRPC
- Authentication: JWT RS256 with 15min expiry + refresh tokens
- Data: PII encrypted at rest, audit logging for sensitive operations

## Configurability
- All settings via environment variables
- Feature flags for gradual rollout
- Per-service configuration isolation
```

### Assessment Criteria

- [ ] OpenAPI spec covers all REST endpoints with request/response schemas
- [ ] Access control matrix covers all roles and all endpoints
- [ ] Non-functional spec defines measurable targets for security, scalability, performance
- [ ] Configuration reference lists all env vars with types, defaults, and valid ranges
- [ ] Specs are detailed enough for another developer to implement from scratch

---

## Sub-Kompetensi 4: Mengelola Perubahan Kebutuhan

### Level 4 Requirements

- Mengkalkulasi dampak perubahan terhadap aplikasi/service lain yang terkait
- Mengkalkulasi dampak perubahan terhadap kebutuhan resource
- Menganalisa solusi alternatif selain perubahan yang diminta user
- Menjelaskan detail solusi alternatif, kelebihan, dan kekurangannya kepada user

### What EXISTS

| Evidence | File Path | Description |
|----------|-----------|-------------|
| Change impact (payment flow) | `docs/superpowers/plans/2026-05-07-two-phase-payment-flow.md` | Lists all affected files, services, and describes migration path |
| Change impact (DB separation) | `docs/superpowers/plans/2026-05-07-database-per-service-separation.md` | Comprehensive file map of changes, rollback strategy, affected services |
| Alternative solutions | `docs/superpowers/specs/2026-05-06-payment-flow-design.md` | Before/after comparison with rationale |
| Service dependency map | `docs/architecture.md` | Shows which services depend on which — useful for impact analysis |
| Migration files | `db/migrations/` | Versioned migrations with up/down scripts for rollback |

### What's MISSING

1. **No formal change impact analysis template** — Impact analysis exists ad-hoc in plan docs but no standardized template
2. **No resource impact calculator for changes** — Changes don't quantify RAM/Storage/Network delta
3. **No change request log** — No tracking of requirement changes over time with status and decisions
4. **No formal alternatives documentation for changes** — Alternatives are embedded in plans, not in a dedicated comparison format
5. **No rollback cost analysis** — Rollback strategies exist but no quantified cost/risk assessment
6. **No dependency impact matrix** — No quick-reference showing "if Service X changes, Services Y and Z are affected"

### ACTION ITEMS

| # | Task | Output File | Priority |
|---|------|-------------|----------|
| 1 | Create change impact analysis template with service/resource/timeline sections | `docs/templates/change-impact-analysis-template.md` | HIGH |
| 2 | Build service dependency impact matrix (change in X → impact on Y) | `docs/specs/dependency-impact-matrix.md` | HIGH |
| 3 | Create change request log tracking all requirement changes | `docs/requirements/10-change-request-log.md` | HIGH |
| 4 | Retroactively document resource impact for payment flow change | `docs/change-impact/payment-flow-resource-impact.md` | MEDIUM |
| 5 | Retroactively document resource impact for DB separation change | `docs/change-impact/db-separation-resource-impact.md` | MEDIUM |
| 6 | Create alternatives analysis template with structured scoring | `docs/templates/alternatives-analysis-template.md` | MEDIUM |

### Change Impact Analysis Template (suggested content)

```markdown
# Change Impact Analysis: [Change Title]

## 1. Change Description
- Requested by: [stakeholder]
- Date: [date]
- Summary: [what changed]

## 2. Service Impact
| Service | Impact Level | Description |
|---------|-------------|-------------|
| Gateway | LOW/MED/HIGH | [what changes] |
| Reservation | LOW/MED/HIGH | [what changes] |
| ... | ... | ... |

## 3. Resource Impact
| Resource | Before | After | Delta |
|----------|--------|-------|-------|
| RAM (total) | X MB | Y MB | +Z MB |
| Storage (monthly) | X GB | Y GB | +Z GB |
| Network (daily) | X MB | Y MB | +Z MB |
| DB connections | X | Y | +Z |

## 4. Timeline Impact
- Estimated effort: [days/weeks]
- Services blocked during migration: [list]
- Rollback time if failed: [hours]

## 5. Alternative Solutions
| Option | Pros | Cons | Effort | Recommendation |
|--------|------|------|--------|----------------|
| A: [requested change] | ... | ... | X days | |
| B: [alternative 1] | ... | ... | Y days | ✅ Recommended |
| C: [alternative 2] | ... | ... | Z days | |

## 6. Decision
- Chosen: [option]
- Rationale: [why]
- Approved by: [stakeholder]
```

### Dependency Impact Matrix (suggested content)

```markdown
## Service Dependency Impact Matrix

When a service changes its API/schema, which other services are affected?

| Changed Service | Gateway | Search | Reservation | Billing | Payment | Presence | Notification |
|----------------|---------|--------|-------------|---------|---------|----------|--------------|
| Gateway | — | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Search | ✅ route | — | ❌ | ❌ | ❌ | ❌ | ❌ |
| Reservation | ✅ route | ✅ NATS | — | ✅ gRPC | ✅ gRPC | ❌ | ✅ NATS |
| Billing | ✅ route | ❌ | ✅ gRPC | — | ❌ | ❌ | ✅ NATS |
| Payment | ✅ route | ❌ | ✅ gRPC | ❌ | — | ❌ | ✅ NATS |
| Presence | ✅ route | ❌ | ❌ | ❌ | ❌ | — | ✅ NATS |
| Notification | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | — |
```

### Assessment Criteria

- [ ] At least 2 changes have formal impact analysis documents with resource quantification
- [ ] Dependency impact matrix is complete and accurate against actual code
- [ ] Change request log tracks at least 3 changes with before/after status
- [ ] Each change analysis includes at least 2 alternative solutions with pros/cons
- [ ] Alternatives are explained in language suitable for non-technical stakeholders

---

## Summary: Overall Gap Status

| Sub-Kompetensi | Existing Evidence | Gap Level | Priority Actions |
|----------------|-------------------|-----------|-----------------|
| 1. Menggali kebutuhan user | Partial (scenarios in design docs) | **MEDIUM** | Formalize elicitation docs, scenario catalog, external integration analysis |
| 2. Menganalisa kebutuhan user | Partial (architecture + alternatives in plans) | **HIGH** | Resource impact analysis, capacity planning, formal comparison matrix |
| 3. Membuat spesifikasi kebutuhan | Partial (proto specs + auth code) | **HIGH** | OpenAPI spec, access control matrix, non-functional requirements doc |
| 4. Mengelola perubahan kebutuhan | Partial (impact in plan docs) | **MEDIUM** | Formalize templates, dependency matrix, change log, resource delta |

### Recommended Execution Order

1. `docs/specs/non-functional-requirements.md` — Covers Sub 3 (security, scalability, performance)
2. `docs/specs/openapi.yaml` — Covers Sub 3 (API specification)
3. `docs/requirements/05-resource-impact-analysis.md` — Covers Sub 2 (resource analysis)
4. `docs/specs/access-control-matrix.md` — Covers Sub 3 (access control)
5. `docs/specs/dependency-impact-matrix.md` — Covers Sub 4 (change impact)
6. `docs/requirements/02-scenario-catalog.md` — Covers Sub 1 (alternative scenarios)
7. `docs/requirements/03-external-integration-analysis.md` — Covers Sub 1 (integration needs)
8. `docs/templates/change-impact-analysis-template.md` — Covers Sub 4 (change management)
9. `docs/requirements/07-alternatives-comparison-matrix.md` — Covers Sub 2 (formal alternatives)
10. `docs/requirements/10-change-request-log.md` — Covers Sub 4 (change tracking)

---

## Quick Reference: File Paths for Evidence

### Already exists (cite these in assessment)
- `docs/architecture.md` — service dependencies, resilience patterns
- `docs/superpowers/specs/2026-05-06-payment-flow-design.md` — scenario analysis
- `docs/superpowers/plans/2026-05-07-two-phase-payment-flow.md` — alternatives with tradeoffs
- `docs/superpowers/plans/2026-05-07-database-per-service-separation.md` — change impact + rollback
- `proto/*/v1/*.proto` — API contracts (6 files)
- `docker-compose.yml` — infrastructure integration map
- `pkg/config/` — configurability evidence
- `docs/api.md` — basic API documentation

### Needs to be created (action items)
- `docs/requirements/` — elicitation, scenarios, resource analysis, alternatives
- `docs/specs/` — OpenAPI, access control, NFR, security, SLA, config reference
- `docs/templates/` — change impact template, alternatives template
- `docs/change-impact/` — retroactive impact analyses for existing changes
