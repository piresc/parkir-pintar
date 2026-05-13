# Scalability Specification

**Project:** ParkirPintar — Smart Parking Backend System  
**Version:** 1.0  
**Date:** 2026-05-13  
**Status:** Approved

---

## 1. Current Capacity Assessment

### 1.1 Infrastructure Baseline

| Component | Current Setup | Capacity |
|-----------|--------------|----------|
| Deployment | Single server, Docker Compose | 1 node |
| Services | 7 Go microservices (stateless) | 1 instance each |
| PostgreSQL | Single instance, container | Max 25 connections per service |
| Redis | Single instance, container | 64 MB memory |
| NATS JetStream | Single instance, file storage | 10,000 msg/s theoretical |
| Reverse Proxy | Traefik (single instance) | ~5,000 req/s |

### 1.2 Estimated Current Capacity

| Metric | Estimated Value | Bottleneck |
|--------|----------------|------------|
| Concurrent users | ~100-200 | PostgreSQL connection pool |
| Requests per second (gateway) | ~50 req/s | Single gateway instance |
| Reservation throughput | ~20 reservations/s | Distributed lock contention |
| Search queries | ~200 req/s (cached) | Redis single instance |
| Event throughput (NATS) | ~5,000 msg/s | Single NATS node |
| Storage growth | ~500 MB/month | Single disk |
| Parking capacity | 400 spots (5 floors) | Fixed physical constraint |

### 1.3 Current Resource Utilization (Estimated)

| Service | CPU | Memory | Network |
|---------|-----|--------|---------|
| Gateway | 0.1 cores | 48 MB | 500 MB/day |
| Search | 0.05 cores | 64 MB | 200 MB/day |
| Reservation | 0.15 cores | 96 MB | 300 MB/day |
| Billing | 0.05 cores | 48 MB | 100 MB/day |
| Payment | 0.08 cores | 64 MB | 150 MB/day |
| Presence | 0.1 cores | 72 MB | 400 MB/day |
| Notification | 0.02 cores | 32 MB | 100 MB/day |
| **Total** | **0.55 cores** | **424 MB** | **1.75 GB/day** |

---

## 2. Target Capacity

### 2.1 Target Metrics (Phase 2)

| Metric | Target | Rationale |
|--------|--------|-----------|
| Concurrent users | 1,000 | Peak hour capacity for 400-spot facility |
| Requests per second | 100 req/s sustained, 300 req/s burst | 10x current with headroom |
| Reservation throughput | 50 reservations/s | Support rapid turnover periods |
| Search queries | 500 req/s | High-frequency availability checks |
| Event throughput | 10,000 msg/s | 2x current for safety margin |
| System availability | 99.9% | < 8.76 hours downtime/year |
| API latency P95 | < 200ms | User experience requirement |
| API latency P99 | < 500ms | Tail latency control |

### 2.2 Growth Projections

| Timeframe | Users | Reservations/day | Storage | Network |
|-----------|-------|-----------------|---------|---------|
| Current | 200 | 500 | 500 MB/mo | 1.75 GB/day |
| 3 months | 500 | 1,500 | 1.5 GB/mo | 5 GB/day |
| 6 months | 1,000 | 3,000 | 3 GB/mo | 10 GB/day |
| 12 months | 2,500 | 8,000 | 8 GB/mo | 25 GB/day |

---

## 3. Horizontal Scaling Strategy per Service

### 3.1 Service Scaling Characteristics

| Service | Stateless? | Scaling Constraint | Min Replicas | Max Replicas | Scale Factor |
|---------|-----------|-------------------|--------------|--------------|--------------|
| Gateway | ✅ Yes | CPU-bound (TLS, JWT validation) | 2 | 10 | CPU > 70% |
| Search | ✅ Yes | Memory-bound (cache), I/O (Redis) | 2 | 5 | CPU > 60%, RPS > 200 |
| Reservation | ✅ Yes | Lock contention (Redis), DB writes | 2 | 5 | CPU > 70%, queue depth |
| Billing | ✅ Yes | CPU-bound (fee calculation) | 2 | 3 | CPU > 70% |
| Payment | ✅ Yes | I/O-bound (external gateway) | 2 | 5 | Pending payments > 50 |
| Presence | ✅ Yes | Network-bound (streaming) | 2 | 3 | Active streams > 500 |
| Notification | ✅ Yes | I/O-bound (external providers) | 1 | 2 | Queue depth > 1000 |

### 3.2 Scaling Prerequisites

All services are already stateless (no in-memory state between requests), enabling horizontal scaling with:

1. **gRPC client-side load balancing** — Round-robin across service replicas via Kubernetes Service or DNS
2. **Shared Redis** — All instances connect to same Redis for locks and cache
3. **Shared PostgreSQL** — Connection pool per instance (25 × N replicas = total connections)
4. **NATS consumer groups** — JetStream queue groups distribute messages across instances
5. **Idempotency** — All write operations are idempotent, safe for retry across replicas

---

## 4. Database Scaling

### 4.1 Connection Pooling

| Parameter | Current | Target | Strategy |
|-----------|---------|--------|----------|
| Max connections per service | 25 | 25 | Maintain per-instance limit |
| Total connections (7 services × 1) | 175 | 175 per replica set | PgBouncer for connection multiplexing |
| Idle connection timeout | 5 min | 5 min | Reclaim unused connections |
| Connection lifetime | 1 hour | 1 hour | Prevent stale connections |

**Phase 2: PgBouncer**
```
Client (25 conns) → PgBouncer (transaction pooling) → PostgreSQL (100 actual conns)
```

### 4.2 Read Replicas

| Service | Read/Write Ratio | Replica Candidate? | Strategy |
|---------|-----------------|-------------------|----------|
| Search | 99% read / 1% write | ✅ Primary candidate | Route all reads to replica |
| Reservation | 40% read / 60% write | ⚠️ Partial | `ListByDriver` to replica, writes to primary |
| Billing | 30% read / 70% write | ❌ Not initially | Writes dominate |
| Payment | 50% read / 50% write | ⚠️ Partial | `GetPaymentStatus` to replica |
| Presence | 20% read / 80% write | ❌ Not initially | Write-heavy streaming |

**Replication topology:**
```
Primary (writes) ──── Streaming Replication ────► Read Replica 1 (Search, read queries)
                                                 ► Read Replica 2 (Reservation reads, Payment reads)
```

### 4.3 Partitioning Strategy (Phase 3)

| Table | Partition Key | Strategy | Trigger |
|-------|--------------|----------|---------|
| `reservations` | `created_at` (monthly) | Range partitioning | > 1M rows |
| `billing_records` | `created_at` (monthly) | Range partitioning | > 1M rows |
| `payments` | `created_at` (monthly) | Range partitioning | > 500K rows |
| `presence_logs` | `recorded_at` (daily) | Range partitioning + TTL | > 10M rows |
| `parking_spots` | None | No partitioning needed | 400 rows (static) |

### 4.4 Schema-per-Service Isolation

Current state: shared PostgreSQL instance with logical schema boundaries.

| Service | Schema | Tables Owned |
|---------|--------|-------------|
| Reservation | `reservation` | `reservations`, `parking_spots` |
| Billing | `billing` | `billing_records`, `penalties` |
| Payment | `payment` | `payments` |
| Presence | `presence` | `presence_logs` |
| Search | `search` | Read-only view of `parking_spots` |
| Analytics | `analytics` | `analytics_events` |

---

## 5. Cache Scaling (Redis)

### 5.1 Current Usage

| Use Case | Service | Key Pattern | TTL | Memory |
|----------|---------|-------------|-----|--------|
| Availability cache | Search | `avail:{vehicle_type}` | 30s | ~1 KB |
| Floor map cache | Search | `floor:{number}` | 30s | ~10 KB |
| Distributed lock | Reservation | `lock:spot:{spot_id}` | 30s | ~100 B |
| Rate limiting | Gateway | `rate:{ip}` | 60s | ~100 B |
| Idempotency | All | `idem:{key}` | 15min | ~500 B |

### 5.2 Scaling Strategy

| Phase | Topology | Capacity | Failover |
|-------|----------|----------|----------|
| Phase 1 (current) | Single Redis instance | 64 MB | None (restart) |
| Phase 2 | Redis Sentinel (1 primary + 2 replicas) | 256 MB | Automatic failover (< 30s) |
| Phase 3 | Redis Cluster (3 masters + 3 replicas) | 1 GB+ | Automatic failover + sharding |

**Phase 2: Redis Sentinel Configuration**
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Sentinel 1  │     │  Sentinel 2  │     │  Sentinel 3  │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                     │                     │
       ▼                     ▼                     ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Redis Primary│────▶│ Redis Replica│     │ Redis Replica│
│   (writes)   │     │   (reads)    │     │   (reads)    │
└──────────────┘     └──────────────┘     └──────────────┘
```

**Key considerations:**
- Distributed locks (SETNX) must always go to primary
- Cache reads can be distributed to replicas
- Rate limiting counters must go to primary (consistency)

---

## 6. Message Queue Scaling (NATS)

### 6.1 Current Configuration

| Parameter | Value |
|-----------|-------|
| Topology | Single NATS server with JetStream |
| Streams | 4 (RESERVATIONS, BILLING, PAYMENTS, PRESENCE) |
| Storage | File-based |
| Retention | Limits-based, 72h max age |
| Delivery | At-least-once with ACK |

### 6.2 NATS Cluster Scaling

| Phase | Topology | Replication | Throughput |
|-------|----------|-------------|------------|
| Phase 1 (current) | Single node | None | 10K msg/s |
| Phase 2 | 3-node cluster | R=3 (all streams) | 30K msg/s |
| Phase 3 | 5-node cluster | R=3 (configurable) | 50K+ msg/s |

**Phase 2: NATS Cluster Configuration**
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   NATS-1    │◄───▶│   NATS-2    │◄───▶│   NATS-3    │
│ (JetStream) │     │ (JetStream) │     │ (JetStream) │
│   Leader    │     │  Follower   │     │  Follower   │
└─────────────┘     └─────────────┘     └─────────────┘
```

**Consumer scaling:**
- Queue groups distribute messages across service replicas
- Each service replica joins the same consumer group
- NATS delivers each message to exactly one consumer in the group

---

## 7. Load Balancing Strategy

### 7.1 External Load Balancing (Client → Gateway)

| Layer | Component | Strategy | Health Check |
|-------|-----------|----------|--------------|
| L7 (HTTP) | Traefik | Round-robin with health checks | `GET /health` every 10s |
| L4 (TCP) | Cloud LB (production) | Least connections | TCP connect |

### 7.2 Internal Load Balancing (Service → Service)

| Pattern | Implementation | Use Case |
|---------|---------------|----------|
| gRPC client-side LB | `grpc.WithDefaultServiceConfig(round_robin)` | All gRPC calls |
| DNS-based discovery | Kubernetes Service / Docker DNS | Service endpoint resolution |
| NATS queue groups | Built-in consumer distribution | Event processing |

### 7.3 Database Load Balancing

| Query Type | Target | Strategy |
|-----------|--------|----------|
| Writes (INSERT, UPDATE, DELETE) | Primary | Direct connection |
| Reads (SELECT) | Read replica | Connection string routing |
| Transactions | Primary | Always primary for ACID |
| Analytics queries | Read replica | Offload from primary |

---

## 8. Auto-Scaling Triggers and Thresholds

### 8.1 Horizontal Pod Autoscaler (HPA) Configuration

| Service | Metric | Scale-Up Threshold | Scale-Down Threshold | Cooldown |
|---------|--------|-------------------|---------------------|----------|
| Gateway | CPU utilization | > 70% for 60s | < 30% for 300s | 120s |
| Gateway | Request rate | > 80 req/s/pod | < 20 req/s/pod | 120s |
| Search | CPU utilization | > 60% for 60s | < 25% for 300s | 120s |
| Reservation | CPU utilization | > 70% for 60s | < 30% for 300s | 180s |
| Reservation | Queue depth (pending) | > 50 pending | < 10 pending | 180s |
| Payment | Pending payments | > 50 in-flight | < 10 in-flight | 180s |
| Presence | Active streams | > 500/pod | < 100/pod | 120s |
| Notification | NATS consumer lag | > 1000 messages | < 100 messages | 60s |

### 8.2 Vertical Scaling Triggers

| Component | Metric | Trigger | Action |
|-----------|--------|---------|--------|
| PostgreSQL | CPU > 80% sustained | 15 min | Upgrade instance class |
| PostgreSQL | Storage > 80% | — | Expand volume |
| PostgreSQL | Connection saturation > 90% | 5 min | Add PgBouncer or increase max_connections |
| Redis | Memory > 80% | — | Upgrade instance or enable eviction |
| NATS | Disk > 70% | — | Expand storage or reduce retention |

### 8.3 Alert-Based Scaling (Manual Triggers)

| Alert | Condition | Recommended Action |
|-------|-----------|-------------------|
| `HighLatency` | P95 > 500ms for 5 min | Scale up affected service |
| `HighErrorRate` | Error rate > 5% for 3 min | Investigate + scale if resource-bound |
| `DatabaseSlowQueries` | Slow queries > 10/min | Add read replica or optimize queries |
| `NATSConsumerLag` | Lag > 5000 messages | Scale up consumer service |

---

## 9. Bottleneck Analysis

### 9.1 Current Bottlenecks (Priority Order)

| # | Bottleneck | Component | Impact | Severity |
|---|-----------|-----------|--------|----------|
| 1 | **Single PostgreSQL instance** | Database | All services share one DB; connection pool exhaustion under load | Critical |
| 2 | **Single Redis instance** | Cache/Locks | No failover; lock contention under concurrent reservations | High |
| 3 | **Single Gateway instance** | Ingress | Single point of failure for all client traffic | High |
| 4 | **Distributed lock contention** | Reservation | Redis SETNX serializes concurrent reservations for same spot | Medium |
| 5 | **Single NATS instance** | Messaging | No replication; message loss on crash | Medium |
| 6 | **Synchronous payment calls** | Payment | External gateway latency (up to 3s) blocks reservation flow | Medium |
| 7 | **No connection pooling proxy** | Database | Each service instance opens 25 connections directly | Low |

### 9.2 Bottleneck Resolution Plan

| # | Bottleneck | Resolution | Phase | Effort |
|---|-----------|-----------|-------|--------|
| 1 | Single PostgreSQL | Add read replicas + PgBouncer | Phase 2 | 3 days |
| 2 | Single Redis | Deploy Redis Sentinel (3 nodes) | Phase 2 | 2 days |
| 3 | Single Gateway | Run 2+ Gateway replicas behind Traefik | Phase 2 | 1 day |
| 4 | Lock contention | Optimize lock granularity; reduce hold time | Phase 2 | 2 days |
| 5 | Single NATS | Deploy 3-node NATS cluster | Phase 2 | 2 days |
| 6 | Sync payment | Async payment with webhook callback (future) | Phase 3 | 5 days |
| 7 | No connection proxy | Deploy PgBouncer in transaction mode | Phase 2 | 1 day |

---

## 10. Scaling Roadmap

### Phase 1: Vertical Scaling (Current → Month 3)

**Goal:** Maximize single-node capacity before adding complexity.

| Action | Component | Expected Gain | Effort |
|--------|-----------|---------------|--------|
| Increase PostgreSQL `max_connections` to 200 | Database | +75 connections | 1 hour |
| Increase Redis `maxmemory` to 256 MB | Cache | 4x cache capacity | 1 hour |
| Optimize slow queries (add indexes) | Database | -30% query latency | 2 days |
| Enable PostgreSQL query plan caching | Database | -15% CPU | 1 hour |
| Increase NATS file storage to 10 GB | Messaging | 10x message retention | 1 hour |
| Tune Go GC (`GOGC=200`) | All services | -20% GC pauses | 1 hour |

**Expected capacity after Phase 1:** 300 concurrent users, 80 req/s

---

### Phase 2: Horizontal Scaling (Month 3 → Month 6)

**Goal:** Eliminate single points of failure; support 1000 concurrent users.

| Action | Component | Expected Gain | Effort |
|--------|-----------|---------------|--------|
| Deploy 2+ Gateway replicas | Gateway | 2x throughput, HA | 1 day |
| Deploy 2+ Reservation replicas | Reservation | 2x write throughput | 1 day |
| Deploy PgBouncer (transaction pooling) | Database | 4x effective connections | 2 days |
| Add PostgreSQL read replica | Database | Offload reads, HA | 3 days |
| Deploy Redis Sentinel (3 nodes) | Cache | HA, read distribution | 2 days |
| Deploy NATS cluster (3 nodes) | Messaging | HA, 3x throughput | 2 days |
| Implement NATS queue groups | All consumers | Distributed processing | 1 day |
| Add Kubernetes HPA | All services | Auto-scaling | 2 days |
| Migrate to Kubernetes | Infrastructure | Orchestration, self-healing | 5 days |

**Expected capacity after Phase 2:** 1,000 concurrent users, 100 req/s

**Architecture after Phase 2:**
```
                    ┌─────────────┐
                    │  Cloud LB   │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │Gateway-1 │ │Gateway-2 │ │Gateway-3 │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │             │             │
             └─────────────┼─────────────┘
                           │ gRPC (client-side LB)
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │ Service  │ │ Service  │ │ Service  │
        │Replicas  │ │Replicas  │ │Replicas  │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │             │             │
             └─────────────┼─────────────┘
                           │
    ┌──────────────────────┼──────────────────────┐
    ▼                      ▼                      ▼
┌────────────┐    ┌──────────────┐    ┌───────────────┐
│ PgBouncer  │    │Redis Sentinel│    │ NATS Cluster  │
│     ↓      │    │ (3 nodes)    │    │  (3 nodes)    │
│ PG Primary │    └──────────────┘    └───────────────┘
│ PG Replica │
└────────────┘
```

---

### Phase 3: Multi-Region / Multi-Facility (Month 6 → Month 12)

**Goal:** Support multiple parking facilities; geographic distribution.

| Action | Component | Expected Gain | Effort |
|--------|-----------|---------------|--------|
| Multi-tenant schema (facility_id) | Database | Multi-facility support | 5 days |
| PostgreSQL Aurora/CockroachDB | Database | Multi-AZ, auto-failover | 5 days |
| Redis Cluster (sharded) | Cache | Horizontal cache scaling | 3 days |
| NATS super-cluster (multi-region) | Messaging | Cross-region events | 3 days |
| CDN for static assets | Frontend | Global latency reduction | 1 day |
| Database partitioning (by facility) | Database | Query isolation | 3 days |
| Service mesh (Istio/Linkerd) | Network | mTLS, observability, traffic control | 5 days |
| Multi-region deployment | Infrastructure | Geographic redundancy | 10 days |

**Expected capacity after Phase 3:** 10,000+ concurrent users, 1000 req/s, multi-facility

---

## 11. Capacity Planning Formulas

### 11.1 Service Replica Calculation

```
Required replicas = ceil(target_rps / rps_per_instance)

Gateway:     ceil(100 / 50) = 2 replicas minimum
Search:      ceil(500 / 200) = 3 replicas (with Redis cache)
Reservation: ceil(50 / 20) = 3 replicas
Billing:     ceil(50 / 30) = 2 replicas
Payment:     ceil(50 / 15) = 4 replicas (external I/O bound)
Presence:    ceil(1000 streams / 500) = 2 replicas
Notification: ceil(1000 events/min / 1000) = 1 replica
```

### 11.2 Database Connection Calculation

```
Total connections = (services × replicas × pool_size) + overhead
                  = (7 × 3 × 25) + 25 (admin/monitoring)
                  = 550 connections

With PgBouncer (transaction mode, 4:1 multiplexing):
Actual PG connections = 550 / 4 = ~140 connections
```

### 11.3 Storage Growth Calculation

```
Monthly storage = (reservations/day × 30 × avg_row_size) + (presence_logs × 30 × avg_row_size)
                = (3000 × 30 × 500B) + (50000 × 30 × 200B)
                = 45 MB + 300 MB
                = ~345 MB/month (data only, excluding indexes and WAL)
```

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-05-13 | Engineering Team | Initial scalability specification |
