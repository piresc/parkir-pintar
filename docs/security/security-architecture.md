# ParkirPintar Security Architecture

## Overview

This document defines the security architecture for ParkirPintar, covering authentication, authorization, data protection, and operational security controls across the microservices platform.

## Authentication

### Strategy: BYO-JWT (Bring Your Own JWT)

ParkirPintar delegates identity management to an external Identity Provider (IdP). The system validates JWTs issued by trusted providers.

```
┌──────────┐     ┌──────────┐     ┌──────────────┐     ┌───────────┐
│  Client  │────▶│  IdP     │────▶│  Gateway     │────▶│  Services │
│  (App)   │◀────│(External)│     │  (Validate)  │     │  (Trust)  │
└──────────┘     └──────────┘     └──────────────┘     └───────────┘
     │                                    │
     │  1. Authenticate with IdP          │
     │  2. Receive JWT                    │
     │  3. Send JWT in Authorization      │
     │  header to Gateway                 │
     │                                    │  4. Validate signature
     │                                    │  5. Extract claims
     │                                    │  6. Forward user context
```

### Token Validation (Gateway Middleware)

| Check                | Implementation                              |
|----------------------|---------------------------------------------|
| Signature            | RS256/ES256 verification with JWKS endpoint |
| Expiration           | `exp` claim validation                      |
| Issuer               | `iss` must match configured IdP             |
| Audience             | `aud` must include ParkirPintar service ID  |
| Not Before           | `nbf` claim validation                      |

### JWKS Caching

- JWKS fetched from IdP's `/.well-known/jwks.json`
- Cached in Redis with 1-hour TTL
- Background refresh every 30 minutes
- Fallback to cached keys if IdP is unreachable

## Authorization

### Role-Based Access Control (RBAC)

Roles are embedded in JWT claims and enforced at the gateway and service level.

| Role       | Description                    | Permissions                                    |
|------------|--------------------------------|------------------------------------------------|
| `driver`   | End-user parking vehicles      | Check-in, check-out, view own sessions         |
| `operator` | Parking location manager       | Manage locations, view reports, manage slots    |
| `admin`    | System administrator           | All permissions, user management, system config |

### JWT Claims Structure

```json
{
  "sub": "user-uuid-123",
  "iss": "https://idp.example.com",
  "aud": ["parkir-pintar"],
  "exp": 1716000000,
  "iat": 1715996400,
  "roles": ["driver"],
  "metadata": {
    "tenant_id": "tenant-001"
  }
}
```

### Authorization Enforcement Points

| Layer    | Mechanism                                    |
|----------|----------------------------------------------|
| Gateway  | Role-based route access (middleware)         |
| Service  | Business rule validation (usecase layer)     |
| Data     | Row-level filtering by user/tenant ID        |

### Permission Matrix

| Resource            | Driver         | Operator       | Admin          |
|---------------------|----------------|----------------|----------------|
| Parking sessions    | Own only       | Own locations  | All            |
| Locations           | Read           | CRUD (own)     | CRUD (all)     |
| Slots               | Read available | CRUD (own loc) | CRUD (all)     |
| Payments            | Own only       | View (own loc) | All            |
| Reports             | ❌             | Own locations  | All            |
| User management     | ❌             | ❌             | ✅             |
| System config       | ❌             | ❌             | ✅             |

## Transport Security

### TLS Termination

```
┌────────┐  HTTPS  ┌─────────┐  HTTP (internal)  ┌──────────┐
│ Client │────────▶│ Traefik │───────────────────▶│ Services │
└────────┘         └─────────┘                    └──────────┘
                   TLS 1.2+
                   Let's Encrypt
```

| Aspect              | Configuration                              |
|---------------------|--------------------------------------------|
| TLS version         | Minimum TLS 1.2                            |
| Certificate         | Let's Encrypt (auto-renewal via Traefik)   |
| Cipher suites       | Modern (ECDHE+AESGCM, CHACHA20)           |
| HSTS                | Enabled, max-age=31536000, includeSubDomains|
| Internal traffic    | Plain HTTP within Docker network (trusted) |

### mTLS Between Services (Future)

- Planned for production deployment on GCP
- Service mesh (Istio/Linkerd) for automatic mTLS
- Certificate rotation via cert-manager
- Zero-trust network model

## Data Protection

### Encryption at Rest

| Data Store   | Mechanism                                    |
|--------------|----------------------------------------------|
| PostgreSQL   | Transparent Data Encryption (TDE) via OS-level disk encryption |
| Redis        | Not encrypted at rest (ephemeral cache data) |
| Backups      | AES-256 encrypted before storage             |
| Logs         | Stored on encrypted volumes                  |

### Encryption in Transit

| Path                    | Protocol    | Notes                          |
|-------------------------|-------------|--------------------------------|
| Client → Traefik       | TLS 1.2+   | Public internet                |
| Traefik → Gateway      | HTTP        | Internal Docker network        |
| Gateway → Services     | gRPC (H2C) | Internal Docker network        |
| Services → PostgreSQL  | TLS         | If external DB                 |
| Services → Redis       | Plain       | Internal network, AUTH enabled |
| Services → NATS        | Plain       | Internal network               |

### Sensitive Data Handling

| Data Type        | Storage          | Access Control        | Retention     |
|------------------|------------------|-----------------------|---------------|
| Plate numbers    | PostgreSQL       | Service-level only    | Active + 90d  |
| Payment tokens   | Not stored       | Pass-through to PSP   | Never stored  |
| User PII         | PostgreSQL       | Encrypted columns     | Account life  |
| Session logs     | Loki             | Admin only            | 30 days       |
| Audit trail      | PostgreSQL       | Append-only, admin    | 1 year        |

## Input Validation

### Protobuf Schema Validation

gRPC services use protobuf definitions as the first line of input validation:

- Required fields enforced by proto3 semantics
- Field types provide basic type safety
- Enum values restrict to valid options

### Custom Validators

Additional validation at the handler/usecase layer:

```go
// Validation rules for check-in request
type CheckInRequest struct {
    LocationID  string `validate:"required,uuid"`
    PlateNumber string `validate:"required,plate_number"`
    VehicleType string `validate:"required,oneof=car motorcycle truck"`
}
```

| Validation Type     | Implementation                          |
|---------------------|-----------------------------------------|
| Struct validation   | `go-playground/validator`               |
| Plate number format | Custom regex validator                  |
| UUID format         | Standard UUID validation                |
| Business rules      | Usecase layer checks                    |
| SQL injection       | Parameterized queries (pgx)             |
| XSS                 | No HTML rendering (API-only)            |

## Rate Limiting

### Strategy: Token Bucket (Redis-Backed)

```
┌────────┐     ┌─────────┐     ┌───────┐
│ Client │────▶│ Gateway │────▶│ Redis │
└────────┘     │ (check) │     │(bucket)│
               └─────────┘     └───────┘
```

### Rate Limit Tiers

| Tier          | Scope    | Limit              | Window  | Burst |
|---------------|----------|--------------------|---------|-------|
| Anonymous     | Per-IP   | 60 requests        | 1 min   | 10    |
| Authenticated | Per-user | 300 requests       | 1 min   | 50    |
| Check-in      | Per-user | 5 requests         | 1 min   | 2     |
| Admin API     | Per-user | 600 requests       | 1 min   | 100   |

### Implementation

- Redis-based sliding window counter
- `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` headers
- 429 Too Many Requests response with `Retry-After` header
- Distributed across gateway instances via shared Redis

## Secrets Management

### Current (VPS Deployment)

| Secret Type          | Storage                    | Access                    |
|----------------------|----------------------------|---------------------------|
| Database credentials | Environment variables      | Container runtime only    |
| JWT signing keys     | External IdP (JWKS)        | Fetched at runtime        |
| API keys             | GitHub Secrets → CI env    | Injected during deploy    |
| Redis password       | Environment variables      | Container runtime only    |
| NATS credentials     | Environment variables      | Container runtime only    |

### Principles

- No secrets in source code or Docker images
- Secrets injected at runtime via environment variables
- GitHub Secrets for CI/CD pipeline credentials
- Coolify manages deployment secrets
- Secret rotation: manual process with documented runbook

### Future (GCP Deployment)

- Google Secret Manager for all secrets
- Workload Identity for service authentication
- Automatic secret rotation
- Audit logging for secret access

## Dependency Scanning

| Tool          | What It Scans                    | Frequency     | Action on Finding    |
|---------------|----------------------------------|---------------|----------------------|
| `govulncheck` | Go module vulnerabilities       | Every PR      | Block merge          |
| Trivy         | Container image CVEs            | Every build   | Block deploy (HIGH+) |
| Dependabot    | Outdated dependencies           | Daily         | Auto-PR              |
| `gitleaks`    | Secrets in git history          | Every PR      | Block merge          |

### Vulnerability Response SLA

| Severity  | Response Time | Fix Time   |
|-----------|---------------|------------|
| Critical  | 4 hours       | 24 hours   |
| High      | 24 hours      | 72 hours   |
| Medium    | 1 week        | 2 weeks    |
| Low       | 2 weeks       | Next sprint|

## OWASP Top 10 Coverage

| # | Risk                              | Mitigation                                          | Status |
|---|-----------------------------------|-----------------------------------------------------|--------|
| 1 | Broken Access Control             | RBAC via JWT claims, row-level filtering            | ✅     |
| 2 | Cryptographic Failures            | TLS 1.2+, no custom crypto, encrypted backups       | ✅     |
| 3 | Injection                         | Parameterized queries (pgx), protobuf schemas       | ✅     |
| 4 | Insecure Design                   | Threat modeling, security ADRs, code review         | ✅     |
| 5 | Security Misconfiguration         | Hardened containers, minimal base images, no defaults| ✅     |
| 6 | Vulnerable Components             | govulncheck, Trivy, Dependabot                      | ✅     |
| 7 | Auth Failures                     | External IdP, JWT validation, rate limiting         | ✅     |
| 8 | Software/Data Integrity Failures  | Signed container images, verified dependencies      | 🔄     |
| 9 | Logging & Monitoring Failures     | OpenTelemetry, structured logging, alerting         | ✅     |
|10 | SSRF                              | No user-controlled URLs, network policies           | ✅     |

## Idempotency Keys

### Purpose

Prevent replay attacks and duplicate operations (especially for payments and check-in/check-out).

### Implementation

```go
// Idempotency middleware
func IdempotencyMiddleware(cache redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.GetHeader("X-Idempotency-Key")
        if key == "" {
            c.Next()
            return
        }

        // Check if request was already processed
        cached, err := cache.Get(ctx, "idempotency:"+key).Result()
        if err == nil {
            // Return cached response
            c.Data(200, "application/json", []byte(cached))
            c.Abort()
            return
        }

        c.Next()

        // Cache successful response
        if c.Writer.Status() < 400 {
            cache.Set(ctx, "idempotency:"+key, responseBody, 24*time.Hour)
        }
    }
}
```

### Scope

| Operation    | Idempotency Key Source     | TTL     |
|--------------|----------------------------|---------|
| Check-in     | Client-generated UUID      | 24h     |
| Check-out    | Session ID + timestamp     | 24h     |
| Payment      | Payment reference ID       | 7 days  |
| Slot update  | Request UUID               | 1h      |

## Distributed Locking

### Purpose

Prevent race conditions in concurrent operations (slot booking, payment processing).

### Implementation (Redis-based)

```go
// Distributed lock for slot reservation
func (uc *ParkingUsecase) CheckIn(ctx context.Context, req CheckInRequest) (*Session, error) {
    lockKey := fmt.Sprintf("lock:slot:%s:%s", req.LocationID, req.SlotID)

    lock, err := uc.locker.Acquire(ctx, lockKey, 30*time.Second)
    if err != nil {
        return nil, ErrSlotUnavailable
    }
    defer lock.Release(ctx)

    // Critical section: verify and reserve slot
    slot, err := uc.repo.GetSlot(ctx, req.SlotID)
    if err != nil || slot.Status != SlotAvailable {
        return nil, ErrSlotUnavailable
    }

    return uc.repo.CreateSession(ctx, req)
}
```

### Lock Patterns

| Operation          | Lock Key Pattern                  | TTL  | Retry |
|--------------------|-----------------------------------|------|-------|
| Slot reservation   | `lock:slot:{location}:{slot}`    | 30s  | 3x    |
| Payment processing | `lock:payment:{session_id}`      | 60s  | No    |
| Slot status update | `lock:slot-status:{slot_id}`     | 10s  | 3x    |
| Report generation  | `lock:report:{type}:{date}`      | 300s | No    |

### Safeguards

- All locks have TTL to prevent deadlocks
- Lock extension for long-running operations
- Fencing tokens to prevent stale lock holders from writing
- Monitoring for lock contention metrics

## Security Incident Response

For detailed incident response procedures, see [Incident Response Runbook](../incident-response/runbook.md).

### Quick Reference

| Severity | Examples                              | Response Time | Escalation       |
|----------|---------------------------------------|---------------|------------------|
| P1       | Data breach, auth bypass              | 15 min        | Immediate        |
| P2       | Vulnerability exploited, DoS          | 1 hour        | Within 2 hours   |
| P3       | Suspicious activity, failed attacks   | 4 hours       | Next business day|
| P4       | Policy violation, minor misconfiguration | 24 hours   | Weekly review    |

### Security Monitoring

- Failed authentication attempts → alert after 10 in 5 minutes
- Rate limit breaches → logged and monitored
- Unusual API patterns → anomaly detection (future)
- Dependency vulnerability alerts → immediate triage

---

*Last updated: 2026-05-13*
*Owner: ParkirPintar Engineering*
*Review cycle: Quarterly*
