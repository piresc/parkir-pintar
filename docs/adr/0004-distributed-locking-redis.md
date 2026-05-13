# 4. Redis SETNX for Distributed Locking on Spot Reservation

## Status

Accepted

## Context

The reservation-service must prevent double-booking: when multiple users attempt to reserve the same parking spot concurrently, only one should succeed. This is a classic distributed concurrency problem.

Requirements:
- Mutual exclusion: only one reservation can proceed for a given spot at a time
- Low latency: lock acquisition should not add significant overhead to the reservation flow
- Automatic expiry: locks must release even if the holding process crashes (no deadlocks)
- Simplicity: minimal infrastructure additions

Alternatives considered:

1. **PostgreSQL advisory locks** — no additional infrastructure, but tied to database connections, harder to manage TTL, and adds load to the database during high-concurrency spikes
2. **Redis SETNX with TTL** — fast, well-understood pattern, automatic expiry via TTL, Redis already planned for caching
3. **etcd/ZooKeeper** — stronger consistency guarantees, but significant operational overhead for a single use case

## Decision

We will use **Redis `SET key NX EX ttl`** (SETNX pattern) for distributed locking when processing spot reservations.

Lock key format: `lock:spot:{spot_id}`
TTL: 30 seconds (sufficient for reservation validation + write)
Unlock: explicit `DEL` on success, TTL expiry on failure/crash

The implementation will use a simple lock-try-release pattern without Redlock (single Redis instance is acceptable given our consistency requirements and the short lock duration).

## Consequences

### Positive

- Fast: Redis operations are sub-millisecond, negligible overhead on reservation flow
- Simple: well-understood pattern, easy to implement and reason about
- Automatic recovery: TTL ensures locks are released even on process crash
- No database contention: locking is offloaded from PostgreSQL
- Already in stack: Redis is used for caching, no additional infrastructure

### Negative

- Redis availability dependency: if Redis is down, reservations cannot proceed (mitigated by Redis Sentinel or managed Redis with HA)
- No fencing token: in rare cases of TTL expiry + delayed execution, a stale process could proceed (mitigated by optimistic concurrency check on DB write)
- Single point of failure without Redlock: acceptable trade-off given managed Redis with replication
- Memory overhead: minimal, locks are short-lived and auto-expire
