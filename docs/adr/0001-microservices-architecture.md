# 1. Microservices Architecture with Per-Service PostgreSQL Schemas

## Status

Accepted

## Context

ParkirPintar is a smart parking system composed of multiple bounded contexts: search, reservation, billing, payment, and presence (sensor/occupancy). Each context has distinct data models, scaling requirements, and development lifecycles.

We evaluated two architectural approaches:

1. **Monolithic application** — single deployable unit with a shared database schema
2. **Microservices** — independent services per bounded context, each owning its own PostgreSQL schema

The system needs to handle concurrent reservations, real-time spot availability, and payment processing — domains with different load profiles and reliability requirements. The team expects to iterate on billing and payment independently from the core search/presence logic.

## Decision

We will use a **microservices architecture** where each service owns a dedicated PostgreSQL schema (logical separation within shared or separate database instances depending on environment).

Services:
- **search-service** — spot discovery, filtering, geolocation queries
- **reservation-service** — booking lifecycle management
- **billing-service** — tariff calculation, invoice generation
- **payment-service** — payment gateway integration, transaction records
- **presence-service** — real-time sensor data, occupancy state

Each service exposes its own API, manages its own schema migrations, and can be deployed independently.

## Consequences

### Positive

- Independent scaling: presence-service (high write throughput) scales separately from billing-service (bursty, compute-heavy)
- Independent deployment: changes to billing logic don't require redeploying the entire system
- Clear ownership boundaries: teams can work on services without stepping on each other
- Fault isolation: a failure in payment-service doesn't take down spot search
- Schema independence: each service evolves its data model without cross-service migration coordination

### Negative

- Operational complexity: more services to monitor, deploy, and debug
- Distributed transactions: cross-service consistency requires eventual consistency patterns (sagas, events)
- Network overhead: inter-service calls add latency compared to in-process function calls
- Local development: requires running multiple services or using service stubs
- Data duplication: some denormalization needed across service boundaries
