# Post-Mortem: [Incident Title]

> **Status:** Draft | In Review | Final
> **Author:** [Name]
> **Date:** [YYYY-MM-DD]
> **Reviewers:** [Names]

---

## Incident Summary

| Field | Value |
|-------|-------|
| **Incident ID** | INC-YYYY-NNN |
| **Date** | YYYY-MM-DD |
| **Duration** | X hours Y minutes |
| **Severity** | P1 / P2 / P3 / P4 |
| **Services Affected** | [e.g., booking-service, payment-service] |
| **Impact** | [Brief description of user-facing impact] |
| **Detection Method** | [Alert / User report / Monitoring] |
| **Responders** | [Names and roles] |

### Impact Summary

- **Users affected:** [number or percentage]
- **Failed transactions:** [count]
- **Revenue impact:** [estimated amount or "none"]
- **SLO budget consumed:** [percentage of monthly error budget used by this incident]

---

## Timeline

All times in UTC.

| Time | Event |
|------|-------|
| HH:MM | **Detection** — [How the incident was detected: alert fired, user report, etc.] |
| HH:MM | **Acknowledged** — [Who acknowledged, initial assessment] |
| HH:MM | **Investigation** — [First diagnostic steps taken] |
| HH:MM | **Identified** — [Root cause identified] |
| HH:MM | **Mitigation** — [Temporary fix applied, impact reduced] |
| HH:MM | **Resolution** — [Permanent fix deployed, service fully restored] |
| HH:MM | **Monitoring** — [Confirmed stable, stood down] |

### Detection Gap

- Time from incident start to detection: **X minutes**
- Could we have detected sooner? [Yes/No — explain]

---

## Root Cause Analysis

### Summary

[1-2 sentence description of the root cause]

### 5 Whys

1. **Why** did [the symptom] happen?
   → Because [immediate cause]

2. **Why** did [immediate cause] happen?
   → Because [deeper cause]

3. **Why** did [deeper cause] happen?
   → Because [even deeper cause]

4. **Why** did [even deeper cause] happen?
   → Because [systemic cause]

5. **Why** did [systemic cause] happen?
   → Because [root cause / process gap]

### Contributing Factors

- [ ] Code defect
- [ ] Configuration error
- [ ] Infrastructure failure
- [ ] Capacity/scaling issue
- [ ] External dependency failure
- [ ] Deployment/rollout issue
- [ ] Monitoring gap
- [ ] Process/communication gap

### Technical Details

[Detailed technical explanation. Include relevant code paths, configuration values, infrastructure state. Reference specific services in the ParkirPintar stack:]

- **Service(s):** [e.g., booking-service v1.2.3]
- **Infrastructure:** [e.g., Docker container OOM, PostgreSQL connection pool, Redis cluster]
- **Deployment:** [e.g., Coolify deployment at HH:MM, Watchtower auto-update]
- **Networking:** [e.g., Traefik routing, NATS message delivery]

```
[Include relevant log snippets, error messages, or trace IDs]
```

---

## Impact Assessment

### User Impact

| Metric | Value |
|--------|-------|
| Total users affected | |
| Affected user percentage | |
| Failed API requests | |
| Failed bookings | |
| Failed payments | |
| Support tickets generated | |

### Business Impact

| Metric | Value |
|--------|-------|
| Estimated revenue loss | |
| Compensation/credits issued | |
| Reputation impact | [Low/Medium/High] |

### SLO Impact

| SLO | Budget (monthly) | Consumed by incident | Remaining |
|-----|-------------------|---------------------|-----------|
| Availability (99.9%) | 43.2 min downtime | | |
| Latency (p95 < 500ms) | 0.1% of requests | | |
| Error rate (< 0.1%) | 0.1% of requests | | |

---

## Action Items

### Preventive (stop it from happening again)

| # | Action | Owner | Priority | Due Date | Status |
|---|--------|-------|----------|----------|--------|
| 1 | | | P1/P2/P3 | YYYY-MM-DD | Open |
| 2 | | | | | |

### Detective (find it faster next time)

| # | Action | Owner | Priority | Due Date | Status |
|---|--------|-------|----------|----------|--------|
| 1 | | | P1/P2/P3 | YYYY-MM-DD | Open |
| 2 | | | | | |

### Mitigative (reduce impact when it happens)

| # | Action | Owner | Priority | Due Date | Status |
|---|--------|-------|----------|----------|--------|
| 1 | | | P1/P2/P3 | YYYY-MM-DD | Open |
| 2 | | | | | |

---

## Lessons Learned

### What went well

- [e.g., Alert fired within 2 minutes of incident start]
- [e.g., Graceful degradation worked as designed for Redis outage]
- [e.g., Rollback via Coolify was fast and clean]

### What went poorly

- [e.g., Took 20 minutes to identify which service was affected]
- [e.g., No runbook existed for this failure mode]
- [e.g., On-call engineer didn't have access to production logs]

### Where we got lucky

- [e.g., Incident happened during low-traffic hours]
- [e.g., A team member happened to be online despite being off-call]

---

## Appendix

### Relevant Links

| Resource | Link |
|----------|------|
| Grafana dashboard | `http://<host>:3000/d/<dashboard-id>` |
| Prometheus query | `http://<host>:9090/graph?g0.expr=<query>` |
| Tempo trace | `http://<host>:3200/api/traces/<trace-id>` |
| Coolify deployment | `http://<host>:8000/...` |
| Incident Slack/Telegram thread | [link] |
| Related PRs/commits | [links] |
| Alert definition | [link to Grafana alert rule] |

### Relevant Metrics/Graphs

[Embed or link to screenshots of relevant Grafana panels showing the incident window]

### Related Incidents

| Incident | Date | Relation |
|----------|------|----------|
| INC-YYYY-NNN | YYYY-MM-DD | [e.g., Same root cause, similar symptoms] |

---

## Sign-off

| Role | Name | Date |
|------|------|------|
| Author | | |
| Reviewer 1 | | |
| Reviewer 2 | | |
| Engineering Manager | | |

---

*This post-mortem is blameless. We focus on systems and processes, not individuals. The goal is to learn and improve.*
