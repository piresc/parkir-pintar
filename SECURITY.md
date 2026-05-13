# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in ParkirPintar, please report it responsibly.

**Email:** security@piresc.dev

**Expected Response Time:** 48 hours for initial acknowledgment

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Affected service(s) and version(s)
- Potential impact assessment
- Any suggested fix (optional)

### What to Expect

1. **Acknowledgment** within 48 hours of your report
2. **Assessment** within 7 days — we'll confirm the vulnerability and its severity
3. **Fix timeline** communicated within 14 days
4. **Patch release** as soon as a fix is validated

## Responsible Disclosure Policy

We follow a **90-day disclosure timeline**:

- After reporting, we have 90 days to address the vulnerability before public disclosure
- If a fix is released before the 90-day window, disclosure may happen sooner with mutual agreement
- Critical vulnerabilities affecting user data may receive expedited patches
- We will credit reporters in our security advisories (unless anonymity is requested)

## Security Update Process

1. Vulnerability is confirmed and triaged by severity (Critical, High, Medium, Low)
2. A fix is developed on a private branch
3. Fix is reviewed and tested in isolation
4. Patch is released with a security advisory
5. All deployment environments are updated within 24h of patch release (Critical/High)
6. Post-mortem is conducted for Critical/High severity issues

## Bug Bounty

A formal bug bounty program is **not applicable** at this time. We appreciate responsible disclosure and will acknowledge contributors in our security advisories.

## Security-Related Configuration

### JWT (JSON Web Tokens)

- Access tokens: short-lived (15 minutes)
- Refresh tokens: longer-lived with rotation
- Algorithm: RS256 with key rotation support
- Tokens are validated on every request at the gateway level

### Rate Limiting

- Per-IP rate limiting on all public endpoints
- Per-user rate limiting on authenticated endpoints
- Stricter limits on authentication endpoints to prevent brute-force
- Distributed rate limiting via Redis for consistency across instances

### TLS

- TLS 1.2+ required for all external communication
- Internal service-to-service communication secured via mTLS where applicable
- Certificates managed via Coolify/reverse proxy
- HSTS headers enforced

## Known Security Measures

- **Input validation** on all API boundaries using protobuf schema validation
- **SQL injection prevention** via parameterized queries (sqlc generated code)
- **Authentication** via JWT with proper token lifecycle management
- **Authorization** with role-based access control (RBAC)
- **Secrets management** — no secrets in code; environment-based configuration
- **Dependency scanning** via GitHub Dependabot and CI pipeline checks
- **DAST scanning** integrated in CI/CD pipeline (ZAP)
- **Container security** — minimal base images, non-root execution
- **Distributed locking** via Redis to prevent race conditions in parking operations
- **Audit logging** for sensitive operations
- **CORS** properly configured per environment
- **OpenTelemetry** tracing for security event correlation
