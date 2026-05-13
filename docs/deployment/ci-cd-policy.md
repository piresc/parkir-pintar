# ParkirPintar CI/CD Policy

**Document Owner:** Platform Engineering Team
**Last Updated:** 2025-01-15
**Version:** 1.0

---

## 1. Purpose

This document defines the Continuous Integration and Continuous Deployment (CI/CD) policies for the ParkirPintar smart parking marketplace platform. It establishes standards for code quality, security, deployment safety, and operational excellence across all environments.

## 2. Pipeline Stages

Every change to the ParkirPintar codebase passes through the following pipeline stages in order. A failure at any stage blocks progression to subsequent stages.

### Stage 1: Secret Scan

| Aspect | Detail |
|--------|--------|
| Tool | `gitleaks`, `trufflehog` |
| Trigger | Every push, every PR |
| Scope | Full diff + commit history |
| Failure Action | Block merge, alert security team |

Scans for accidentally committed secrets including API keys, database credentials, JWT secrets, and private keys. Runs before any other stage to prevent secret propagation.

### Stage 2: Lint & Format

| Aspect | Detail |
|--------|--------|
| Tool | `golangci-lint` (Go), `hadolint` (Dockerfile), `yamllint` (YAML) |
| Config | `.golangci.yml` in repository root |
| Failure Action | Block merge |

Enforces consistent code style and catches common programming errors. All linting rules are defined in version-controlled configuration files.

### Stage 3: Test

| Aspect | Detail |
|--------|--------|
| Unit Tests | `go test ./...` with race detector |
| Integration Tests | Docker Compose test environment |
| Coverage Threshold | Minimum 70% line coverage |
| Timeout | 15 minutes maximum |
| Failure Action | Block merge |

Tests run in isolated containers with ephemeral databases. Integration tests verify inter-service communication (gRPC, NATS events).

### Stage 4: Security Scan

| Aspect | Detail |
|--------|--------|
| SAST | `gosec`, `semgrep` |
| Dependency Scan | `govulncheck`, `trivy` (filesystem mode) |
| Container Scan | `trivy` (image mode) |
| Failure Action | Block on HIGH/CRITICAL, warn on MEDIUM |

Security scanning covers source code, dependencies, and built container images. Critical vulnerabilities must be resolved before merge.

### Stage 5: Build

| Aspect | Detail |
|--------|--------|
| Builder | Multi-stage Dockerfile |
| Registry | `ghcr.io/piresc/parkir-pintar` |
| Tag Strategy | See [Release Process](./release-process.md) |
| Platforms | `linux/amd64` |
| Failure Action | Block deployment |

Produces a single multi-binary container image. Build artifacts are signed with cosign for supply chain integrity.

### Stage 6: Deploy

| Aspect | Detail |
|--------|--------|
| Strategy | Blue-green (production), rolling (staging) |
| Tool | Docker Compose + Traefik |
| Verification | Health checks + smoke tests |
| Failure Action | Automatic rollback |

Deployment is environment-specific with increasing gates. See Section 4 for environment details.

## 3. Branch Protection Rules

### Main Branch (`main`)

- **Require pull request**: All changes must come through a PR
- **Require CI pass**: All pipeline stages must succeed
- **Require review**: Minimum 1 approval from code owner
- **Require up-to-date**: Branch must be current with main before merge
- **No force push**: History rewriting is prohibited
- **No direct commits**: Includes repository administrators
- **Signed commits**: GPG or SSH signature required

### Release Branches (`release/*`)

- **Require pull request**: Cherry-picks only from main
- **Require CI pass**: Full pipeline must pass
- **Require review**: Minimum 2 approvals (1 must be tech lead)
- **No force push**: History rewriting is prohibited

### Feature Branches (`feature/*`, `fix/*`, `chore/*`)

- **Naming convention**: `<type>/<ticket-id>-<short-description>`
- **Lifetime**: Maximum 5 business days (stale branches auto-closed)
- **Rebase preferred**: Keep linear history where possible

## 4. Deployment Environments

### Development (`dev`)

| Aspect | Detail |
|--------|--------|
| Trigger | Push to any feature branch |
| Approval | None (automatic) |
| Data | Synthetic/seed data |
| URL | `dev-parkir-pintar.piresc.dev` |
| Retention | Ephemeral, destroyed on branch delete |

### Staging (`staging`)

| Aspect | Detail |
|--------|--------|
| Trigger | Merge to `main` |
| Approval | Automatic after CI passes |
| Data | Anonymized production subset |
| URL | `staging-parkir-pintar.piresc.dev` |
| Retention | Persistent, reset weekly |

### Production (`production`)

| Aspect | Detail |
|--------|--------|
| Trigger | Release tag (`v*.*.*`) |
| Approval | Manual — requires tech lead sign-off |
| Data | Live production data |
| URL | `parkir-pintar.piresc.dev` |
| Strategy | Blue-green with health verification |

## 5. Promotion Criteria

### Dev → Staging

- All CI pipeline stages pass
- PR approved by at least 1 reviewer
- No unresolved security findings (HIGH/CRITICAL)
- Feature flag coverage for incomplete features

### Staging → Production

- Staging deployment stable for minimum 24 hours
- Integration test suite passes on staging
- Performance benchmarks within acceptable thresholds:
  - API p95 latency < 200ms
  - Error rate < 0.1%
  - No memory leaks detected
- Product owner sign-off (for feature releases)
- Tech lead deployment approval
- No open P0/P1 bugs related to the release

## 6. Rollback Procedures

### Automatic Rollback

Triggered when:
- Health checks fail within 5 minutes of deployment
- Error rate exceeds 5% within 10 minutes
- Any service fails to start

Process:
1. Monitoring detects failure condition
2. Alert fires to on-call engineer
3. Blue-green switch reverts to previous deployment
4. Incident channel created automatically
5. Post-mortem scheduled within 48 hours

### Manual Rollback

```bash
# Production rollback via blue-green switch
cd deploy/blue-green
./switch.sh --rollback

# Staging rollback via image tag
IMAGE_TAG=v1.x.y docker compose -f deploy/staging/docker-compose.yml up -d
```

### Rollback SLA

| Environment | Detection | Execution | Total |
|-------------|-----------|-----------|-------|
| Production | < 2 min | < 1 min | < 3 min |
| Staging | < 5 min | < 2 min | < 7 min |

## 7. Hotfix Process

For critical production issues requiring immediate resolution:

1. **Branch**: Create `hotfix/PP-<ticket>-<description>` from latest release tag
2. **Fix**: Implement minimal fix with test coverage
3. **Review**: Expedited review (1 reviewer, can be async)
4. **CI**: Full pipeline must still pass (no skipping stages)
5. **Deploy**: Direct to production after staging smoke test (minimum 30 min soak)
6. **Backport**: Cherry-pick fix to `main` immediately after production deploy
7. **Version**: Increment patch version (e.g., `v1.2.1`)

### Hotfix Criteria

A hotfix is justified only when:
- Production service is degraded or down
- Data integrity is at risk
- Security vulnerability is actively exploited
- Revenue-impacting bug affects >5% of users

## 8. Release Versioning

ParkirPintar follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH

v1.0.0  → Initial production release
v1.1.0  → New feature (backward compatible)
v1.1.1  → Bug fix
v2.0.0  → Breaking API change
```

### Pre-release Tags

- `v1.2.0-rc.1` — Release candidate
- `v1.2.0-beta.1` — Beta (staging only)
- `v1.2.0-alpha.1` — Alpha (dev only)

### Version Locations

- Git tag (source of truth)
- Docker image tag
- `/health` endpoint response
- `version.go` constant (auto-updated by CI)

## 9. Artifact Retention Policy

| Artifact | Retention | Storage |
|----------|-----------|---------|
| Docker images (release tags) | Indefinite | GHCR |
| Docker images (PR/branch) | 7 days after branch delete | GHCR |
| Docker images (main) | 90 days | GHCR |
| CI logs | 30 days | GitHub Actions |
| Test reports | 90 days | GitHub Actions artifacts |
| SBOM (Software Bill of Materials) | Indefinite (per release) | GHCR attestation |
| Security scan results | 1 year | GitHub Security tab |

### Cleanup Automation

- Untagged images pruned weekly via GitHub Actions scheduled workflow
- Branch-tagged images deleted on branch deletion webhook
- Old `main` images pruned monthly (keep last 10)

## 10. Secret Management Policy

### Principles

- **No secrets in code**: Zero tolerance for hardcoded credentials
- **No secrets in CI logs**: All secret values masked in output
- **Least privilege**: Each environment has its own credentials
- **Rotation**: All secrets rotated quarterly (minimum)

### Secret Storage

| Environment | Store | Access |
|-------------|-------|--------|
| Local dev | `.env` file (gitignored) | Developer machine |
| CI/CD | GitHub Actions Secrets | Repository admins |
| Staging | GitHub Environment Secrets | Deployment team |
| Production | GitHub Environment Secrets + manual approval | Tech leads only |

### Required Secrets

| Secret | Rotation | Owner |
|--------|----------|-------|
| `DB_PASSWORD` | 90 days | Platform team |
| `REDIS_PASSWORD` | 90 days | Platform team |
| `JWT_SECRET` | 180 days | Security team |
| `GHCR_TOKEN` | 365 days | CI/CD admin |

### Secret Rotation Process

1. Generate new secret value
2. Update in secret store
3. Deploy with new secret (blue-green ensures zero downtime)
4. Verify services authenticate correctly
5. Revoke old secret after 24-hour grace period
6. Update rotation log

## 11. Access Control for Deployments

### Role-Based Access

| Role | Dev | Staging | Production | Rollback |
|------|-----|---------|------------|----------|
| Developer | ✅ Deploy | ❌ | ❌ | ❌ |
| Senior Developer | ✅ Deploy | ✅ Deploy | ❌ | ❌ |
| Tech Lead | ✅ Deploy | ✅ Deploy | ✅ Approve | ✅ Execute |
| Platform Engineer | ✅ Deploy | ✅ Deploy | ✅ Execute | ✅ Execute |
| On-Call Engineer | ❌ | ❌ | ❌ | ✅ Execute |

### Deployment Approval Flow (Production)

1. Developer creates release PR
2. CI pipeline passes all stages
3. Tech lead reviews and approves
4. GitHub Environment protection rule enforces approval
5. Deployment executes automatically after approval
6. On-call engineer monitors post-deployment

### Emergency Access

During P0 incidents, on-call engineers can execute rollbacks without prior approval. All emergency actions are logged and reviewed in post-mortem.

## 12. Audit Trail Requirements

### What Is Logged

- Every deployment (who, what version, when, which environment)
- Every rollback (trigger: automatic or manual, reason)
- Every approval/rejection of deployment
- Secret access and rotation events
- Pipeline failures and their resolution
- Access control changes

### Log Format

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "event": "deployment",
  "actor": "github-actions[bot]",
  "approver": "tech-lead-username",
  "environment": "production",
  "version": "v1.2.0",
  "previous_version": "v1.1.3",
  "strategy": "blue-green",
  "target_color": "green",
  "status": "success",
  "duration_seconds": 45,
  "commit_sha": "abc123def456"
}
```

### Retention

- Deployment audit logs: 2 years
- Access control change logs: 5 years
- Security event logs: 5 years

### Monitoring & Alerting

- Failed deployments → Immediate alert to deployment team
- Unauthorized access attempts → Alert to security team
- Unusual deployment patterns (off-hours, rapid succession) → Alert to tech lead

---

## Appendix A: Pipeline Configuration Reference

```yaml
# .github/workflows/ci.yml (simplified)
name: ParkirPintar CI/CD

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  release:
    types: [published]

jobs:
  secret-scan:
    # Stage 1
  lint:
    # Stage 2 — depends on secret-scan
  test:
    # Stage 3 — depends on lint
  security:
    # Stage 4 — depends on test
  build:
    # Stage 5 — depends on security
  deploy-staging:
    # Stage 6a — depends on build, only on main
  deploy-production:
    # Stage 6b — depends on build, only on release tag
    environment: production  # Requires manual approval
```

## Appendix B: Compliance

This CI/CD policy supports compliance with:
- **SOC 2 Type II**: Change management controls, audit trails
- **PCI DSS**: Secure development lifecycle, access controls
- **ISO 27001**: Information security management

---

*This is a living document. Changes require PR approval from the Platform Engineering team lead.*
