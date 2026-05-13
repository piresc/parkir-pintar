# 3. NATS JetStream for Asynchronous Event-Driven Communication

## Status

Accepted

## Context

Several workflows in ParkirPintar are inherently asynchronous and benefit from event-driven decoupling:

- Spot status changes (occupied → available) should trigger search index updates without presence-service knowing about search-service
- Reservation completion should trigger billing calculation without tight coupling
- Payment confirmation should update reservation status via events

Requirements:
- At-least-once delivery guarantees
- Durable subscriptions (consumers can catch up after downtime)
- Low operational overhead
- Support for subject-based routing

Alternatives considered:

1. **RabbitMQ** — mature, feature-rich, but heavier operationally, AMQP protocol complexity
2. **Apache Kafka** — excellent for high-throughput event streaming, but significant operational overhead (ZooKeeper/KRaft, partition management), overkill for our event volume
3. **NATS JetStream** — lightweight, built-in persistence layer on top of NATS core, simple deployment, subject-based routing, at-least-once delivery

## Decision

We will use **NATS JetStream** as the event bus for all asynchronous inter-service communication.

Event subjects will follow the pattern: `parkir.<domain>.<event>` (e.g., `parkir.presence.spot_updated`, `parkir.reservation.completed`, `parkir.payment.confirmed`).

Each consuming service will use durable push-based subscriptions with explicit acknowledgment.

## Consequences

### Positive

- Lightweight: single binary, minimal resource footprint, simple to deploy on Cloud Run or GKE
- At-least-once delivery: JetStream provides persistence and redelivery on NACK/timeout
- Subject-based routing: flexible topic hierarchy without complex exchange/binding configuration
- Low latency: NATS core is extremely fast for pub/sub
- Simple operations: no ZooKeeper, no partition rebalancing, no broker coordination

### Negative

- Smaller ecosystem: fewer connectors, less community tooling compared to Kafka
- No exactly-once semantics: consumers must be idempotent (mitigated by deduplication at application level)
- Less mature for event sourcing: not ideal if we later need full event replay at scale
- Monitoring: fewer off-the-shelf dashboards compared to Kafka (mitigated by OpenTelemetry integration)
- Message ordering: per-subject ordering only, not global ordering (acceptable for our use cases)
