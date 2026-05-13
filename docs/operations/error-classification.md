# Error Classification System — ParkirPintar

## Error Severity Levels

| Level | Name | Description | Example |
|-------|------|-------------|---------|
| **P0** | Critical / Data Loss | Data corruption, data loss, or security breach. All users affected. | Database corruption, payment double-charge, auth bypass |
| **P1** | Service Down | One or more services completely unavailable. Core functionality broken. | Reservation service crash loop, database unreachable |
| **P2** | Degraded | Service partially functional. Some users or features affected. | High latency (>2s), intermittent timeouts, notification delays |
| **P3** | Minor | Non-critical functionality impaired. Workaround available. | Dashboard slow, non-critical background job failing |
| **P4** | Cosmetic | Visual or UX issues. No functional impact. | Misaligned UI element, typo in notification text |

## Error Categories

### Infrastructure

- Server/VM failures
- Network connectivity issues
- DNS resolution failures
- Disk space exhaustion
- Memory/CPU saturation
- Docker container OOM kills

### Application

- Unhandled panics
- Business logic errors
- Configuration errors
- Memory leaks
- Goroutine leaks
- Race conditions

### Integration

- gRPC connection failures between services
- NATS publish/subscribe failures
- Third-party API failures (payment gateway, SMS provider)
- gRPC deadline exceeded
- Circuit breaker open

### Data

- Database deadlocks
- Data inconsistency between services
- Migration failures
- Backup corruption
- Cache poisoning (stale Redis data)

### Security

- Authentication failures (brute force)
- Authorization bypass attempts
- Rate limit violations
- Suspicious API patterns
- Certificate expiration

## Response Time SLAs

| Severity | Acknowledge | First Response | Resolution Target |
|----------|-------------|----------------|-------------------|
| **P0** | 5 min | 15 min | 2 hours |
| **P1** | 15 min | 1 hour | 4 hours |
| **P2** | 30 min | 4 hours | 24 hours |
| **P3** | 4 hours | 24 hours | Next sprint |
| **P4** | Next business day | Next sprint | Backlog |

*Times are from alert trigger, not from user report.*

## Escalation Matrix

```
P0: On-call engineer → Team lead → CTO (within 15 min if no progress)
P1: On-call engineer → Team lead (within 1h if no progress)
P2: On-call engineer → Discussed in daily standup
P3: Assigned in sprint planning
P4: Added to backlog, prioritized quarterly
```

### Escalation Triggers

- No acknowledgment within SLA → auto-escalate to next level
- No progress after 50% of resolution target → escalate
- Customer-facing impact confirmed → bump severity by one level
- Data loss confirmed → immediately escalate to P0

## Error Trending and Reporting

### Real-Time

- Grafana dashboards: error rate per service, error rate by category
- Alert rules fire on threshold breach (see Prometheus alert rules)
- NATS dead letter queue monitoring

### Daily

- Automated error summary posted to Telegram dev channel
- Top 5 errors by frequency
- New error types (first seen in last 24h)

### Weekly

- Error trend report: week-over-week comparison
- SLA compliance percentage
- Mean Time to Detect (MTTD) and Mean Time to Resolve (MTTR)

### Monthly

- Error category distribution
- Recurring issues identification
- Capacity planning based on error patterns
- Post-mortem backlog review

## Root Cause Analysis Process

### 5 Whys Method

Used for P0 and P1 incidents:

```
Problem: Reservation service returned 500 errors for 10 minutes

Why 1: The service couldn't connect to PostgreSQL
Why 2: PostgreSQL connection pool was exhausted
Why 3: A slow query was holding connections for >30s
Why 4: Missing index on reservations.user_id column
Why 5: Index was dropped during last migration (migration 023)

Root Cause: Migration 023 accidentally dropped the user_id index
Fix: Add index back, add migration tests that verify index existence
```

### Fishbone (Ishikawa) Diagram

Used for complex P0/P1 incidents with multiple contributing factors:

Categories to investigate:
- **People**: On-call response time, knowledge gaps
- **Process**: Deployment procedure, review gaps
- **Technology**: Infrastructure limits, software bugs
- **Environment**: Load patterns, external dependencies

### Post-Mortem Template

See: [Post-Mortem Template](../templates/post-mortem-template.md)

Every P0 and P1 incident requires a post-mortem within 48 hours of resolution.

Post-mortem structure:
1. Incident summary
2. Timeline
3. Impact assessment
4. Root cause analysis
5. Contributing factors
6. Action items (with owners and deadlines)
7. Lessons learned

## Common Error Patterns in ParkirPintar

### Redis Timeout

```
Error: context deadline exceeded (redis)
Service: reservation-service, billing-service
Category: Infrastructure
Typical Severity: P2
```

**Symptoms:**
- Increased latency on reservation lookups
- Cache miss rate spikes
- `redis_command_duration_seconds` > 1s

**Common Causes:**
- Redis memory pressure (maxmemory reached, eviction happening)
- Network latency between service and Redis
- Large key operations (HGETALL on large hashes)
- Redis persistence (RDB save) causing latency spike

**Mitigation:**
1. Check Redis memory usage: `redis-cli info memory`
2. Check slow log: `redis-cli slowlog get 10`
3. If memory pressure: review eviction policy, increase maxmemory
4. If persistence: adjust save intervals

---

### NATS Disconnect

```
Error: nats: connection lost / nats: no responders
Service: all services (event-driven communication)
Category: Integration
Typical Severity: P1-P2
```

**Symptoms:**
- Events not being delivered
- Reservation confirmations delayed
- Billing events lost (if JetStream not configured)

**Common Causes:**
- NATS server restart/crash
- Network partition
- Client connection timeout (stale connection)
- Max payload exceeded

**Mitigation:**
1. Check NATS server status: `nats server check connection`
2. Verify JetStream consumers: `nats consumer ls PARKIR`
3. Check for pending messages: `nats stream info PARKIR`
4. Restart affected service to re-establish connection
5. Replay missed events from JetStream if needed

---

### Database Deadlock

```
Error: pq: deadlock detected
Service: reservation-service, billing-service
Category: Data
Typical Severity: P2-P3
```

**Symptoms:**
- Intermittent 500 errors on write operations
- Transaction rollbacks in logs
- Increased `pg_stat_activity` blocked queries

**Common Causes:**
- Concurrent reservation updates on same parking spot
- Billing reconciliation conflicting with new charges
- Missing row-level locking (SELECT FOR UPDATE)

**Mitigation:**
1. Check active locks: `SELECT * FROM pg_locks WHERE NOT granted;`
2. Identify blocking queries: `SELECT * FROM pg_stat_activity WHERE state = 'active';`
3. If persistent: review transaction isolation levels
4. Add retry logic with exponential backoff (already in place)
5. Consider advisory locks for hot-path operations

---

### gRPC Deadline Exceeded

```
Error: rpc error: code = DeadlineExceeded desc = context deadline exceeded
Service: inter-service calls
Category: Integration
Typical Severity: P2
```

**Symptoms:**
- Cascading timeouts across services
- Client-side errors on reservation creation (calls billing + notification)
- Increased error rate correlating with latency spike

**Common Causes:**
- Downstream service overloaded
- Database slow queries propagating up
- Network congestion
- Insufficient timeout configuration

**Mitigation:**
1. Identify which downstream call is slow (check Tempo traces)
2. Check downstream service health in Grafana
3. If DB-related: check `pg_stat_statements` for slow queries
4. Temporary: increase deadline in client config
5. Long-term: add circuit breaker, optimize slow path

## Error Code Standards

All ParkirPintar services use consistent error codes:

```go
// gRPC status codes mapping
codes.InvalidArgument  → 400 Bad Request
codes.Unauthenticated  → 401 Unauthorized
codes.PermissionDenied → 403 Forbidden
codes.NotFound         → 404 Not Found
codes.AlreadyExists    → 409 Conflict
codes.ResourceExhausted → 429 Too Many Requests
codes.Internal         → 500 Internal Server Error
codes.Unavailable      → 503 Service Unavailable
codes.DeadlineExceeded → 504 Gateway Timeout
```

### Application Error Codes

Format: `PARKIR-{SERVICE}-{NUMBER}`

```
PARKIR-RES-001: Spot already reserved
PARKIR-RES-002: Reservation expired
PARKIR-RES-003: Invalid time range
PARKIR-BIL-001: Payment method declined
PARKIR-BIL-002: Insufficient balance
PARKIR-BIL-003: Billing calculation error
PARKIR-AUTH-001: Invalid credentials
PARKIR-AUTH-002: Token expired
PARKIR-AUTH-003: Account locked
```
