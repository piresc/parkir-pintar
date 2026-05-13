# Stakeholder Analysis — ParkirPintar

## Document Information

| Field | Value |
|-------|-------|
| Project | ParkirPintar — Smart Parking Reservation System |
| Last Updated | 2026-05-13 |
| Analysis Method | Power/Interest Grid |

## Stakeholder Matrix

| Stakeholder | Role | Interest Level | Influence Level | Quadrant | Communication Strategy | Key Concerns |
|-------------|------|:--------------:|:---------------:|----------|----------------------|--------------|
| End Users (Drivers) | Primary system consumers who search, reserve, and pay for parking spots | H | L | Keep Informed | In-app notifications, UX feedback surveys, public changelog, response time SLOs visible in status page | Reservation reliability, fast search response (<500ms), transparent billing, mobile-friendly UX, real-time spot availability |
| Parking Lot Operators | Business partners who register lots, manage capacity, and receive revenue | H | M | Keep Satisfied | Monthly usage reports, operator dashboard, direct email for incidents, onboarding documentation | Revenue accuracy, real-time occupancy data, easy lot management, integration simplicity, uptime guarantees |
| Development Team | Engineers building and maintaining the microservices (Go, gRPC, NATS) | H | H | Manage Closely | Daily standups, sprint planning/review, Slack channel, ADR reviews, pair programming sessions | Code quality, clear architecture decisions, manageable tech debt, good DX (developer experience), test coverage, documentation |
| DevOps/SRE | Infrastructure and reliability engineers managing deployment pipeline and observability | H | H | Manage Closely | Incident post-mortems, SLO reviews, infrastructure-as-code PRs, on-call rotation schedule, runbook updates | System reliability (99.9% uptime), deployment safety, observability coverage, alert fatigue reduction, infrastructure cost |
| Product Owner | Defines product vision, prioritizes backlog, accepts deliverables | H | H | Manage Closely | Sprint reviews, backlog refinement sessions, weekly 1:1, roadmap updates, milestone demos | Feature delivery velocity, alignment with business goals, competitive differentiation, user adoption metrics, assessment readiness |
| Payment Gateway Provider (Midtrans) | Third-party payment processing service for billing transactions | M | M | Monitor | API changelog monitoring, integration health dashboard, support ticket escalation path, quarterly review | API stability, transaction volume compliance, PCI-DSS adherence, webhook reliability, settlement accuracy |
| University Assessors | Academic evaluators assessing software engineering competency (Level 4) | H | H | Manage Closely | Comprehensive documentation (ADRs, SLOs, test reports), demo sessions, architecture diagrams, project management artifacts | Evidence of engineering rigor, proper methodology application, quality metrics, security practices, professional documentation, traceability |

## Power/Interest Grid

```
                    HIGH INFLUENCE
                         │
    ┌────────────────────┼────────────────────┐
    │                    │                    │
    │   Keep Satisfied   │  Manage Closely    │
    │                    │                    │
    │  • Parking Lot     │  • Dev Team        │
    │    Operators       │  • DevOps/SRE      │
    │                    │  • Product Owner   │
    │                    │  • Uni Assessors   │
H   │                    │                    │
I   ├────────────────────┼────────────────────┤
G   │                    │                    │
H   │      Monitor       │   Keep Informed    │
    │                    │                    │
I   │  • Payment Gateway │  • End Users       │
N   │    Provider        │    (Drivers)       │
T   │                    │                    │
E   │                    │                    │
R   └────────────────────┼────────────────────┘
E                        │
S                   LOW INFLUENCE
T
```

## Communication Plan

### High-Touch Stakeholders (Manage Closely)

| Stakeholder | Channel | Frequency | Format | Responsible |
|-------------|---------|-----------|--------|-------------|
| Development Team | Slack + GitHub | Daily | Standup notes, PR reviews | Scrum Master |
| Development Team | Video call | Bi-weekly | Sprint planning & review | Product Owner |
| DevOps/SRE | Grafana + PagerDuty | Continuous | Dashboards, alerts | SRE Lead |
| DevOps/SRE | Document | Per incident | Post-mortem report | Incident Commander |
| Product Owner | 1:1 meeting | Weekly | Progress update, blockers | Tech Lead |
| University Assessors | Documentation | Per milestone | Technical reports, demos | Product Owner |

### Medium-Touch Stakeholders (Keep Satisfied / Keep Informed)

| Stakeholder | Channel | Frequency | Format | Responsible |
|-------------|---------|-----------|--------|-------------|
| End Users | In-app + Status page | As needed | Notifications, changelogs | Product Owner |
| Parking Lot Operators | Email + Dashboard | Monthly | Usage reports, updates | Product Owner |
| Payment Gateway | API monitoring | Continuous | Health checks, alerts | DevOps |

## Stakeholder Engagement Strategy

### End Users (Drivers)
- **Engagement Goal:** High satisfaction, low churn
- **Success Metrics:** NPS > 40, reservation completion rate > 95%, search P99 < 500ms
- **Risk if Disengaged:** Low adoption, negative reviews, project failure

### Parking Lot Operators
- **Engagement Goal:** Reliable partnership, accurate revenue
- **Success Metrics:** Billing accuracy 99.99%, operator dashboard uptime 99.9%
- **Risk if Disengaged:** Supply-side shortage, revenue disputes

### Development Team
- **Engagement Goal:** High productivity, low burnout, quality output
- **Success Metrics:** Sprint velocity stable ±10%, test coverage > 80%, zero critical bugs in prod
- **Risk if Disengaged:** Technical debt accumulation, delivery delays

### University Assessors
- **Engagement Goal:** Clear evidence of Level 4 competency
- **Success Metrics:** All assessment criteria documented with traceability
- **Risk if Disengaged:** Failed assessment despite good technical work

## RACI Matrix (Key Decisions)

| Decision | Dev Team | DevOps | Product Owner | Assessors |
|----------|:--------:|:------:|:-------------:|:---------:|
| Architecture changes | R | C | A | I |
| Security policy | C | R | A | I |
| Feature prioritization | C | I | R/A | I |
| Deployment strategy | C | R | A | I |
| SLO definitions | R | R | A | I |
| Technology selection | R | C | A | I |
| Documentation standards | R | C | A | C |

*R = Responsible, A = Accountable, C = Consulted, I = Informed*
