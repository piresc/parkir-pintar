# Kompetensi 1 — Software Design: Level 4 Gap Analysis

> **Project:** parkir-pintar (Go Microservices Smart Parking)
> **Assessment Date:** 2026-05-12
> **Current Score:** 60%
> **Target:** Level 4 (Senior/Lead Developer)

---

## Overview

Kompetensi 1 menilai kemampuan merancang arsitektur, komponen, struktur data, pemilihan teknologi, dan dokumentasi desain. Parkir-pintar memiliki implementasi yang kuat (7 microservices, gRPC, NATS JetStream, CQRS-lite) namun lemah di dokumentasi formal keputusan arsitektur dan analisis perbandingan.

---

## Sub 1 — Design Application Architecture

### Level 4 Requirements

- Mampu menganalisa berbagai alternatif desain arsitektur aplikasi
- Mampu melakukan review dan memberikan feedback terhadap rancangan Medium Developer
- Dapat menganalisa dampak desain aplikasi terhadap deployment, operasional, dan pemeliharaan aplikasi
- **Assessment:** Menyediakan beberapa pilihan arsitektur, impact analysis, architecture design review

### What Currently EXISTS ✅

| Evidence | File Path | Description |
|----------|-----------|-------------|
| System Architecture Diagram | `README.md` (lines 25-62) | Mermaid diagram showing 7 services, communication patterns, data layer |
| Architecture Documentation | `docs/architecture.md` | Clean architecture layers, middleware chains, resilience patterns, data flow |
| Cloud Deployment Reference | `README.md` (section "Cloud Deployment Reference Architecture") | High-level deployment topology |
| Code Review | `REVIEW.md` | Severity-ranked findings (Critical/High/Medium/Low) with actionable fixes |
| Design Specs with Alternatives | `docs/superpowers/specs/2026-05-06-payment-flow-design.md` | Before/after design with target flow |
| Implementation Plans | `docs/superpowers/plans/` (7 files) | Some plans include option analysis (e.g., two-phase payment has Option a/b/c) |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No formal ADR (Architecture Decision Record) directory | Cannot trace why architectural decisions were made | 🔴 High |
| No architecture alternatives comparison document | Cannot demonstrate analysis of multiple options | 🔴 High |
| No impact analysis matrix per architecture option | Cannot show deployment/ops/maintenance tradeoffs | 🔴 High |
| No formal architecture review process documentation | Cannot demonstrate structured review capability | 🟡 Medium |
| No architecture review board/approval workflow | No evidence of governance process | 🟡 Medium |

### ACTION ITEMS

| # | Task | Output File | Effort |
|---|------|-------------|--------|
| 1.1 | Create ADR directory with template | `docs/adr/000-template.md` | 1h |
| 1.2 | Write ADR: "Why Microservices over Monolith" | `docs/adr/001-microservices-architecture.md` | 2h |
| 1.3 | Write ADR: "Why gRPC over REST for inter-service" | `docs/adr/002-grpc-communication.md` | 1.5h |
| 1.4 | Write ADR: "Why NATS JetStream over Kafka/RabbitMQ" | `docs/adr/003-nats-jetstream-messaging.md` | 2h |
| 1.5 | Write Architecture Alternatives Comparison | `docs/adr/004-architecture-alternatives-comparison.md` | 3h |
| 1.6 | Write Impact Analysis (deployment, ops, maintenance) | `docs/architecture-impact-analysis.md` | 3h |
| 1.7 | Document Architecture Review Process | `docs/architecture-review-process.md` | 1.5h |

### Assessment Criteria

- [ ] Minimal 3 ADRs with alternatives considered, decision rationale, and consequences
- [ ] Architecture alternatives comparison with pros/cons matrix (min 3 options)
- [ ] Impact analysis covering: deployment complexity, operational cost, maintenance burden, scalability ceiling
- [ ] Evidence of architecture review feedback (review comments on design docs)

---

## Sub 2 — Design Component and Module

### Level 4 Requirements

- Mampu melakukan review terhadap struktur komponen/modul yang dibuat oleh Medium Developer
- Mampu membuat desain alternatif terhadap requirement aplikasi
- Mampu merancang arsitektur berbasis messaging, aplikasi terdistribusi, microservices, dan event sourcing
- **Assessment:** Review modul, alternatif desain, rancangan arsitektur terdistribusi/messaging/microservices/event sourcing

### What Currently EXISTS ✅

| Evidence | File Path | Description |
|----------|-----------|-------------|
| Code Review with Module Analysis | `REVIEW.md` | Severity-ranked review of all 7 services, identifies structural issues |
| Messaging Architecture | `docs/architecture.md` (lines 28-35) | NATS JetStream with 4 streams, 12 subjects, pub/sub pattern |
| Distributed System Design | All `internal/*/` services | 7 microservices with gRPC, distributed locks, circuit breakers |
| Microservices with Schema-per-Service | `migrations/` per service | Full database isolation per service |
| CQRS-lite Pattern | `internal/search/` | Read model synced via NATS events from reservation service |
| Alternative Design Specs | `docs/superpowers/specs/2026-05-06-payment-flow-design.md` | Before/after with detailed target flow |
| Resilience Patterns | `docs/architecture.md` (lines 76-84) | Circuit breaker, retry, distributed lock, idempotency, singleflight |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No full event sourcing (event store + replay) | Cannot demonstrate event sourcing competency | 🔴 High |
| No formal alternative design documents per component | Cannot show systematic design exploration | 🟡 Medium |
| No CODEOWNERS file | Cannot demonstrate review assignment governance | 🟡 Medium |
| No component interaction matrix | Cannot show dependency analysis | 🟢 Low |

### ACTION ITEMS

| # | Task | Output File | Effort |
|---|------|-------------|--------|
| 2.1 | Implement event sourcing for billing/payment domain | `internal/billing/eventsource/` | 8h |
| 2.2 | Write Event Sourcing ADR with design rationale | `docs/adr/005-event-sourcing-billing.md` | 2h |
| 2.3 | Create alternative design document for reservation module | `docs/design-alternatives/reservation-module.md` | 3h |
| 2.4 | Create alternative design document for payment flow | `docs/design-alternatives/payment-flow.md` | 2h |
| 2.5 | Add CODEOWNERS file | `.github/CODEOWNERS` | 0.5h |
| 2.6 | Write component interaction matrix | `docs/component-interaction-matrix.md` | 2h |
| 2.7 | Document module review checklist | `docs/module-review-checklist.md` | 1h |

### Assessment Criteria

- [ ] Event sourcing implementation with event store, event replay, and snapshot capability
- [ ] Minimum 2 alternative design documents showing different approaches to same requirement
- [ ] CODEOWNERS file mapping services to reviewers
- [ ] Evidence of module review (REVIEW.md already satisfies partially)
- [ ] Messaging architecture documented with stream topology, consumer groups, retry policies

---

## Sub 3 — Design Data Structure

### Level 4 Requirements

- Menguasai berbagai model struktur data dan pilihan produk/teknologi
- Mampu menganalisa dan membandingkan kesesuaian berbagai pilihan model struktur data
- Dapat menjelaskan kelebihan dan kekurangan masing-masing pilihan

### What Currently EXISTS ✅

| Evidence | File Path | Description |
|----------|-----------|-------------|
| PostgreSQL with proper schema design | `README.md` (ERD section) | Normalized tables, CHECK constraints, partial unique indexes |
| Redis for caching & distributed locks | `pkg/redislock/`, `internal/search/repository/` | Key-value store for ephemeral data |
| NATS JetStream for event streaming | `pkg/nats/` | Message persistence with replay capability |
| Multiple data access patterns | Various repositories | SQL queries, Redis GET/SET, NATS pub/sub |
| Schema-per-service isolation | `migrations/` directories | Each service owns its schema |
| CQRS read model | `internal/search/` | Denormalized read model synced via events |
| Indexed query patterns | Migration files | Proper indexes for all query patterns |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No data storage technology comparison document | Cannot demonstrate analysis capability | 🔴 High |
| No justification for PostgreSQL over alternatives (MySQL, MongoDB, CockroachDB) | Cannot explain decision rationale | 🔴 High |
| No justification for Redis over alternatives (Memcached, Hazelcast, Valkey) | Missing tradeoff analysis | 🟡 Medium |
| No capacity planning for data growth | Cannot show operational foresight | 🟡 Medium |
| No data model evolution strategy document | Missing long-term planning | 🟡 Medium |

### ACTION ITEMS

| # | Task | Output File | Effort |
|---|------|-------------|--------|
| 3.1 | Write data storage technology comparison | `docs/adr/006-data-storage-comparison.md` | 3h |
| 3.2 | Write PostgreSQL vs alternatives analysis | `docs/data-technology-decisions/relational-db-comparison.md` | 2h |
| 3.3 | Write Redis vs alternatives analysis | `docs/data-technology-decisions/cache-store-comparison.md` | 1.5h |
| 3.4 | Write NATS vs Kafka vs RabbitMQ comparison | `docs/data-technology-decisions/message-broker-comparison.md` | 2h |
| 3.5 | Create capacity planning document | `docs/capacity-planning.md` | 3h |
| 3.6 | Document data model evolution strategy | `docs/data-evolution-strategy.md` | 2h |

### Assessment Criteria

- [ ] Comparison document covering minimum 3 database technologies with pros/cons matrix
- [ ] Comparison document covering minimum 2 caching technologies
- [ ] Comparison document covering minimum 3 message broker technologies
- [ ] Each comparison includes: performance characteristics, consistency model, operational complexity, cost, ecosystem maturity
- [ ] Capacity planning with growth projections (1 year, 3 year)

---

## Sub 4 — Select and Use Framework and Library

### Level 4 Requirements

- Menguasai beberapa alternatif technology stack
- Dapat menjelaskan kelebihan dan kekurangan masing-masing alternatif
- Dapat mengkombinasikan stack dari berbagai platform/bahasa pemrograman

### What Currently EXISTS ✅

| Evidence | File Path | Description |
|----------|-----------|-------------|
| Go backend stack | `go.mod` | chi, sqlx, go-redis, nats.go, grpc-go, otel, zap |
| React frontend | `frontend/` | TypeScript, React, Vite |
| Protocol Buffers (cross-language) | `proto/` | 6 .proto files defining service contracts |
| Multi-runtime combination | `docker-compose.yml` | Go + Node.js + PostgreSQL + Redis + NATS |
| Internal library ecosystem | `pkg/` (14+ packages) | Reusable libraries: circuitbreaker, redislock, nats, crypto, etc. |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No technology stack comparison document | Cannot demonstrate evaluation capability | 🔴 High |
| No framework comparison (chi vs gin vs fiber vs echo) | Missing framework selection rationale | 🟡 Medium |
| No ORM/data-access comparison (sqlx vs gorm vs ent vs sqlc) | Missing library selection rationale | 🟡 Medium |
| No polyglot microservice example | Limited cross-platform demonstration | 🟢 Low |
| No evaluation criteria/scoring matrix for tech selection | No systematic selection process | 🟡 Medium |

### ACTION ITEMS

| # | Task | Output File | Effort |
|---|------|-------------|--------|
| 4.1 | Write technology stack comparison document | `docs/adr/007-technology-stack-selection.md` | 3h |
| 4.2 | Write Go web framework comparison | `docs/tech-stack-decisions/go-web-framework.md` | 2h |
| 4.3 | Write data access library comparison | `docs/tech-stack-decisions/data-access-library.md` | 2h |
| 4.4 | Write frontend framework comparison | `docs/tech-stack-decisions/frontend-framework.md` | 1.5h |
| 4.5 | Create tech selection scoring matrix template | `docs/tech-stack-decisions/evaluation-matrix-template.md` | 1h |
| 4.6 | (Optional) Add Python/Rust sidecar service as polyglot demo | `services/analytics-sidecar/` | 8h |

### Assessment Criteria

- [ ] Technology stack comparison covering minimum 3 alternatives per category
- [ ] Each comparison includes: learning curve, performance, community size, enterprise readiness, ecosystem
- [ ] Scoring matrix with weighted criteria
- [ ] Evidence of cross-platform integration (Go + React + Protobuf already satisfies partially)
- [ ] Documented rationale for each major technology choice

---

## Sub 5 — Document Design Writing

### Level 4 Requirements

- Mampu menuliskan desain dokumen dengan anotasi yang sesuai
- Menulis dokumen desain
- Mereview dokumen desain

### What Currently EXISTS ✅

| Evidence | File Path | Description |
|----------|-----------|-------------|
| Comprehensive README | `README.md` (656 lines) | HLD, LLD, ERD with Mermaid diagrams, sequence diagrams |
| Architecture Document | `docs/architecture.md` (104 lines) | Clean architecture, middleware, resilience patterns |
| Product Requirements Document | `PRD.md` (849 lines) | Full PRD with scenarios, NFRs, acceptance criteria |
| Design Specifications | `docs/superpowers/specs/` (2 files) | Before/after design with rationale |
| Implementation Plans | `docs/superpowers/plans/` (7 files) | Step-by-step implementation guides |
| API Documentation | `docs/api.md` | REST endpoint documentation |
| Code Review Document | `REVIEW.md` (118 lines) | Structured review with severity levels |
| Proto Definitions as Contracts | `proto/*.proto` (6 files) | gRPC service contracts with field annotations |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No formal design document review process | Cannot demonstrate review governance | 🟡 Medium |
| No document versioning/changelog | Cannot track document evolution | 🟡 Medium |
| No review approval stamps/sign-off | No evidence of formal review completion | 🟡 Medium |
| No design document template | No standardized format | 🟢 Low |
| No diagram annotation standards | Inconsistent documentation quality | 🟢 Low |

### ACTION ITEMS

| # | Task | Output File | Effort |
|---|------|-------------|--------|
| 5.1 | Create design document template with required sections | `docs/templates/design-document-template.md` | 1.5h |
| 5.2 | Document the design review process | `docs/design-review-process.md` | 1.5h |
| 5.3 | Add changelog/version history to existing docs | Update `docs/architecture.md`, `README.md` headers | 1h |
| 5.4 | Create design review checklist | `docs/templates/design-review-checklist.md` | 1h |
| 5.5 | Add review sign-off section to design specs | Update `docs/superpowers/specs/*.md` | 0.5h |
| 5.6 | Write diagram annotation guidelines | `docs/templates/diagram-guidelines.md` | 1h |

### Assessment Criteria

- [ ] Design document template with: context, problem, options, decision, consequences, review history
- [ ] Documented review process with roles (author, reviewer, approver)
- [ ] At least 2 design documents with completed review sign-off
- [ ] Version history/changelog in major documents
- [ ] Consistent use of annotations in diagrams (Mermaid notes, comments)

---

## Summary & Priority Matrix

### Overall Sub-Competency Scores

| Sub-Competency | Current | Target | Gap |
|----------------|---------|--------|-----|
| 1. Design Application Architecture | 60% | 90% | 30% |
| 2. Design Component and Module | 75% | 90% | 15% |
| 3. Design Data Structure | 55% | 90% | 35% |
| 4. Select and Use Framework/Library | 50% | 85% | 35% |
| 5. Document Design Writing | 70% | 90% | 20% |

### Top Priority Actions (Quick Wins)

These items provide the highest assessment value for the least effort:

1. **Create ADR directory + 3 ADRs** (Sub 1) — 6h total, closes biggest gap
2. **Write data storage comparison** (Sub 3) — 3h, demonstrates analysis skill
3. **Write tech stack comparison** (Sub 4) — 3h, demonstrates evaluation skill
4. **Create design document template + review process** (Sub 5) — 3h, formalizes existing practice
5. **Add CODEOWNERS** (Sub 2) — 0.5h, quick governance win

### Estimated Total Effort

| Priority | Items | Effort |
|----------|-------|--------|
| 🔴 High (must-have for Level 4) | 12 items | ~28h |
| 🟡 Medium (strengthens case) | 11 items | ~16h |
| 🟢 Low (nice-to-have) | 4 items | ~11h |
| **Total** | **27 items** | **~55h** |

### Recommended Execution Order

1. **Week 1:** ADR directory + 3 core ADRs (items 1.1–1.5)
2. **Week 2:** Data & tech comparisons (items 3.1–3.4, 4.1–4.3)
3. **Week 3:** Event sourcing implementation + ADR (items 2.1–2.2)
4. **Week 4:** Design templates, review process, alternative designs (items 5.1–5.4, 2.3–2.4)
5. **Week 5:** Impact analysis, capacity planning, remaining items (items 1.6, 3.5, 3.6)

---

## References

- `README.md` — System architecture, HLD, LLD, ERD
- `docs/architecture.md` — Clean architecture, middleware, resilience
- `PRD.md` — Product requirements with NFRs
- `REVIEW.md` — Code review findings
- `docs/superpowers/specs/` — Design specifications
- `docs/superpowers/plans/` — Implementation plans
- `docs/LEVEL4_GAP_ANALYSIS.md` — Overall gap analysis summary
