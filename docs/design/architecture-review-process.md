# Architecture Review Process

**Project:** ParkirPintar — Smart Parking Backend System  
**Version:** 1.0  
**Date:** 2026-05-13  
**Status:** Approved

---

## 1. When Reviews Are Required

### 1.1 Mandatory Review Triggers

| Trigger | Examples in ParkirPintar | Review Type |
|---------|------------------------|-------------|
| **New service addition** | Adding an Analytics service, adding a Loyalty service | Full architecture review |
| **Cross-service API changes** | Modifying `reservation.proto`, adding new NATS subjects | Integration review |
| **Infrastructure changes** | Adding Redis Sentinel, migrating to Kubernetes, adding PgBouncer | Infrastructure review |
| **Database schema changes (breaking)** | Removing columns, changing data types, table splits | Data review |
| **New external integration** | Integrating Midtrans payment gateway, FCM push notifications | Integration review |
| **Security-sensitive changes** | Auth flow modifications, new API key schemes, encryption changes | Security review |
| **Performance-critical changes** | Changing distributed lock strategy, modifying cache TTLs | Performance review |
| **Communication pattern changes** | Converting sync gRPC call to async NATS event | Full architecture review |

### 1.2 Review Exemptions (No Review Required)

- Bug fixes within a single service that don't change APIs
- Adding unit/integration tests
- Documentation updates
- Dependency version bumps (patch/minor)
- Configuration value changes (non-structural)
- Adding new fields to existing proto messages (backward compatible)

### 1.3 Decision Flowchart

```
Change proposed
    │
    ├── Affects multiple services? ──── Yes ──► Architecture Review
    │
    ├── Changes proto/API contract? ──── Yes ──► Integration Review
    │
    ├── Changes infrastructure? ──────── Yes ──► Infrastructure Review
    │
    ├── Affects auth/security? ────────── Yes ──► Security Review
    │
    ├── Single-service internal change? ── Yes ──► Code Review (standard PR)
    │
    └── Documentation/tests only? ──────── Yes ──► No review needed
```

---

## 2. Review Board Composition

### 2.1 Standing Members

| Role | Responsibility | Required For |
|------|---------------|--------------|
| **Technical Lead** | Final decision authority, architecture consistency | All reviews |
| **Service Owner (affected)** | Domain expertise, impact assessment | All reviews |
| **DevOps/SRE** | Operational feasibility, deployment impact | Infrastructure, performance |
| **Security Engineer** | Security implications, threat assessment | Security reviews |

### 2.2 Optional/Invited Members

| Role | When Invited |
|------|-------------|
| Product Owner | When change affects user-facing behavior or timelines |
| Database Administrator | Schema changes, partitioning, replication changes |
| QA Lead | When testing strategy needs revision |
| Finance/Billing SME | When billing logic or payment flow changes |

### 2.3 Quorum Requirements

| Review Type | Minimum Attendees | Required Roles |
|-------------|-------------------|----------------|
| Full architecture | 3 | Tech Lead + 2 Service Owners |
| Integration | 2 | Tech Lead + affected Service Owner |
| Infrastructure | 2 | Tech Lead + DevOps |
| Security | 2 | Tech Lead + Security Engineer |
| Performance | 2 | Tech Lead + Service Owner |

---

## 3. Review Checklist

### 3.1 Scalability

- [ ] Does the change maintain stateless service design?
- [ ] Can the affected services still scale horizontally?
- [ ] Does the change introduce new bottlenecks (single points of contention)?
- [ ] Is the database impact acceptable? (connection count, query load, storage growth)
- [ ] Are there new hot paths that need caching?
- [ ] Does the change respect current capacity targets (1000 concurrent users, 100 req/s)?

**ParkirPintar-specific:**
- [ ] Does the change affect distributed lock contention on parking spots?
- [ ] Does the change increase PostgreSQL connection pressure?
- [ ] Is Redis memory impact acceptable (< 256 MB total)?

### 3.2 Security

- [ ] Are new endpoints properly authenticated (JWT at Gateway)?
- [ ] Is authorization enforced (driver can only access own resources)?
- [ ] Are inputs validated (proto schema + handler validation)?
- [ ] Are secrets managed via environment variables (never hardcoded)?
- [ ] Is the change resistant to injection attacks (parameterized queries)?
- [ ] Does the change maintain TLS for external communication?
- [ ] Is rate limiting applied to new public endpoints?

**ParkirPintar-specific:**
- [ ] Can a driver access another driver's reservation/payment data?
- [ ] Are idempotency keys validated to prevent replay attacks?
- [ ] Does the Presence service location data have appropriate access controls?

### 3.3 Observability

- [ ] Are new code paths instrumented with OpenTelemetry spans?
- [ ] Are new metrics exposed (counters, histograms)?
- [ ] Is structured logging added for new operations?
- [ ] Are trace context propagated across service boundaries?
- [ ] Do new error paths have appropriate log levels?
- [ ] Are alerting rules needed for new failure modes?

**ParkirPintar-specific:**
- [ ] Are new gRPC methods traced via interceptors?
- [ ] Are new NATS subjects included in consumer lag monitoring?
- [ ] Is the Grafana dashboard updated for new metrics?

### 3.4 Cost

- [ ] What is the infrastructure cost delta? (compute, storage, network)
- [ ] Does the change require new infrastructure components?
- [ ] Is the cost proportional to the business value delivered?
- [ ] Are there cheaper alternatives that meet the same requirements?
- [ ] Does the change affect external API call volume (payment gateway costs)?

**ParkirPintar-specific:**
- [ ] Does the change increase payment gateway API calls (cost per transaction)?
- [ ] Does the change increase NATS message volume significantly?
- [ ] Is additional Redis memory justified?

### 3.5 Maintainability

- [ ] Does the change follow existing patterns (clean architecture, repository pattern)?
- [ ] Is the change backward compatible with existing clients?
- [ ] Is the migration path clear and reversible?
- [ ] Is the change documented (ADR if architectural, inline comments if complex)?
- [ ] Does the change increase or decrease system complexity?
- [ ] Are there new dependencies? Are they well-maintained and pinned?

**ParkirPintar-specific:**
- [ ] Does the change follow the `internal/{service}/usecase/` → `repository/` → `handler/` pattern?
- [ ] Are proto changes backward compatible (no removed/renumbered fields)?
- [ ] Is the NATS event schema documented?

### 3.6 Reliability

- [ ] What happens when the new component fails?
- [ ] Is there a circuit breaker for new external calls?
- [ ] Is the change idempotent (safe to retry)?
- [ ] Does the change have a rollback plan?
- [ ] Are health checks updated for new dependencies?
- [ ] Does the change maintain the 99.9% availability target?

**ParkirPintar-specific:**
- [ ] Does the change maintain graceful degradation (Notification/Presence failures don't cascade)?
- [ ] Are distributed locks properly released on failure (TTL expiry)?
- [ ] Is the reservation expiry worker still functioning correctly?

---

## 4. Review Meeting Format

### 4.1 Standard Review (30 minutes)

| Time | Activity | Owner |
|------|----------|-------|
| 0:00 - 10:00 | **Presentation** — Proposer presents the change: what, why, how, impact | Proposer |
| 10:00 - 25:00 | **Discussion** — Board asks questions, raises concerns, suggests alternatives | All |
| 25:00 - 30:00 | **Decision** — Tech Lead summarizes and announces decision | Tech Lead |

### 4.2 Presentation Template (10 minutes)

The proposer should cover:

1. **Problem statement** (1 min) — What problem does this solve?
2. **Proposed solution** (3 min) — Architecture diagram, affected services, data flow
3. **Alternatives considered** (2 min) — What else was evaluated and why rejected?
4. **Impact assessment** (2 min) — Services affected, resource delta, risk level
5. **Migration plan** (2 min) — How to deploy, rollback strategy, timeline

### 4.3 Discussion Guidelines

- Focus on architectural concerns, not implementation details
- Raise blocking concerns early (security, data loss, breaking changes)
- Suggest alternatives rather than just rejecting proposals
- Reference existing ADRs and patterns for consistency
- Time-box tangential discussions ("take offline")

### 4.4 Extended Review (60 minutes)

For complex changes (new service, infrastructure migration), extend to 60 minutes:

| Time | Activity |
|------|----------|
| 0:00 - 15:00 | Presentation (extended) |
| 15:00 - 45:00 | Discussion (deep dive) |
| 45:00 - 55:00 | Action items and conditions |
| 55:00 - 60:00 | Decision |

---

## 5. Decision Outcomes

### 5.1 Outcome Types

| Outcome | Meaning | Next Steps |
|---------|---------|------------|
| **Approved** | Change is accepted as proposed | Proceed with implementation |
| **Approved with Conditions** | Change is accepted pending specific modifications | Address conditions, then proceed (no re-review needed) |
| **Rejected** | Change is not acceptable in current form | Proposer may revise and re-submit |
| **Deferred** | More information needed or timing is wrong | Gather info, re-schedule review |

### 5.2 Condition Examples

| Condition Type | Example |
|----------------|---------|
| Add observability | "Add OTel spans to the new gRPC handler before merging" |
| Add tests | "Add integration test for the Reservation → new Billing RPC path" |
| Add documentation | "Document the new NATS subject in the integration matrix" |
| Modify approach | "Use async NATS instead of sync gRPC for notification delivery" |
| Add safety mechanism | "Add circuit breaker to the new external API call" |
| Performance validation | "Run load test proving < 200ms P95 with the change" |

### 5.3 Escalation Path

If consensus cannot be reached:

1. Tech Lead makes final decision (documented with dissenting opinions)
2. If Tech Lead is the proposer, escalate to Engineering Manager
3. Dissenting opinions recorded in the ADR for future reference

---

## 6. Documentation Requirements Post-Review

### 6.1 Required Artifacts

| Outcome | Required Documentation |
|---------|----------------------|
| Approved | ADR (if architectural), updated architecture docs |
| Approved with Conditions | ADR + condition tracking (checklist in PR) |
| Rejected | Brief rejection note in review log (reason + alternatives suggested) |
| Deferred | Action items with owners and deadlines |

### 6.2 ADR Template (for approved architectural changes)

```markdown
# ADR-XXXX: [Title]

## Status
Accepted

## Context
[Why this decision was needed]

## Decision
[What was decided]

## Consequences
### Positive
- [benefit 1]
- [benefit 2]

### Negative
- [trade-off 1]
- [trade-off 2]

### Neutral
- [observation]

## Alternatives Considered
- [Alternative A]: rejected because [reason]
- [Alternative B]: rejected because [reason]

## Review
- Date: YYYY-MM-DD
- Attendees: [names]
- Decision: Approved / Approved with conditions
- Conditions: [if any]
```

### 6.3 Review Log Entry

Each review is logged in `docs/architecture/review-log.md`:

```markdown
| Date | CR/PR | Title | Proposer | Outcome | ADR |
|------|-------|-------|----------|---------|-----|
| 2026-05-07 | CR-007 | Two-Phase Payment Flow | [name] | Approved | — |
| 2026-05-07 | CR-008 | Database-per-Service Separation | [name] | Deferred | — |
```

---

## 7. Link to ADR Process

### 7.1 Existing ADRs

| ADR | Title | Status | Review Date |
|-----|-------|--------|-------------|
| [ADR-0001](../adr/0001-microservices-architecture.md) | Microservices Architecture | Accepted | 2025-01-01 |
| [ADR-0002](../adr/0002-grpc-internal-communication.md) | gRPC Internal Communication | Accepted | 2025-01-01 |
| [ADR-0003](../adr/0003-nats-event-driven.md) | NATS Event-Driven Architecture | Accepted | 2025-01-01 |
| [ADR-0004](../adr/0004-distributed-locking-redis.md) | Distributed Locking with Redis | Accepted | 2025-01-01 |
| [ADR-0005](../adr/0005-opentelemetry-observability.md) | OpenTelemetry Observability | Accepted | 2025-01-01 |
| [ADR-0006](../adr/0006-terraform-iac.md) | Terraform Infrastructure as Code | Accepted | 2025-01-01 |

### 7.2 ADR Lifecycle

```
Proposed → Under Review → Accepted / Rejected / Superseded
                              │
                              ▼
                    Implemented (linked to code)
                              │
                              ▼
                    Superseded (by newer ADR, if applicable)
```

### 7.3 When to Write an ADR

- Any decision that would be hard to reverse
- Any decision that affects multiple services
- Any decision where alternatives were seriously considered
- Any decision that future team members would question "why?"

**ParkirPintar examples requiring ADRs:**
- Choosing NATS over Kafka (ADR-0003 ✅)
- Choosing schema-per-service over DB-per-service
- Choosing two-phase payment over single-phase
- Choosing Redis SETNX over PostgreSQL advisory locks (ADR-0004 ✅)

---

## 8. Review Process Metrics

### 8.1 Process Health Indicators

| Metric | Target | Measurement |
|--------|--------|-------------|
| Review turnaround time | < 3 business days | Time from request to decision |
| Review meeting duration | ≤ 30 min (standard) | Calendar time |
| Rejection rate | < 20% | Indicates good pre-alignment |
| Condition completion rate | 100% | All conditions met before merge |
| ADR coverage | 100% of architectural decisions | No undocumented decisions |

### 8.2 Anti-Patterns to Avoid

| Anti-Pattern | Symptom | Remedy |
|--------------|---------|--------|
| Rubber-stamping | All reviews approved in < 5 min | Ensure checklist is actually reviewed |
| Gatekeeping | Reviews take > 1 week, high rejection | Pre-alignment conversations before formal review |
| Scope creep | Review discusses unrelated improvements | Time-box, "take offline" for tangents |
| Missing reviews | Architectural changes merged without review | CI check: PRs touching proto/infra require review label |
| Stale ADRs | ADRs don't reflect current architecture | Quarterly ADR audit |

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-05-13 | Engineering Team | Initial architecture review process |
