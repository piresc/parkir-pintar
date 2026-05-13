# ParkirPintar Quality Assurance Plan

## Overview

This document defines the quality assurance processes, metrics, and gates for ParkirPintar. It ensures consistent software quality across all microservices through automated checks, code review processes, and continuous improvement practices.

## Quality Objectives

| Objective                  | Target              | Measurement                          |
|----------------------------|---------------------|--------------------------------------|
| Code coverage              | ≥ 80% overall      | SonarCloud / go test -cover          |
| Critical path coverage     | ≥ 90% (usecase)    | Per-package coverage reports         |
| Defect density             | < 2 bugs/KLOC      | SonarCloud issues / lines of code    |
| Mean Time to Recovery      | < 1 hour (P1/P2)   | Incident tracking                    |
| Build success rate         | > 95%              | GitHub Actions metrics               |
| Test pass rate             | > 99%              | CI pipeline history                  |
| Security vulnerabilities   | 0 Critical/High    | govulncheck + Trivy                  |
| Code duplication           | < 3%               | SonarCloud                           |
| Cyclomatic complexity      | ≤ 15 per function  | golangci-lint (gocyclo)              |

## Quality Gates

### PR Quality Gate (Must Pass to Merge)

| Check                        | Tool                | Threshold                    |
|------------------------------|---------------------|------------------------------|
| Linting                      | golangci-lint       | Zero errors                  |
| Unit tests                   | go test             | 100% pass                    |
| Integration tests            | go test -tags=integration | 100% pass              |
| Code coverage                | go test -cover      | Cannot decrease from main    |
| Security scan (SAST)         | gosec               | Zero HIGH/CRITICAL           |
| Dependency vulnerabilities   | govulncheck         | Zero known vulnerabilities   |
| Secret detection             | gitleaks            | Zero findings                |
| Build                        | go build            | Successful compilation       |
| Proto generation             | buf lint            | Zero errors                  |

### Main Branch Quality Gate (Post-Merge)

| Check                        | Tool                | Threshold                    |
|------------------------------|---------------------|------------------------------|
| E2E tests                    | go test -tags=e2e_docker | 100% pass               |
| Performance (smoke)          | k6                  | p95 < 500ms, errors < 1%    |
| Container scan               | Trivy               | Zero CRITICAL                |
| SonarCloud quality gate      | SonarCloud          | Pass all conditions          |

### Release Quality Gate

| Check                        | Criteria                                    |
|------------------------------|---------------------------------------------|
| All CI checks                | Green on release branch                     |
| E2E tests                    | Full suite passing                          |
| Performance test             | Load test within thresholds                 |
| Security scan                | Full DAST scan (ZAP) clean                  |
| Documentation                | API docs updated, changelog written         |
| Rollback tested              | Verified rollback procedure works           |

## Code Review Process

### Workflow

```
┌──────────┐    ┌───────────┐    ┌──────────────┐    ┌─────────┐
│Developer │───▶│ Open PR   │───▶│ Automated    │───▶│ Human   │
│ commits  │    │           │    │ Checks Pass  │    │ Review  │
└──────────┘    └───────────┘    └──────────────┘    └────┬────┘
                                                          │
                                                    ┌─────▼─────┐
                                                    │  Approve  │
                                                    │  & Merge  │
                                                    └───────────┘
```

### Review Requirements

| Aspect              | Requirement                                          |
|---------------------|------------------------------------------------------|
| Reviewers           | Minimum 1 reviewer (AI assistant or peer)            |
| Automated first     | All CI checks must pass before human review          |
| Review scope        | Logic correctness, error handling, security, tests   |
| Response time       | Within 24 hours for standard PRs                     |
| Blocking issues     | Security flaws, data loss risk, missing tests        |

### Code Review Checklist

- [ ] Business logic is correct and handles edge cases
- [ ] Error handling is comprehensive (no swallowed errors)
- [ ] Tests cover the new/changed code adequately
- [ ] No hardcoded secrets or sensitive data
- [ ] API contracts are backward-compatible (or versioned)
- [ ] Database migrations are reversible
- [ ] Logging is sufficient for debugging (no PII in logs)
- [ ] Performance implications considered
- [ ] Documentation updated if public API changed
- [ ] Follows project coding conventions

## Definition of Done

A feature/task is considered "Done" when all of the following are satisfied:

### Code Complete

- [ ] Implementation matches acceptance criteria
- [ ] Unit tests written and passing (≥ 80% coverage for new code)
- [ ] Integration tests written for data store interactions
- [ ] No TODO/FIXME left without a linked issue
- [ ] Error handling implemented with appropriate error types

### Quality Verified

- [ ] All CI quality gates passing
- [ ] Code reviewed and approved
- [ ] No new SonarCloud issues introduced
- [ ] Security scan clean (gosec, govulncheck)
- [ ] Performance acceptable (no regression in smoke tests)

### Deployment Ready

- [ ] Database migrations tested (up and down)
- [ ] Configuration documented (new env vars, feature flags)
- [ ] Monitoring/alerting updated if new failure modes introduced
- [ ] API documentation updated (protobuf comments, OpenAPI)
- [ ] Changelog entry added

### Validated

- [ ] E2E test covers the happy path
- [ ] Manual smoke test on staging (if applicable)
- [ ] Acceptance criteria verified

## Defect Management

### Severity Levels

| Severity | Definition                                    | Examples                              |
|----------|-----------------------------------------------|---------------------------------------|
| S1 - Critical | System down, data loss, security breach | Auth bypass, DB corruption, full outage |
| S2 - High     | Major feature broken, no workaround     | Check-in fails, payment errors        |
| S3 - Medium   | Feature degraded, workaround exists     | Slow response, incorrect calculation  |
| S4 - Low      | Minor issue, cosmetic                   | Log formatting, typos in messages     |

### Fix SLA

| Severity | Response Time | Fix Deployed | Escalation                    |
|----------|---------------|--------------|-------------------------------|
| S1       | 15 minutes    | 4 hours      | Immediate hotfix, skip review |
| S2       | 2 hours       | 24 hours     | Priority in current sprint    |
| S3       | 1 day         | Next sprint  | Backlog prioritization        |
| S4       | 1 week        | Best effort  | Batch with related work       |

### Defect Lifecycle

```
┌────────┐    ┌──────────┐    ┌───────────┐    ┌────────┐    ┌────────┐
│  New   │───▶│ Triaged  │───▶│ In Progress│───▶│ Fixed  │───▶│ Closed │
└────────┘    └──────────┘    └───────────┘    └────────┘    └────────┘
                   │                                │
                   ▼                                ▼
              ┌──────────┐                    ┌──────────┐
              │ Won't Fix│                    │ Reopened │
              └──────────┘                    └──────────┘
```

### Defect Tracking

- All defects tracked as GitHub Issues with `bug` label
- Severity label applied during triage
- Root cause analysis required for S1/S2 defects
- Post-mortem for S1 incidents (see incident-response/post-mortem-template.md)

## Continuous Improvement

### Metrics Review (Monthly)

| Metric                    | Source          | Action if Degraded                    |
|---------------------------|-----------------|---------------------------------------|
| Test coverage trend       | SonarCloud      | Add tests for uncovered paths         |
| Build failure rate        | GitHub Actions  | Fix flaky tests, improve CI stability |
| Defect escape rate        | Issue tracker   | Improve test coverage for area        |
| Mean time to fix          | Issue tracker   | Process improvement, better tooling   |
| Code duplication          | SonarCloud      | Refactoring sprint                    |
| Dependency freshness      | Dependabot      | Schedule update sprint                |

### Retrospective Process

- Sprint retrospective every 2 weeks
- Focus areas: what went well, what didn't, action items
- Track action item completion rate
- Quarterly quality review with metrics dashboard

### Quality Improvement Backlog

Maintained as GitHub Issues with `quality` label:

- Test infrastructure improvements
- CI/CD pipeline optimizations
- Tooling upgrades
- Documentation gaps
- Technical debt items

## Tools

| Tool            | Purpose                          | Integration Point     |
|-----------------|----------------------------------|-----------------------|
| golangci-lint   | Static analysis, linting         | PR check, pre-commit  |
| gosec           | Security-focused static analysis | PR check              |
| govulncheck     | Dependency vulnerability scan    | PR check              |
| gitleaks        | Secret detection                 | PR check, pre-commit  |
| SonarCloud      | Quality gate, coverage tracking  | Post-merge analysis   |
| Trivy           | Container image scanning         | Image build pipeline  |
| OWASP ZAP       | Dynamic application security     | Weekly scheduled scan |
| k6              | Performance/load testing         | PR (smoke), scheduled |
| go test -race   | Race condition detection         | Every test run        |

## Acceptance Criteria Template

When writing user stories or tasks, use this template for acceptance criteria:

```markdown
### Acceptance Criteria

**Given** [precondition/context]
**When** [action performed]
**Then** [expected outcome]

**Edge Cases:**
- [ ] [Edge case 1 and expected behavior]
- [ ] [Edge case 2 and expected behavior]

**Non-Functional:**
- [ ] Response time < [X]ms at p95
- [ ] No new security findings
- [ ] Test coverage ≥ 80% for new code
```

### Example: Check-In Feature

```markdown
### Acceptance Criteria

**Given** a driver with a valid JWT and an available parking slot
**When** they submit a check-in request with location_id, plate_number, and vehicle_type
**Then** a parking session is created, the slot is marked occupied, and a session ID is returned

**Edge Cases:**
- [ ] Duplicate plate number at same location → reject with 409
- [ ] No available slots → reject with 422 and clear message
- [ ] Invalid location_id → reject with 404
- [ ] Concurrent check-in for same slot → only one succeeds (distributed lock)

**Non-Functional:**
- [ ] Check-in response time < 200ms at p95
- [ ] Idempotent with X-Idempotency-Key header
- [ ] Event published to NATS for occupancy update
```

## Regression Testing Approach

### Strategy

| Trigger                    | Regression Scope                              |
|----------------------------|-----------------------------------------------|
| Bug fix                    | Add regression test for the specific bug      |
| Feature change             | Run full test suite for affected service      |
| Dependency update          | Full integration + E2E suite                  |
| Infrastructure change      | Full E2E + performance smoke                  |
| Security patch             | Security scan + affected service tests        |

### Regression Test Organization

```
tests/
├── unit/                    # Package-level unit tests (co-located)
├── integration/             # Cross-boundary tests
│   ├── postgres/
│   ├── redis/
│   ├── nats/
│   └── grpc/
├── e2e/                     # Full system tests
│   ├── parking_flow_test.go
│   ├── payment_flow_test.go
│   └── operator_flow_test.go
├── performance/             # k6 scripts
│   ├── smoke.js
│   ├── load.js
│   └── stress.js
└── regression/              # Bug-specific regression tests
    └── issue_XXX_test.go
```

### Regression Prevention

- Every bug fix must include a test that reproduces the bug
- Tests tagged with issue number for traceability
- Regression suite runs on every PR (included in integration tests)
- Flaky test quarantine: moved to `//go:build flaky` tag, fixed within 1 sprint

---

*Last updated: 2026-05-13*
*Owner: ParkirPintar Engineering*
*Review cycle: Quarterly*
