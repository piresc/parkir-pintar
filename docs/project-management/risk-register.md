# Risk Register — ParkirPintar

## Document Information

| Field | Value |
|-------|-------|
| Project | ParkirPintar — Smart Parking Reservation System |
| Last Updated | 2026-05-13 |
| Owner | Development Team Lead |
| Review Frequency | Per Sprint (bi-weekly) |

## Risk Scoring Matrix

| Probability \ Impact | High (3) | Medium (2) | Low (1) |
|---------------------|-----------|------------|---------|
| **High (3)** | 9 - Critical | 6 - High | 3 - Medium |
| **Medium (2)** | 6 - High | 4 - Medium | 2 - Low |
| **Low (1)** | 3 - Medium | 2 - Low | 1 - Low |

## Risk Register

| ID | Risk Description | Category | Probability | Impact | Score | Mitigation Strategy | Owner | Status |
|----|-----------------|----------|-------------|--------|-------|--------------------:|-------|--------|
| R-01 | PostgreSQL connection pool exhaustion under peak load causing reservation failures | Technical | M | H | 6 | Configure pgxpool with max connections (25), implement connection timeout (5s), add circuit breaker pattern on DB calls, monitor `db.pool.active_connections` metric | Backend Dev | Active |
| R-02 | Redis single point of failure causing distributed lock and cache unavailability | Technical | M | H | 6 | Deploy Redis with AOF persistence, implement graceful degradation (fallback to DB-based locking), add Redis health check in readiness probe, plan Redis Sentinel for production | DevOps | Active |
| R-03 | NATS message loss during service restarts causing billing/notification inconsistency | Technical | M | M | 4 | Enable JetStream with at-least-once delivery, implement idempotent consumers with deduplication window (2min), add dead letter queue for failed messages, monitor `nats.msgs.dropped` | Backend Dev | Active |
| R-04 | Container image vulnerabilities (CVEs) in base images or dependencies | Technical | H | M | 6 | Run Trivy scan in CI pipeline (fail on CRITICAL/HIGH), use distroless base images, enable Dependabot for Go modules, schedule weekly image rebuilds | DevOps | Mitigated |
| R-05 | gRPC deadline exceeded on inter-service calls during high concurrency | Technical | M | M | 4 | Set appropriate deadlines per RPC (search: 2s, reserve: 5s, billing: 10s), implement retry with exponential backoff (max 3 attempts), add circuit breaker with half-open state | Backend Dev | Active |
| R-06 | Watchtower auto-pulling broken/untested image to production | Operational | M | H | 6 | Pin Watchtower to poll only `latest` tag from GHCR, gate image promotion behind CI green status, implement health-check based rollback (3 consecutive failures → revert), add Slack notification on pull | DevOps | Active |
| R-07 | Disk space exhaustion on VPS from container logs and Prometheus TSDB | Operational | M | M | 4 | Configure Docker log rotation (max-size: 10m, max-file: 3), set Prometheus retention to 15 days, add disk usage alert at 80% threshold, implement log shipping to Loki with TTL | DevOps | Active |
| R-08 | TLS certificate expiry causing service unavailability | Operational | L | H | 3 | Use Caddy with automatic ACME renewal (Let's Encrypt), add certificate expiry monitoring (alert at 14 days), document manual renewal procedure as runbook | DevOps | Mitigated |
| R-09 | Monitoring stack (Prometheus/Grafana/Loki) failure causing blind spots during incidents | Operational | L | M | 2 | Implement meta-monitoring (Prometheus self-scrape alert), add external uptime check (UptimeRobot) as independent signal, configure PagerDuty integration for critical alerts | DevOps | Active |
| R-10 | JWT token theft via XSS or man-in-the-middle enabling unauthorized reservations | Security | L | H | 3 | Use short-lived access tokens (15min) with refresh token rotation, set HttpOnly/Secure/SameSite cookie flags, implement token binding to client fingerprint, add anomaly detection on token usage patterns | Backend Dev | Active |
| R-11 | SQL injection via dynamic query construction in search/filter endpoints | Security | L | H | 3 | Use parameterized queries exclusively (sqlc generated code), implement input validation with struct tags, run DAST (ZAP) in CI pipeline, conduct code review checklist for raw SQL | Backend Dev | Mitigated |
| R-12 | DDoS attack on public API endpoints exhausting server resources | Security | M | H | 6 | Implement rate limiting per IP (100 req/min) and per user (300 req/min), deploy behind Cloudflare with DDoS protection, add request size limits (1MB), implement adaptive throttling based on server load | DevOps | Active |
| R-13 | Payment gateway (Midtrans) downtime causing failed billing transactions | Business | M | H | 6 | Implement async payment processing via NATS queue, add payment retry with exponential backoff (max 5 attempts over 1hr), provide graceful degradation (reserve now, pay later within 15min), monitor gateway health endpoint | Backend Dev | Active |
| R-14 | Data loss from failed or partially-applied database migrations | Business | L | H | 3 | Use golang-migrate with transactional migrations, implement pre-migration backup (pg_dump), test migrations against production-like data in staging, add rollback migration for every up migration, version-lock migration tool | Backend Dev | Mitigated |

## Risk Response Actions

### Immediate Actions (Score ≥ 6)
- R-01: Load test connection pool under 500 concurrent users, tune pool settings
- R-02: Document Redis failover procedure, test graceful degradation path
- R-04: Review latest Trivy scan results, update base images
- R-06: Add pre-pull health validation script to Watchtower workflow
- R-12: Validate rate limiter configuration under simulated attack
- R-13: Implement and test payment retry queue

### Monitoring Actions (Score 4)
- R-03: Verify JetStream consumer acknowledgment in integration tests
- R-05: Add gRPC deadline metrics to Grafana dashboard
- R-07: Set up disk usage alerting in Prometheus

### Accepted Risks (Score ≤ 3)
- R-08: Caddy auto-renewal is reliable; manual procedure documented
- R-09: External uptime check provides minimum viable monitoring
- R-10: Short token lifetime limits blast radius
- R-11: sqlc-generated queries eliminate dynamic SQL construction
- R-14: Transactional migrations with tested rollbacks provide adequate safety

## Review History

| Date | Reviewer | Changes |
|------|----------|---------|
| 2026-03-17 | Dev Team | Initial risk register created |
| 2026-04-01 | Dev Team | Added R-04 after Trivy integration, marked as mitigated |
| 2026-04-14 | Dev Team | Added R-11, marked mitigated after DAST pipeline added |
| 2026-05-13 | Dev Team | Updated scores after load testing results, added R-13 |
