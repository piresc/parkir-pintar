# Kompetensi 5 — Software Security (Level 4) Gap Analysis

## Overview

This document analyzes the parkir-pintar project's current security posture against BNSP Level 4 Software Security competency requirements. The assessment covers secure development practices, security testing automation, incident response capabilities, and compliance documentation.

---

## Level 4 Requirements Summary

| # | Requirement | Status |
|---|-------------|--------|
| 1 | Implementasi standar security (PCI/DSS, OWASP Pentest Checklist) | ⚠️ Partial |
| 2 | Implementasi security testing tools dalam development | ⚠️ Partial |
| 3 | Langkah preventif untuk mencegah kebocoran | ⚠️ Partial |
| 4 | Langkah recovery terhadap data bocor/hilang | ❌ Missing |
| 5 | Laporan post mortem terhadap security breach | ❌ Missing |
| 6 | **Assessment:** Automated security testing dalam CI/CD | ⚠️ Partial |

---

## 1. Authentication & Authorization

### Level 4 Requires
- Secure token management with algorithm pinning
- Multi-layer authentication across all service boundaries
- Role-Based Access Control (RBAC) enforcement
- Secrets management with vault integration

### What EXISTS ✅

| Component | File Path | Details |
|-----------|-----------|---------|
| JWT Auth | `pkg/auth/jwt.go` | golang-jwt/jwt/v5, HS256 algorithm pinning, empty secret rejection, expiration enforcement |
| HTTP Auth Middleware | `pkg/middleware/auth.go` | Bearer token extraction, claims propagation |
| gRPC Auth Interceptor | `pkg/grpcmiddleware/auth.go` | Unary + stream interceptors, metadata extraction |
| API Key Auth | Gateway service | Secondary auth mechanism for service-to-service |
| JWT Secret Validation | Config startup | `JWT_SECRET` validated as required, rejects empty values |
| Auth Tests | `pkg/auth/jwt_test.go` | 9 test cases covering edge cases |
| E2E Auth Tests | `tests/e2e_docker/middleware_test.go` | 401 without JWT, 401 invalid JWT |
| Property-Based Tests | Auth middleware | Fuzz testing for token validation |

### What's MISSING ❌

- **RBAC enforcement in handlers** — Role is extracted from JWT claims but never checked at handler level
- **Secrets vault integration** — No HashiCorp Vault or AWS Secrets Manager; secrets in env vars only
- **Password hashing** — No bcrypt/argon2 implementation (may rely on external auth provider)
- **Token refresh/rotation** — No refresh token mechanism documented
- **Session invalidation** — No token blacklist or revocation mechanism

### ACTION ITEMS

| Priority | Task | Suggested Location |
|----------|------|--------------------|
| HIGH | Implement RBAC middleware that checks role claims against endpoint permissions | `pkg/middleware/rbac.go` |
| HIGH | Add role-permission mapping configuration | `configs/rbac_policy.yaml` |
| MEDIUM | Integrate HashiCorp Vault for secrets management | `pkg/vault/client.go` |
| MEDIUM | Add token refresh endpoint | `internal/gateway/handler_refresh.go` |
| LOW | Implement token blacklist with Redis TTL | `pkg/auth/blacklist.go` |

---

## 2. Input Validation & Injection Prevention

### Level 4 Requires
- Comprehensive input validation at all entry points
- SQL injection prevention
- SSRF protection
- XSS prevention

### What EXISTS ✅

| Component | File Path | Details |
|-----------|-----------|---------|
| gRPC Input Validation | All service handlers | Required field checks, returns `codes.InvalidArgument` |
| SQL Injection Prevention | All repository layers | Parameterized queries ($1, $2) via sqlx |
| CORS Configuration | Gateway service | Origin allowlisting configured |
| Idempotency Middleware | Gateway | Prevents replay attacks |

### What's MISSING ❌

- **SSRF protection** — HTTP client has no URL allowlisting; A10 risk
- **Security headers** — No HSTS, X-Content-Type-Options, X-Frame-Options, CSP; A05 risk
- **Request body size limits** — Not explicitly configured at gateway level
- **Content-Type validation** — No strict enforcement on API endpoints

### ACTION ITEMS

| Priority | Task | Suggested Location |
|----------|------|--------------------|
| HIGH | Add security headers middleware (HSTS, X-Content-Type, X-Frame-Options, CSP) | `pkg/middleware/security_headers.go` |
| HIGH | Implement URL allowlist for outbound HTTP calls | `pkg/httpclient/ssrf_guard.go` |
| MEDIUM | Add request body size limits at gateway | `internal/gateway/config.go` |
| MEDIUM | Implement Content-Type validation middleware | `pkg/middleware/content_type.go` |

---

## 3. Network & Transport Security

### Level 4 Requires
- TLS encryption for all communications
- mTLS for service-to-service communication
- Secure configuration of all network components

### What EXISTS ✅

| Component | File Path | Details |
|-----------|-----------|---------|
| TLS Config Support | Service configs | `GRPC_TLS_CERT_PATH`, `GRPC_TLS_KEY_PATH` environment variables |
| DB SSL Mode | Config | `DB_SSL_MODE` configurable per environment |
| Non-root Containers | Dockerfiles | `appuser` with minimal privileges |

### What's MISSING ❌

- **mTLS between services** — No mutual TLS for inter-service gRPC calls
- **TLS enforcement** — TLS paths are optional, not enforced in production
- **Certificate rotation** — No automated cert renewal mechanism
- **Network policies** — No Kubernetes NetworkPolicy definitions

### ACTION ITEMS

| Priority | Task | Suggested Location |
|----------|------|--------------------|
| HIGH | Implement mTLS for gRPC inter-service communication | `pkg/grpcmiddleware/mtls.go` |
| HIGH | Add TLS enforcement for production environment | `configs/production.yaml` |
| MEDIUM | Add cert-manager configuration for auto-rotation | `deploy/k8s/cert-manager/` |
| MEDIUM | Define Kubernetes NetworkPolicies | `deploy/k8s/network-policies/` |

---

## 4. Security Testing & CI/CD Integration

### Level 4 Requires
- SAST (Static Application Security Testing) in pipeline
- DAST (Dynamic Application Security Testing) in pipeline
- Dependency vulnerability scanning
- Secret scanning
- Automated security gates that block deployment

### What EXISTS ✅

| Component | File Path | Details |
|-----------|-----------|---------|
| SAST — gosec | `.gitlab-ci.yml` | Static analysis, `allow_failure: false` (blocks pipeline) |
| Secret Scanning — gitleaks | `.gitlab-ci.yml` | Detects leaked secrets, `allow_failure: false` |
| Gitleaks Config | `.gitleaks.toml` | Custom rules and allowlists |
| SonarCloud | `.gitlab-ci.yml` | Code quality + security hotspots |
| .gitignore | `.gitignore` | Excludes .env, config/.env |
| .env.example | `.env.example` | Placeholder values only, no real secrets |

### What's MISSING ❌

- **DAST (OWASP ZAP)** — `ZAP_DOCKER_VERSION` is defined in CI but the ZAP scan stage is not implemented
- **Dependency vulnerability scanning** — No `govulncheck` or `nancy` in pipeline
- **Container image scanning** — No Trivy or Grype for Docker image CVEs
- **License compliance scanning** — No license audit for dependencies
- **Security test coverage metrics** — No tracking of security-specific test coverage

### ACTION ITEMS

| Priority | Task | Suggested Location |
|----------|------|--------------------|
| **CRITICAL** | Implement OWASP ZAP DAST stage in CI/CD | `.gitlab-ci.yml` (dast stage) |
| **CRITICAL** | Add govulncheck for Go dependency scanning | `.gitlab-ci.yml` (security stage) |
| HIGH | Add Trivy container image scanning | `.gitlab-ci.yml` (container_scan stage) |
| MEDIUM | Add license compliance check | `.gitlab-ci.yml` (compliance stage) |
| MEDIUM | Create ZAP scan configuration | `tests/security/zap-config.yaml` |

### Suggested CI/CD Security Stage

```yaml
# .gitlab-ci.yml additions
govulncheck:
  stage: security
  image: golang:1.22
  script:
    - go install golang.org/x/vuln/cmd/govulncheck@latest
    - govulncheck ./...
  allow_failure: false

dast_zap:
  stage: dast
  image: ghcr.io/zaproxy/zaproxy:${ZAP_DOCKER_VERSION}
  script:
    - zap-api-scan.py -t $API_SPEC_URL -f openapi -r zap-report.html
  artifacts:
    paths:
      - zap-report.html
  allow_failure: false

trivy_scan:
  stage: security
  image: aquasec/trivy:latest
  script:
    - trivy image --exit-code 1 --severity HIGH,CRITICAL $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
  allow_failure: false
```

---

## 5. Incident Response & Recovery

### Level 4 Requires
- Documented incident response plan
- Data recovery procedures
- Post-mortem template and process
- Preventive measures documentation
- Responsible disclosure policy

### What EXISTS ✅

| Component | Details |
|-----------|---------|
| Circuit Breaker | `pkg/circuitbreaker/` — 3-state, thread-safe; prevents cascade failures |
| Distributed Locks | Prevents race conditions during concurrent operations |
| Rate Limiting | HTTP + gRPC token bucket; mitigates DoS |

### What's MISSING ❌

- **SECURITY.md** — No responsible disclosure policy
- **Incident Response Plan** — No documented procedure for security breaches
- **Data Recovery Procedures** — No documented backup/restore process
- **Post-Mortem Template** — No standardized format for breach analysis
- **Preventive Measures Documentation** — No lessons-learned process
- **Runbooks** — No operational security runbooks

### ACTION ITEMS

| Priority | Task | Suggested Location |
|----------|------|--------------------|
| **CRITICAL** | Create SECURITY.md with responsible disclosure policy | `SECURITY.md` |
| **CRITICAL** | Write incident response plan | `docs/security/incident-response-plan.md` |
| **CRITICAL** | Create post-mortem template | `docs/security/post-mortem-template.md` |
| HIGH | Document data recovery procedures | `docs/security/data-recovery.md` |
| HIGH | Write preventive measures playbook | `docs/security/preventive-measures.md` |
| MEDIUM | Create operational security runbooks | `docs/security/runbooks/` |

### Suggested Post-Mortem Template Structure

```markdown
# Security Incident Post-Mortem: [INCIDENT-ID]

## Summary
## Timeline
## Root Cause Analysis
## Impact Assessment
  - Data affected
  - Users affected
  - Services affected
## Detection & Response
  - How was it detected?
  - Response timeline
  - Containment actions
## Recovery Actions
## Preventive Measures
  - Short-term fixes
  - Long-term improvements
## Lessons Learned
## Action Items (with owners and deadlines)
```

---

## 6. Compliance & Standards

### Level 4 Requires
- PCI/DSS compliance documentation (if handling payment data)
- OWASP compliance evidence
- Security policy documentation

### What EXISTS ✅

| Component | Details |
|-----------|---------|
| OWASP Top 10 Coverage | 8/10 risks mitigated through implementation |
| Secure coding practices | Parameterized queries, input validation, auth enforcement |

### What's MISSING ❌

- **PCI/DSS compliance documentation** — No formal compliance mapping
- **OWASP Pentest Checklist** — No documented checklist execution
- **Security policy document** — No organizational security policy
- **Data classification** — No data sensitivity classification scheme
- **Privacy impact assessment** — No PIA documentation

### ACTION ITEMS

| Priority | Task | Suggested Location |
|----------|------|--------------------|
| HIGH | Create OWASP compliance checklist with evidence | `docs/security/owasp-compliance.md` |
| HIGH | Document PCI/DSS applicability and controls | `docs/security/pci-dss-mapping.md` |
| MEDIUM | Write security policy document | `docs/security/security-policy.md` |
| MEDIUM | Create data classification scheme | `docs/security/data-classification.md` |

---

## OWASP Top 10 (2021) Compliance Matrix

| # | Risk | Status | Evidence | Gap |
|---|------|--------|----------|-----|
| A01 | Broken Access Control | ⚠️ Partial | JWT auth on all endpoints, API key for services | RBAC not enforced at handler level |
| A02 | Cryptographic Failures | ✅ Covered | HS256 algorithm pinning, TLS support, DB SSL mode | mTLS not implemented |
| A03 | Injection | ✅ Covered | Parameterized SQL ($1,$2), gRPC input validation | — |
| A04 | Insecure Design | ✅ Covered | Circuit breaker, rate limiting, distributed locks, idempotency | — |
| A05 | Security Misconfiguration | ❌ Gap | Non-root containers, .gitignore secrets | **Missing security headers (HSTS, CSP, X-Frame-Options)** |
| A06 | Vulnerable Components | ⚠️ Partial | gosec SAST in CI | **No govulncheck/dependency scanning** |
| A07 | Auth Failures | ✅ Covered | JWT expiration, empty secret rejection, rate limiting | — |
| A08 | Software/Data Integrity | ✅ Covered | Gitleaks secret scanning, idempotency keys | — |
| A09 | Logging & Monitoring | ⚠️ Partial | Structured logging present | No security event alerting |
| A10 | SSRF | ❌ Gap | — | **No URL allowlisting on HTTP client** |

### OWASP Score: 6/10 Fully Covered, 3/10 Partial, 1/10 Gap (Critical)

---

## 7. Data Protection & Privacy

### Level 4 Requires
- Data encryption at rest and in transit
- Backup and recovery mechanisms
- Data breach notification procedures

### What EXISTS ✅

| Component | Details |
|-----------|---------|
| TLS in transit | Configurable via env vars |
| DB SSL | `DB_SSL_MODE` configurable |
| Secret exclusion | .gitignore, .env.example with placeholders |

### What's MISSING ❌

- **Encryption at rest** — No application-level field encryption for PII
- **Backup automation** — No documented/automated PostgreSQL backup strategy
- **Data breach notification** — No procedure for notifying affected users
- **Data retention policy** — No defined retention/deletion schedules
- **PII inventory** — No mapping of where personal data is stored

### ACTION ITEMS

| Priority | Task | Suggested Location |
|----------|------|--------------------|
| HIGH | Implement automated PostgreSQL backup with pg_dump/WAL archiving | `deploy/scripts/backup.sh` |
| HIGH | Document data breach notification procedure | `docs/security/breach-notification.md` |
| MEDIUM | Create PII data inventory | `docs/security/pii-inventory.md` |
| MEDIUM | Implement field-level encryption for sensitive data | `pkg/crypto/field_encrypt.go` |
| LOW | Define data retention policy | `docs/security/data-retention.md` |

---

## Assessment Criteria Alignment

### Primary Assessment: "Mengimplementasikan automated security testing tool dalam proses CI/CD"

| Requirement | Current State | Target State | Gap |
|-------------|---------------|--------------|-----|
| SAST in CI | ✅ gosec (blocking) | ✅ Met | — |
| Secret scanning in CI | ✅ gitleaks (blocking) | ✅ Met | — |
| DAST in CI | ❌ ZAP defined but unused | OWASP ZAP API scan | **Implement ZAP stage** |
| Dependency scanning | ❌ Not present | govulncheck in pipeline | **Add govulncheck** |
| Container scanning | ❌ Not present | Trivy image scan | **Add Trivy** |
| Security quality gate | ⚠️ gosec + gitleaks block | All tools block on HIGH/CRITICAL | **Add remaining gates** |

---

## Priority Roadmap

### Phase 1 — Critical (Week 1-2)
1. ✍️ Create `SECURITY.md` responsible disclosure policy
2. ✍️ Write incident response plan
3. ✍️ Create post-mortem template
4. 🔧 Implement OWASP ZAP DAST in CI/CD pipeline
5. 🔧 Add govulncheck dependency scanning to CI/CD

### Phase 2 — High Priority (Week 3-4)
6. 🔧 Add security headers middleware (HSTS, CSP, X-Frame-Options, X-Content-Type-Options)
7. 🔧 Implement RBAC middleware with role-permission checks
8. 🔧 Add SSRF protection (URL allowlist) on HTTP client
9. ✍️ Document data recovery procedures
10. 🔧 Add Trivy container image scanning

### Phase 3 — Medium Priority (Week 5-8)
11. 🔧 Implement mTLS for inter-service communication
12. 🔧 Integrate HashiCorp Vault for secrets management
13. ✍️ Create OWASP compliance checklist with evidence
14. ✍️ Document PCI/DSS applicability mapping
15. 🔧 Implement automated PostgreSQL backup strategy

### Phase 4 — Hardening (Week 9-12)
16. 🔧 Add security event alerting and monitoring
17. ✍️ Create operational security runbooks
18. 🔧 Implement field-level encryption for PII
19. ✍️ Define data retention and classification policies
20. 🔧 Add Kubernetes NetworkPolicies

---

## Summary

| Category | Score | Notes |
|----------|-------|-------|
| Authentication & Authorization | 70% | Strong JWT, missing RBAC enforcement |
| Input Validation | 75% | Good SQL/gRPC validation, missing headers & SSRF |
| Network Security | 40% | TLS support exists but mTLS missing |
| CI/CD Security Testing | 55% | SAST + secrets done, DAST + deps missing |
| Incident Response | 10% | Only runtime protections, no documentation |
| Compliance | 30% | Implementation exists, documentation missing |
| Data Protection | 35% | Basic TLS/SSL, no backup/recovery docs |

### Overall Security Maturity: ~45% of Level 4 Requirements Met

**Critical path to Level 4 assessment:** Implement DAST (ZAP) + dependency scanning in CI/CD, create incident response documentation (SECURITY.md, post-mortem template, recovery procedures), and add security headers middleware.
