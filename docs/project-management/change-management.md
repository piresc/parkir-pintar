# Change Management Process — ParkirPintar

## Document Information

| Field | Value |
|-------|-------|
| Project | ParkirPintar — Smart Parking Reservation System |
| Last Updated | 2026-05-13 |
| Approved By | Product Owner / Tech Lead |
| Review Frequency | Quarterly or after major incidents |

---

## 1. Change Categories

| Category | Description | Approval Required | Lead Time | Examples |
|----------|-------------|:-----------------:|:---------:|---------|
| **Standard** | Pre-approved, low-risk, routine changes | None (pre-authorized) | Immediate | Dependency patch updates, log level changes, dashboard edits, documentation updates |
| **Normal** | Planned changes requiring review and approval | Tech Lead + 1 reviewer | ≥ 2 business days | New feature deployment, schema migration, infrastructure config change, new service endpoint |
| **Emergency** | Urgent fixes for production incidents | Tech Lead (verbal OK, post-hoc documentation) | Immediate | Security patch for active exploit, hotfix for data corruption, rollback of broken deploy |

---

## 2. Change Request Template

```markdown
## Change Request

### CR-[YYYY]-[NNN]

| Field | Value |
|-------|-------|
| Requester | [Name] |
| Date Submitted | [YYYY-MM-DD] |
| Category | Standard / Normal / Emergency |
| Priority | Critical / High / Medium / Low |
| Target Date | [YYYY-MM-DD] |
| Services Affected | [reservation / search / billing / infrastructure / all] |

### Description
[What is being changed and why]

### Business Justification
[Why this change is needed now — link to issue/story]

### Technical Details
- **Components Modified:** [list files, services, configs]
- **Database Changes:** Yes / No — [describe migration if yes]
- **API Changes:** Yes / No — [breaking / non-breaking]
- **Infrastructure Changes:** Yes / No — [describe]

### Impact Assessment
- **User Impact:** [None / Degraded / Outage during deploy]
- **Blast Radius:** [Single service / Multiple services / Full system]
- **Reversibility:** [Fully reversible / Partially reversible / Irreversible]
- **Data Impact:** [No data change / Data migration / Data deletion]

### Risk Assessment
- **Probability of Failure:** H / M / L
- **Impact of Failure:** H / M / L
- **Risk Score:** [P × I]

### Implementation Plan
1. [Step-by-step deployment procedure]
2. [Include pre-deployment checks]
3. [Include verification steps]

### Rollback Plan
1. [Step-by-step rollback procedure]
2. [Expected rollback duration]
3. [Data recovery steps if applicable]

### Testing Evidence
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Load test (if performance-sensitive)
- [ ] DAST scan (if security-relevant)
- [ ] Staging deployment verified

### Approvals
| Role | Name | Decision | Date |
|------|------|----------|------|
| Tech Lead | | Approve / Reject | |
| Reviewer | | Approve / Reject | |
| Product Owner | | Approve / Reject (if user-facing) | |
```

---

## 3. Approval Workflow

### Standard Changes
```
Developer → Commit → CI passes → Auto-deploy to staging → Verify → Merge to main → Auto-deploy
```
No manual approval gate. Pre-authorized by category definition.

### Normal Changes
```
Developer → Create CR → Tech Lead Review → Peer Review → Schedule Window → Deploy → Verify → Close CR
                              │                  │
                              ▼                  ▼
                         Reject → Revise    Reject → Revise
```

### Emergency Changes
```
Developer → Verbal approval (Tech Lead) → Implement fix → Deploy → Verify → Post-hoc CR documentation
                                                                                      │
                                                                                      ▼
                                                                              Incident post-mortem
```

### Approval Matrix

| Change Type | Tech Lead | Peer Review | Product Owner | CAB |
|-------------|:---------:|:-----------:|:-------------:|:---:|
| Standard | — | — | — | — |
| Normal (code) | ✅ | ✅ | — | — |
| Normal (user-facing) | ✅ | ✅ | ✅ | — |
| Normal (infrastructure) | ✅ | ✅ | — | — |
| Normal (database schema) | ✅ | ✅ | — | — |
| Emergency | ✅ (verbal) | Post-hoc | — | — |

---

## 4. Impact Assessment Criteria

### Severity Classification

| Level | User Impact | System Impact | Response Time |
|-------|-------------|---------------|:-------------:|
| **SEV-1** | All users affected, core flow broken | Full system outage or data loss | Immediate (< 15 min) |
| **SEV-2** | Subset of users affected, workaround exists | Single service degraded | < 1 hour |
| **SEV-3** | Minor inconvenience, no data impact | Performance degradation | < 4 hours |
| **SEV-4** | No user impact | Internal tooling issue | Next business day |

### Impact Dimensions

| Dimension | Questions to Assess |
|-----------|-------------------|
| **Availability** | Will this cause downtime? How long? Which services? |
| **Performance** | Will latency increase? By how much? For which endpoints? |
| **Data Integrity** | Does this modify data? Is it reversible? What's the blast radius? |
| **Security** | Does this change auth, access control, or expose new attack surface? |
| **Compatibility** | Does this break existing API contracts? Are clients affected? |
| **Observability** | Will monitoring still work? Are new metrics/alerts needed? |

### Change Risk Score

| Factor | Low (1) | Medium (2) | High (3) |
|--------|---------|------------|----------|
| Complexity | Single file change | Multi-service change | Architecture change |
| Blast radius | One service | Multiple services | Full system |
| Reversibility | Git revert | Manual rollback steps | Irreversible (data) |
| Testing confidence | Full coverage | Partial coverage | Untested path |
| Deployment frequency | Done before | Similar done before | First time |

**Total Score:** Sum of factors (5–15)
- 5–7: Low risk → Standard process
- 8–11: Medium risk → Normal process with extra review
- 12–15: High risk → Normal process with staging soak period (24h)

---

## 5. Rollback Procedures

### 5.1 Application Rollback (Container)

**Trigger:** Health check failures, error rate > SLO threshold, user-reported critical bug

```bash
# Option A: Revert to previous image tag (Coolify)
# 1. Identify last known good image
docker images ghcr.io/piresc/parkir-pintar-* --format "{{.Tag}} {{.CreatedAt}}" | head -5

# 2. Update Coolify deployment to previous tag
# Via Coolify UI: Applications → [service] → Deployments → Rollback

# Option B: Git revert + redeploy
git revert HEAD --no-edit
git push origin main
# CI/CD will build and deploy the reverted state

# Option C: Watchtower rollback (if configured)
# Watchtower will detect unhealthy container and revert to previous image
# Requires WATCHTOWER_ROLLING_RESTART=true and health checks configured
```

**Verification after rollback:**
1. Check service health endpoints return 200
2. Verify error rate returns to baseline in Grafana
3. Confirm no data inconsistency from partial deployment

### 5.2 Database Migration Rollback

**Trigger:** Migration fails mid-execution, data corruption detected, application incompatible with new schema

```bash
# 1. Check current migration version
migrate -path migrations -database "$DATABASE_URL" version

# 2. Roll back one version
migrate -path migrations -database "$DATABASE_URL" down 1

# 3. If migration was partially applied (dirty state)
migrate -path migrations -database "$DATABASE_URL" force <last_good_version>

# 4. Verify schema state
psql $DATABASE_URL -c "\dt" # list tables
psql $DATABASE_URL -c "SELECT * FROM schema_migrations;"
```

**Prevention:**
- All migrations must be transactional (wrapped in BEGIN/COMMIT)
- Every `up` migration must have a corresponding `down` migration
- Test migrations against production-size dataset in staging
- Take pg_dump backup before applying to production

### 5.3 Infrastructure Rollback (Terraform)

**Trigger:** Terraform apply causes service disruption, misconfigured resources

```bash
# 1. Check Terraform state for drift
terraform plan

# 2. Revert to previous state
# Option A: Apply previous commit's Terraform
git checkout HEAD~1 -- terraform/
terraform apply

# Option B: Targeted resource revert
terraform state show <resource>
terraform apply -target=<resource> -var="previous_config=true"

# 3. For emergency: manual GCP console intervention
# Document all manual changes for later IaC reconciliation
```

### 5.4 Configuration Rollback

**Trigger:** Misconfigured environment variables, wrong feature flags

```bash
# 1. Environment variables (Coolify)
# Via Coolify UI: Applications → [service] → Environment → Revert

# 2. Feature flags (if implemented)
# Toggle flag off via admin API or config update

# 3. Restart affected services
docker restart <container_name>
```

### 5.5 Rollback Decision Matrix

| Scenario | Action | Expected Duration | Data Risk |
|----------|--------|:-----------------:|:---------:|
| Bad deploy, no data changes | Revert container image | < 5 min | None |
| Bad deploy with schema migration | Rollback migration + revert image | < 15 min | Low |
| Partial data corruption | Restore from backup + replay events | < 1 hour | Medium |
| Full system failure | Restore from last backup | < 2 hours | High |
| Security breach | Isolate + rotate credentials + restore | < 30 min | Variable |

---

## 6. Change Freeze Periods

| Period | Reason | Exceptions |
|--------|--------|-----------|
| Assessment demo day (±1 day) | Stability for demonstration | Emergency security fixes only |
| Major load test windows | Baseline integrity | None |
| Friday 5PM — Monday 9AM | Reduced team availability | Emergency changes only |

---

## 7. Post-Change Review

### For Normal Changes
- Verify deployment success within 30 minutes
- Monitor error rate and latency for 2 hours post-deploy
- Close change request with outcome notes

### For Emergency Changes
- Complete CR documentation within 24 hours
- Conduct incident post-mortem within 48 hours
- Identify preventive actions to avoid recurrence
- Update runbooks if new failure mode discovered

### Change Success Metrics

| Metric | Target | Current |
|--------|:------:|:-------:|
| Change success rate | > 95% | 97% |
| Mean time to deploy (normal) | < 30 min | 22 min |
| Mean time to rollback | < 10 min | 7 min |
| Emergency change ratio | < 10% | 5% |
| Changes causing incidents | < 5% | 3% |

---

## 8. Change Log

| CR ID | Date | Category | Description | Outcome |
|-------|------|----------|-------------|---------|
| CR-2026-001 | 2026-03-30 | Normal | Deploy gRPC services to staging | ✅ Success |
| CR-2026-002 | 2026-04-13 | Normal | Add OTel pipeline + Grafana dashboards | ✅ Success |
| CR-2026-003 | 2026-04-20 | Normal | Database migration: add billing indexes | ✅ Success |
| CR-2026-004 | 2026-04-27 | Normal | CI/CD pipeline with Trivy scanning | ✅ Success |
| CR-2026-005 | 2026-05-02 | Standard | Update Go dependencies (patch versions) | ✅ Success |
| CR-2026-006 | 2026-05-05 | Normal | Add rate limiting middleware | ✅ Success |
| CR-2026-007 | 2026-05-08 | Emergency | Fix connection pool exhaustion under load | ✅ Success |
