# Project Charter

## ParkirPintar — Smart Parking Reservation System

| Field | Value |
|-------|-------|
| Document ID | PP-PM-CHARTER-001 |
| Version | 1.0 |
| Date | 2025-01-15 |
| Status | Approved |
| Sponsor | VP of Engineering |
| Project Manager | Tech Lead, ParkirPintar |

---

## 1. Project Purpose and Justification

ParkirPintar addresses the critical inefficiency in urban parking management across Indonesian cities. Current systems suffer from:

- **Double-booking conflicts** — legacy systems allow multiple drivers to reserve the same spot simultaneously, causing 12% reservation failure rate
- **Poor real-time visibility** — drivers waste an average of 15 minutes searching for available spots
- **Revenue leakage** — manual billing processes result in ~8% uncollected fees
- **No-show waste** — 20% of reserved spots go unused without penalty enforcement

The project delivers a microservices-based smart parking reservation platform that eliminates double-booking through distributed locking, provides real-time spot availability, automates billing and penalty enforcement, and scales horizontally to support 10,000+ concurrent users.

**Business Case:** Projected 30% increase in parking utilization, 95% reduction in booking conflicts, and 15% revenue uplift through automated penalty collection.

---

## 2. Measurable Objectives

| # | Objective | Target | Measurement Method |
|---|-----------|--------|-------------------|
| O1 | Eliminate double-booking | 0% conflict rate | Reservation conflict counter metric |
| O2 | Low-latency responses | p95 < 500ms, p99 < 1s | Prometheus histogram percentiles |
| O3 | High availability | 99.5% uptime (monthly) | Uptime monitoring (excluding planned maintenance) |
| O4 | Throughput capacity | 500 reservations/sec sustained | k6 load test results |
| O5 | Payment success rate | > 99% for valid transactions | Payment gateway success ratio |
| O6 | Search freshness | < 3s stale window | Event lag metric (NATS → read model) |

---

## 3. High-Level Requirements

### 3.1 Functional Requirements

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-01 | Real-time parking spot availability search | Must Have |
| FR-02 | Reservation creation with conflict-free guarantee | Must Have |
| FR-03 | Driver check-in/check-out with presence detection | Must Have |
| FR-04 | Automated billing calculation (hourly/daily rates) | Must Have |
| FR-05 | Payment processing via external gateway (Midtrans) | Must Have |
| FR-06 | Penalty enforcement for no-shows and overstays | Should Have |
| FR-07 | Reservation cancellation with refund policy | Must Have |
| FR-08 | Admin dashboard for lot management | Should Have |

### 3.2 Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-01 | Horizontal scalability | Each service scales independently |
| NFR-02 | Data consistency | Strong consistency for reservations, eventual for search |
| NFR-03 | Fault tolerance | Graceful degradation on component failure |
| NFR-04 | Security | JWT auth, encrypted PII, PCI-DSS awareness for payments |
| NFR-05 | Observability | Structured logging, distributed tracing, metrics |

### 3.3 Architecture Requirements

- 6 microservices: Reservation, Billing, Payment, Presence, Search, Gateway
- Event-driven communication via NATS JetStream
- PostgreSQL per-service databases (schema isolation)
- Redis for distributed locking and caching
- Kubernetes deployment with Helm charts
- gRPC inter-service communication, REST external API

---

## 4. High-Level Risks

The following are top-5 risks. Full analysis available in the [Risk Register](./risk-register.md).

| Risk ID | Description | Impact | Probability | Mitigation |
|---------|-------------|--------|-------------|------------|
| R-01 | Redis cluster failure causes reservation deadlocks | High | Low | Lock TTL expiry + fallback to DB advisory locks |
| R-02 | NATS message loss causes stale search results | Medium | Medium | JetStream persistence + consumer ack + replay |
| R-03 | Payment gateway downtime blocks billing | High | Low | Circuit breaker + retry queue + manual reconciliation |
| R-04 | Database migration breaks backward compatibility | High | Medium | Blue-green migration strategy + feature flags |
| R-05 | Load spike exceeds provisioned capacity | Medium | Medium | HPA auto-scaling + rate limiting + load shedding |

---

## 5. Summary Milestone Schedule

Full roadmap available in the [Project Roadmap](./roadmap.md).

| Milestone | Target Date | Deliverable |
|-----------|-------------|-------------|
| M1: Foundation | Week 4 | Core infrastructure, CI/CD, service scaffolding |
| M2: Reservation MVP | Week 8 | Reservation + Search services, distributed lock |
| M3: Billing & Payment | Week 12 | Billing + Payment services, Midtrans integration |
| M4: Presence & Penalties | Week 14 | Presence service, penalty automation |
| M5: Integration Testing | Week 16 | End-to-end flows, load testing, chaos testing |
| M6: Production Release | Week 18 | Production deployment, monitoring, runbooks |
| M7: Stabilization | Week 20 | Bug fixes, performance tuning, documentation |

---

## 6. Budget Summary

Full resource allocation available in the [Resource Allocation Plan](./resource-allocation.md).

| Category | Allocation | Notes |
|----------|-----------|-------|
| Engineering (4 engineers × 5 months) | 20 person-months | Backend-focused team |
| Infrastructure (cloud) | $2,500/month | GKE cluster, Cloud SQL, Memorystore |
| Third-party services | $500/month | Midtrans, monitoring tools |
| Load testing infrastructure | $300 (one-time) | k6 Cloud for distributed load tests |
| Contingency | 15% of total | Risk buffer |

---

## 7. Stakeholders

Full analysis available in the [Stakeholder Analysis](./stakeholder-analysis.md).

| Stakeholder | Role | Interest | Influence |
|-------------|------|----------|-----------|
| VP of Engineering | Sponsor | Project success, ROI | High |
| Product Manager | Requirements owner | Feature completeness, UX | High |
| Tech Lead | Project Manager | Architecture, quality | High |
| Backend Engineers (3) | Development team | Implementation, learning | Medium |
| DevOps Engineer | Infrastructure | Deployment, reliability | Medium |
| Parking Lot Operators | End users (admin) | Operational efficiency | Medium |
| Drivers | End users (consumer) | Booking experience | Low (indirect) |
| Finance Team | Payment oversight | Revenue accuracy | Low |

---

## 8. Project Manager Authority

The Project Manager (Tech Lead) has authority to:

- **Staffing:** Request team member allocation changes through engineering management
- **Budget:** Approve infrastructure spend within allocated monthly budget; escalate overages > 10%
- **Technical decisions:** Final authority on architecture and technology choices within approved constraints
- **Schedule:** Adjust internal sprint priorities; milestone date changes require sponsor approval
- **Scope:** Accept/reject change requests for items ≤ 3 story points; larger changes require Change Control Board

### Approval Chain

| Decision Type | Authority | Escalation |
|---------------|-----------|------------|
| Technical architecture | Tech Lead | VP of Engineering |
| Scope change (small) | Tech Lead | Product Manager |
| Scope change (large) | Change Control Board | VP of Engineering |
| Budget increase | VP of Engineering | CTO |
| Schedule extension | VP of Engineering | CTO |

---

## 9. Success Criteria and Acceptance Criteria

### 9.1 Success Criteria (Project Level)

The project is considered successful when:

1. All measurable objectives (Section 2) are met in production for 30 consecutive days
2. Zero critical (P0) bugs in production after stabilization period
3. System handles 2x projected peak load without degradation
4. All services achieve ≥ 80% unit test coverage
5. Operational runbooks are complete and validated through incident simulation

### 9.2 Acceptance Criteria (Delivery Level)

| Deliverable | Acceptance Criteria |
|-------------|-------------------|
| Reservation Service | Creates reservation in < 200ms; zero double-bookings under concurrent load |
| Search Service | Returns results in < 100ms; read model lag < 3s |
| Billing Service | Correct rate calculation for all billing scenarios; idempotent charge creation |
| Payment Service | Successful Midtrans integration; handles all webhook events; reconciliation report |
| Presence Service | Accurate check-in/out tracking; triggers penalty after grace period |
| Gateway | Rate limiting, JWT validation, request routing, < 50ms overhead |

---

## 10. Constraints and Assumptions

### 10.1 Constraints

| # | Constraint | Impact |
|---|-----------|--------|
| C1 | Team size fixed at 4 engineers | Limits parallelization; sequential service development |
| C2 | Must use Go as primary language | Team expertise; no polyglot services |
| C3 | PostgreSQL as primary datastore | Organizational standard; no NoSQL for transactional data |
| C4 | Midtrans as payment gateway | Contractual obligation; no alternative gateway |
| C5 | 5-month delivery timeline | Aggressive; requires scope discipline |
| C6 | Must run on GKE (Kubernetes) | Infrastructure team mandate |

### 10.2 Assumptions

| # | Assumption | Risk if Invalid |
|---|-----------|-----------------|
| A1 | Parking lot operators provide accurate spot inventory | Incorrect availability data |
| A2 | Midtrans API remains stable (v2) | Integration rework required |
| A3 | Peak concurrent users ≤ 10,000 | Capacity planning insufficient |
| A4 | Average reservation duration is 1-8 hours | Billing model may need adjustment |
| A5 | Network latency between services < 5ms (same cluster) | Timeout tuning needed |
| A6 | Team members available full-time (no split allocation) | Schedule slippage |
| A7 | Redis cluster provides < 1ms lock acquisition | Lock timeout values need increase |

---

## Approvals

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Project Sponsor | VP of Engineering | _________ | _________ |
| Project Manager | Tech Lead | _________ | _________ |
| Product Manager | Product Manager | _________ | _________ |

---

*This charter authorizes the project to proceed. Changes to scope, schedule, or budget require formal change request approval as defined in Section 8.*
