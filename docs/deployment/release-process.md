# ParkirPintar Release Process

**Document Owner:** Platform Engineering Team
**Last Updated:** 2025-01-15
**Version:** 1.0

---

## 1. Version Numbering

ParkirPintar uses [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH
  │     │     └── Bug fixes, security patches (backward compatible)
  │     └──────── New features (backward compatible)
  └────────────── Breaking changes (API contract changes)
```

### Examples

| Change | Version Bump | Example |
|--------|-------------|---------|
| Fix parking slot availability race condition | PATCH | `v1.2.3` → `v1.2.4` |
| Add parking reservation notifications | MINOR | `v1.2.4` → `v1.3.0` |
| Restructure billing API response format | MAJOR | `v1.3.0` → `v2.0.0` |

### Version Source of Truth

The Git tag is the single source of truth for versioning. The CI pipeline injects the version into:
- Docker image tags
- Application `/health` endpoint
- Build metadata in binary

## 2. Release Branch Workflow

### Standard Release

```
main ─────●────●────●────●────●────●────────────────────
           \                       ↑
            \                     merge
             \                   /
release/v1.2 ─●────●────●──────● ← tag: v1.2.0
              rc.1  rc.2  final
```

### Steps

1. **Create release branch** from `main`:
   ```bash
   git checkout main
   git pull origin main
   git checkout -b release/v1.2
   git push -u origin release/v1.2
   ```

2. **Stabilize** on the release branch:
   - Only bug fixes and documentation changes
   - No new features
   - Tag release candidates: `v1.2.0-rc.1`, `v1.2.0-rc.2`

3. **Final release**:
   - Tag final version: `v1.2.0`
   - Merge release branch back to `main`
   - Delete release branch

### Hotfix Release

```
main ─────●────●────●────●────────────────
                          ↑
                         merge
                        /
tag: v1.2.0 ──●────●──● ← tag: v1.2.1
              hotfix/fix-payment
```

Hotfixes branch from the release tag, not from `main`. After deployment, the fix is cherry-picked back to `main`.

## 3. Changelog Generation

### Automated Changelog

Changelogs are generated automatically from conventional commit messages using `git-cliff`.

### Commit Message Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat` — New feature (MINOR bump)
- `fix` — Bug fix (PATCH bump)
- `perf` — Performance improvement
- `refactor` — Code refactoring
- `docs` — Documentation
- `test` — Test changes
- `ci` — CI/CD changes
- `chore` — Maintenance

**Scopes:** `gateway`, `search`, `reservation`, `billing`, `payment`, `presence`, `notification`, `infra`

### Example Commits

```
feat(reservation): add real-time slot availability via WebSocket
fix(payment): handle duplicate payment webhook callbacks
perf(search): add spatial index for nearby parking queries
docs(gateway): update API documentation for v1.2 endpoints
```

### Generated Changelog Format

```markdown
## [v1.2.0] - 2025-01-15

### Features
- **reservation**: Add real-time slot availability via WebSocket (#142)
- **search**: Support filtering by parking facility amenities (#138)

### Bug Fixes
- **payment**: Handle duplicate payment webhook callbacks (#145)
- **presence**: Fix sensor heartbeat timeout calculation (#143)

### Performance
- **search**: Add spatial index for nearby parking queries (#140)

### Breaking Changes
- None
```

### Changelog Generation Command

```bash
# Generate changelog for next release
git cliff --unreleased --tag v1.2.0 > CHANGELOG.md

# Generate full changelog
git cliff -o CHANGELOG.md
```

## 4. Tag Creation

### Tagging Process

```bash
# Ensure you're on the release branch with all changes
git checkout release/v1.2
git pull origin release/v1.2

# Create annotated tag
git tag -a v1.2.0 -m "Release v1.2.0

Features:
- Real-time slot availability
- Amenity-based search filtering

Fixes:
- Payment webhook deduplication
- Sensor heartbeat timeout"

# Push tag (triggers CI/CD build + deploy pipeline)
git push origin v1.2.0
```

### Tag Naming Convention

| Tag | Purpose | Deploys To |
|-----|---------|-----------|
| `v1.2.0-alpha.1` | Early development | Dev only |
| `v1.2.0-beta.1` | Feature complete, testing | Staging |
| `v1.2.0-rc.1` | Release candidate | Staging |
| `v1.2.0` | Production release | Production |

### Tag Protection

- Only tech leads and platform engineers can push release tags
- Tags matching `v*.*.*` trigger the production deployment pipeline
- Tags cannot be deleted or moved after creation

## 5. Docker Image Tagging Strategy

### Tag Matrix

| Source | Image Tag | Example | Lifetime |
|--------|-----------|---------|----------|
| Release tag | `v{semver}` | `v1.2.0` | Permanent |
| Release tag | `v{major}.{minor}` | `v1.2` | Overwritten on patch |
| Release tag | `v{major}` | `v1` | Overwritten on minor |
| Main branch | `main-{short-sha}` | `main-abc123f` | 90 days |
| Main branch | `main` | `main` | Always latest main |
| PR branch | `pr-{number}` | `pr-142` | Until PR closed |
| Release candidate | `v1.2.0-rc.1` | `v1.2.0-rc.1` | 30 days |

### Image Build & Push

```yaml
# CI builds and tags on release
- name: Build and push
  uses: docker/build-push-action@v5
  with:
    push: true
    tags: |
      ghcr.io/piresc/parkir-pintar:${{ github.ref_name }}
      ghcr.io/piresc/parkir-pintar:${{ steps.semver.outputs.major }}.${{ steps.semver.outputs.minor }}
      ghcr.io/piresc/parkir-pintar:${{ steps.semver.outputs.major }}
```

### Image Signing

All release images are signed with `cosign`:

```bash
cosign sign --key cosign.key ghcr.io/piresc/parkir-pintar:v1.2.0
```

Verification before deployment:

```bash
cosign verify --key cosign.pub ghcr.io/piresc/parkir-pintar:v1.2.0
```

## 6. Deployment Approval Gates

### Gate Matrix

| Gate | Dev | Staging | Production |
|------|-----|---------|------------|
| CI pipeline passes | ✅ Required | ✅ Required | ✅ Required |
| Code review approved | ❌ | ✅ Required | ✅ Required |
| Security scan clean | ❌ | ✅ Required | ✅ Required |
| Staging soak (24h) | ❌ | ❌ | ✅ Required |
| Tech lead approval | ❌ | ❌ | ✅ Required |
| Change window | ❌ | ❌ | ✅ Tue-Thu, 10:00-16:00 WIB |
| On-call acknowledged | ❌ | ❌ | ✅ Required |

### Production Change Windows

- **Standard releases**: Tuesday–Thursday, 10:00–16:00 WIB
- **Hotfixes**: Any time (with on-call acknowledgment)
- **Frozen periods**: No deployments during:
  - National holidays
  - Major promotional events
  - Scheduled maintenance windows

### Approval Process

1. Release PR created with changelog
2. CI pipeline completes successfully
3. Tech lead reviews deployment plan
4. On-call engineer acknowledges readiness
5. GitHub Environment protection rule requires manual approval click
6. Deployment proceeds automatically after approval

## 7. Post-Release Verification Checklist

Execute within 15 minutes of production deployment:

### Automated Checks (run by CI)

- [ ] All 7 services report `healthy` status
- [ ] Gateway `/health` returns correct version
- [ ] API smoke tests pass (auth, search, reservation flow)
- [ ] Error rate < 0.1% (compared to pre-deployment baseline)
- [ ] p95 latency < 200ms
- [ ] No new error patterns in logs (first 5 minutes)

### Manual Checks (performed by deployer)

- [ ] Verify deployment in Grafana dashboard
- [ ] Check NATS JetStream consumer lag
- [ ] Confirm database connection pool metrics stable
- [ ] Test critical user flow end-to-end:
  1. User login
  2. Search for parking
  3. Create reservation
  4. Process payment
  5. Receive notification
- [ ] Verify WebSocket connections (presence service)
- [ ] Check Redis memory usage trend

### Communication

- [ ] Post deployment notification in `#deployments` channel
- [ ] Update status page (if applicable)
- [ ] Notify product team of feature availability

### Sign-Off

```
Deployment: v1.2.0
Deployed by: @username
Time: 2025-01-15 14:30 WIB
Status: ✅ All checks passed
Monitoring: No anomalies after 15 min observation
```

## 8. Rollback Criteria

### Automatic Rollback Triggers

The system automatically rolls back when any of these conditions are met within 10 minutes of deployment:

| Condition | Threshold | Detection |
|-----------|-----------|-----------|
| Service health check failure | Any service unhealthy > 60s | Docker health check |
| HTTP error rate spike | > 5% of requests return 5xx | Traefik metrics |
| Response time degradation | p95 > 500ms (2.5x baseline) | Prometheus |
| Memory leak | RSS growth > 100MB/min | cAdvisor |
| Crash loop | > 3 restarts in 5 minutes | Docker events |

### Manual Rollback Decision

Initiate manual rollback when:

- Users report inability to complete parking reservations
- Payment processing failures exceed 1%
- Data inconsistency detected between services
- Security vulnerability discovered in deployed version
- Unexpected behavior not caught by automated checks

### Rollback Execution

```bash
# Quick rollback (< 30 seconds)
cd deploy/blue-green
./switch.sh --rollback

# Verify rollback
curl -s https://parkir-pintar.piresc.dev/health | jq .version
# Should show previous version
```

### Post-Rollback Actions

1. **Immediate** (within 5 minutes):
   - Confirm service restored
   - Notify team in `#incidents` channel
   - Page on-call if not already engaged

2. **Short-term** (within 1 hour):
   - Collect logs from failed deployment
   - Identify root cause
   - Create incident ticket

3. **Follow-up** (within 48 hours):
   - Conduct post-mortem
   - Document lessons learned
   - Create action items to prevent recurrence
   - Update test suite to catch the failure

### Rollback Does NOT Apply When

- The issue existed before the deployment (pre-existing bug)
- The issue is in shared infrastructure (database, Redis, NATS)
- A database migration has already been applied that is not backward-compatible (requires forward-fix instead)

---

## Appendix: Release Checklist Template

```markdown
## Release v1.X.0 Checklist

### Pre-Release
- [ ] Release branch created from main
- [ ] All target PRs merged to release branch
- [ ] Changelog generated and reviewed
- [ ] Version bumped in source
- [ ] Release candidate deployed to staging
- [ ] Staging soak period complete (24h minimum)
- [ ] No P0/P1 bugs open against this release

### Release Day
- [ ] Within change window (Tue-Thu, 10:00-16:00 WIB)
- [ ] On-call engineer acknowledged
- [ ] Tech lead approved deployment
- [ ] Tag created and pushed
- [ ] Image built and signed
- [ ] Blue-green deployment executed
- [ ] Post-release verification checklist complete
- [ ] Team notified

### Post-Release
- [ ] Release branch merged back to main
- [ ] Release branch deleted
- [ ] GitHub Release created with changelog
- [ ] Documentation updated (if API changes)
- [ ] Monitoring dashboards reviewed (24h post-deploy)
```

---

*This is a living document. Changes require PR approval from the Platform Engineering team lead.*
