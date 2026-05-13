# Capacity Planning — ParkirPintar

> **Status:** Living Document  
> **Author:** piresc  
> **Date:** 2026-05-13  
> **Last Review:** 2026-05-13  
> **Next Review:** 2026-08-13 (quarterly)

---

## 1. Current Resource Allocation

### Staging Environment (Coolify — Shared VM: 4 vCPU, 32GB RAM)

| Service | CPU Limit | Memory Limit | Replicas | Notes |
|---------|-----------|--------------|----------|-------|
| Gateway | 0.5 vCPU | 256 MB | 1 | REST to gRPC routing |
| Reservation | 0.5 vCPU | 256 MB | 1 | Core booking logic |
| Billing | 0.25 vCPU | 128 MB | 1 | Fee calculation |
| Payment | 0.25 vCPU | 128 MB | 1 | Payment processing |
| Search | 0.5 vCPU | 256 MB | 1 | Availability queries |
| Presence | 0.25 vCPU | 128 MB | 1 | Sensor data ingestion |
| Notification | 0.25 vCPU | 128 MB | 1 | Event-driven alerts |
| PostgreSQL | 1.0 vCPU | 1 GB | 1 | Shared database (staging) |
| Redis | 0.25 vCPU | 256 MB | 1 | Cache + locks |
| NATS | 0.25 vCPU | 128 MB | 1 | JetStream messaging |
| Traefik | 0.25 vCPU | 128 MB | 1 | Reverse proxy + TLS |
| **Total** | **4.25 vCPU** | **2.85 GB** | — | Fits within VM capacity |

### Production Environment (GCP Cloud Run — Target)

| Service | CPU | Memory | Min Instances | Max Instances | Concurrency |
|---------|-----|--------|---------------|---------------|-------------|
| Gateway | 1 vCPU | 512 MB | 1 | 10 | 80 |
| Reservation | 1 vCPU | 512 MB | 1 | 5 | 50 |
| Billing | 0.5 vCPU | 256 MB | 0 | 3 | 50 |
| Payment | 0.5 vCPU | 256 MB | 0 | 3 | 30 |
| Search | 1 vCPU | 512 MB | 1 | 8 | 100 |
| Presence | 0.5 vCPU | 256 MB | 1 | 5 | 80 |
| Notification | 0.5 vCPU | 256 MB | 0 | 3 | 50 |

**Cloud SQL (PostgreSQL 14):** `db-custom-2-4096` (2 vCPU, 4 GB RAM, 50 GB SSD)  
**Memorystore (Redis):** Basic tier, 1 GB  
**NATS:** GKE pod or dedicated VM (2 vCPU, 2 GB RAM)

---

## 2. Growth Projections

### User Growth Model

| Metric | Month 1 | Month 3 | Month 6 | Month 12 |
|--------|---------|---------|---------|----------|
| Registered users | 100 | 500 | 2,000 | 10,000 |
| Daily active users | 30 | 150 | 600 | 3,000 |
| Concurrent users (peak) | 10 | 50 | 200 | 1,000 |
| Parking locations | 5 | 15 | 50 | 200 |
| Parking spots (total) | 500 | 1,500 | 5,000 | 20,000 |

### Request Volume Projections

| Metric | Month 1 | Month 3 | Month 6 | Month 12 |
|--------|---------|---------|---------|----------|
| Requests/day | 5,000 | 25,000 | 100,000 | 500,000 |
| Requests/sec (avg) | 0.06 | 0.3 | 1.2 | 6 |
| Requests/sec (peak) | 1 | 5 | 20 | 100 |
| Reservations/day | 50 | 250 | 1,000 | 5,000 |
| Payments/day | 40 | 200 | 800 | 4,000 |

### Request Distribution (estimated)

| Endpoint Category | % of Traffic | Peak RPS (Month 12) |
|-------------------|-------------|---------------------|
| Search/Availability | 45% | 45 |
| Reservations (CRUD) | 25% | 25 |
| Presence updates | 15% | 15 |
| Payments | 8% | 8 |
| Auth/Health/Other | 7% | 7 |

### Storage Growth

| Resource | Month 1 | Month 3 | Month 6 | Month 12 |
|----------|---------|---------|---------|----------|
| PostgreSQL data | 100 MB | 500 MB | 2 GB | 10 GB |
| PostgreSQL WAL | 500 MB | 1 GB | 2 GB | 5 GB |
| Redis memory | 50 MB | 100 MB | 200 MB | 500 MB |
| NATS JetStream | 100 MB | 250 MB | 500 MB | 1 GB |
| Logs (Cloud Logging) | 1 GB | 5 GB | 20 GB | 100 GB |
| Traces (Tempo) | 500 MB | 2 GB | 8 GB | 30 GB |

---

## 3. Scaling Triggers and Thresholds

### Cloud Run Auto-Scaling

| Metric | Scale-Up Threshold | Scale-Down Threshold | Cooldown |
|--------|-------------------|---------------------|----------|
| CPU utilization | > 70% for 60s | < 30% for 300s | 60s |
| Request concurrency | > 80% of max | < 20% of max | 60s |
| Request latency (p95) | > 500ms | < 100ms | 120s |

### Manual Scaling Triggers (alerts to human decision)

| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| Instance count | > 70% of max | = max | Increase max instances |
| DB connections | > 80% of pool | > 95% of pool | Increase pool or add read replica |
| Redis memory | > 70% of budget | > 85% of budget | Increase instance size or evict |
| Error rate (5xx) | > 1% | > 5% | Investigate + scale if load-related |
| NATS pending messages | > 10,000 | > 50,000 | Scale consumers |

### Scaling Decision Matrix

| Traffic Level | Gateway | Reservation | Search | Action |
|---------------|---------|-------------|--------|--------|
| < 10 RPS | 1 instance | 1 instance | 1 instance | Baseline |
| 10-50 RPS | 2-3 instances | 2 instances | 3-4 instances | Auto-scale |
| 50-100 RPS | 5-7 instances | 3-4 instances | 5-8 instances | Auto-scale |
| > 100 RPS | 10 instances | 5 instances | 8 instances | Review max limits |

---

## 4. Database Connection Pool Sizing

### Rationale

PostgreSQL max connections formula:
```
max_connections = (num_services x pool_size_per_service) + admin_overhead
```

### Current Configuration

| Service | Pool Size (max_open) | Idle Conns | Conn Lifetime | Justification |
|---------|---------------------|------------|---------------|---------------|
| Gateway | 5 | 2 | 5 min | Minimal direct DB access (auth only) |
| Reservation | 25 | 10 | 5 min | Highest write throughput, transactions |
| Billing | 10 | 5 | 5 min | Moderate writes (invoices) |
| Payment | 10 | 5 | 5 min | Moderate writes (transactions) |
| Search | 15 | 5 | 5 min | Read-heavy, benefits from pooling |
| Presence | 10 | 5 | 5 min | Frequent sensor updates |
| Notification | 5 | 2 | 5 min | Low DB usage (mostly NATS-driven) |
| **Total** | **80** | **34** | — | — |

### PostgreSQL Server Settings

```
# Cloud SQL (db-custom-2-4096)
max_connections = 100          # 80 app + 10 admin + 10 buffer
shared_buffers = 1GB           # 25% of 4GB RAM
effective_cache_size = 3GB     # 75% of RAM
work_mem = 16MB                # Per-sort operation
maintenance_work_mem = 256MB   # VACUUM, CREATE INDEX
```

### Scaling Pool Size

When scaling Cloud Run instances, each new instance opens its own pool:
```
Total connections = instances x pool_size_per_service
```

**Risk:** 5 Reservation instances x 25 conns = 125 conns > max_connections (100)

**Mitigation:**
1. Use Cloud SQL Auth Proxy with connection limits
2. Reduce per-instance pool size when instance count > 2:
   - 1-2 instances: 25 conns
   - 3-4 instances: 15 conns
   - 5+ instances: 10 conns
3. Consider PgBouncer sidecar for connection multiplexing at scale

### Connection Pool Health Metrics

```go
// Exposed via OpenTelemetry metrics
db.pool.open_connections      // Current open connections
db.pool.idle_connections      // Idle connections
db.pool.wait_count            // Total waits for connection
db.pool.wait_duration_total   // Total time waiting
db.pool.max_idle_closed       // Closed due to max idle
db.pool.max_lifetime_closed   // Closed due to max lifetime
```

---

## 5. Redis Memory Budget

### Total Budget: 1 GB (Memorystore Basic)

| Use Case | Estimated Size | TTL | Eviction Policy |
|----------|---------------|-----|-----------------|
| Session/JWT cache | 100 MB | 24h | volatile-lru |
| Distributed locks | 10 MB | 30s-5min | volatile-ttl |
| Rate limiter counters | 50 MB | 1min-1h | volatile-ttl |
| Search cache (availability) | 200 MB | 30s-5min | volatile-lru |
| Idempotency keys | 50 MB | 24h | volatile-ttl |
| Circuit breaker state | 5 MB | 30s-2min | volatile-ttl |
| Presence sensor cache | 100 MB | 10s | volatile-ttl |
| **Reserved headroom** | **485 MB** | — | — |

### Memory Policy

```
maxmemory 1gb
maxmemory-policy volatile-lru
```

**Why `volatile-lru`:** Only evicts keys with TTL set. All ParkirPintar keys have TTLs, so this safely evicts least-recently-used cached data without touching lock keys (which have short TTLs and expire naturally).

### Key Naming Convention

```
pp:{service}:{type}:{identifier}

pp:search:avail:location_123          # Availability cache
pp:reservation:lock:spot_456          # Distributed lock
pp:gateway:rate:ip_192.168.1.1        # Rate limit counter
pp:gateway:idemp:req_abc123           # Idempotency key
pp:presence:sensor:spot_456           # Sensor state
```

### Monitoring Thresholds

| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| used_memory | > 700 MB (70%) | > 850 MB (85%) | Reduce TTLs or upgrade |
| evicted_keys | > 100/min | > 1000/min | Upgrade instance size |
| connected_clients | > 80 | > 95 | Check for connection leaks |
| keyspace_misses ratio | > 50% | > 70% | Review cache strategy |

---

## 6. Cost Estimation — GCP Cloud Run

### Tier 1: Development/Demo (< 5 RPS)

| Service | Config | Monthly Cost |
|---------|--------|-------------|
| Cloud Run (7 services) | Min 0-1, Max 2 | $5-15 |
| Cloud SQL (PostgreSQL) | db-f1-micro, 10 GB | $10 |
| Memorystore (Redis) | Basic, 1 GB | $15 |
| NATS (GKE pod) | e2-small | $15 |
| Cloud DNS | 1 zone | $0.50 |
| Secret Manager | 6 secrets | $0.36 |
| Cloud Logging | 5 GB/mo | Free tier |
| **Total** | | **~$46-56/mo** |

### Tier 2: Early Production (5-50 RPS)

| Service | Config | Monthly Cost |
|---------|--------|-------------|
| Cloud Run (7 services) | Min 1, Max 5 | $30-80 |
| Cloud SQL (PostgreSQL) | db-custom-2-4096, 50 GB | $50 |
| Memorystore (Redis) | Basic, 1 GB | $15 |
| NATS (GKE pod) | e2-medium | $25 |
| Cloud DNS | 1 zone | $0.50 |
| Secret Manager | 6 secrets | $0.36 |
| Cloud Logging | 20 GB/mo | $10 |
| Cloud Trace | 5M spans | Free tier |
| Load Balancer | 1 rule | $18 |
| **Total** | | **~$150-200/mo** |

### Tier 3: Growth (50-100 RPS)

| Service | Config | Monthly Cost |
|---------|--------|-------------|
| Cloud Run (7 services) | Min 1-2, Max 10 | $100-250 |
| Cloud SQL (PostgreSQL) | db-custom-4-8192, 100 GB + read replica | $150 |
| Memorystore (Redis) | Standard, 2 GB (HA) | $60 |
| NATS (GKE cluster) | 3-node cluster | $75 |
| Cloud DNS | 1 zone | $0.50 |
| Secret Manager | 10 secrets | $0.60 |
| Cloud Logging | 100 GB/mo | $50 |
| Cloud Trace | 20M spans | $40 |
| Load Balancer | 1 rule + CDN | $30 |
| **Total** | | **~$500-650/mo** |

### Cost Optimization Strategies

1. **Min instances = 0** for low-traffic services (Billing, Payment, Notification) — cold start ~2s acceptable
2. **Committed use discounts** for Cloud SQL at Tier 3 (30% savings)
3. **Log exclusion filters** to reduce Cloud Logging costs (exclude debug logs in prod)
4. **Preemptible/Spot VMs** for NATS if running on GKE
5. **Cloud CDN** for static availability data (reduces Search service load)

---

## 7. Capacity Test Results

### Load Testing Framework

Tests run using [k6](https://k6.io/) with scenarios matching projected traffic patterns.

**Test scripts location:** `tests/load/`

```
tests/load/
├── scenarios/
│   ├── baseline.js          # Steady 10 RPS for 5 min
│   ├── peak-hour.js         # Ramp to 50 RPS over 10 min
│   ├── stress.js            # Ramp to 200 RPS until failure
│   ├── spike.js             # Sudden 0 to 100 RPS
│   └── soak.js              # 20 RPS for 2 hours
├── helpers/
│   ├── auth.js              # JWT token generation
│   └── data.js              # Test data factories
└── k6.config.js             # Shared thresholds and options
```

### Baseline Results (Staging — Single Instance)

| Scenario | VUs | RPS Achieved | p50 Latency | p95 Latency | p99 Latency | Error Rate |
|----------|-----|-------------|-------------|-------------|-------------|------------|
| Baseline | 10 | 10 | 12ms | 45ms | 120ms | 0% |
| Peak Hour | 50 | 48 | 25ms | 95ms | 250ms | 0.1% |
| Stress | 200 | 120 | 85ms | 450ms | 1200ms | 2.5% |
| Spike | 100 | 95 | 35ms | 180ms | 500ms | 0.5% |

### Bottleneck Analysis

| RPS Level | Bottleneck | Symptom | Resolution |
|-----------|-----------|---------|------------|
| > 80 RPS | DB connections | Connection wait time > 100ms | Increase pool or add PgBouncer |
| > 100 RPS | Redis latency | Lock contention on popular spots | Shard locks by location |
| > 150 RPS | CPU (Gateway) | p95 > 500ms | Scale to 2+ instances |
| > 200 RPS | NATS backpressure | Consumer lag > 5s | Scale notification consumers |

### SLO Compliance at Each Tier

| Tier | Target p95 | Achieved p95 | Target Availability | Achieved |
|------|-----------|-------------|--------------------|---------| 
| Tier 1 (< 5 RPS) | < 200ms | 45ms | 99.5% | 99.8% |
| Tier 2 (5-50 RPS) | < 200ms | 95ms | 99.9% | TBD |
| Tier 3 (50-100 RPS) | < 300ms | TBD | 99.9% | TBD |

---

## 8. Capacity Review Cadence

| Activity | Frequency | Owner |
|----------|-----------|-------|
| Load test execution | Monthly | Engineering |
| Cost review | Monthly | Engineering |
| Growth projection update | Quarterly | Product + Engineering |
| Scaling threshold review | Quarterly | Engineering |
| Database sizing review | Quarterly | Engineering |
| Full capacity plan update | Quarterly | Engineering |
