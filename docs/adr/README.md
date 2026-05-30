# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the ParkirPintar project.

ADRs document significant architectural decisions, their context, and consequences. They serve as a historical record for the team to understand *why* decisions were made.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [0001](0001-microservices-architecture.md) | Microservices Architecture with Per-Service PostgreSQL Schemas | Accepted |
| [0002](0002-grpc-internal-communication.md) | gRPC for Internal Service Communication | Accepted |
| [0003](0003-nats-event-driven.md) | NATS JetStream for Asynchronous Event-Driven Communication | Accepted |
| [0004](0004-distributed-locking-redis.md) | Redis SETNX for Distributed Locking on Spot Reservation | Accepted |
| [0005](0005-opentelemetry-observability.md) | OpenTelemetry Pipeline for Observability | Accepted |
| [0006](0006-terraform-iac.md) | Terraform for Infrastructure-as-Code Targeting AWS EKS | Accepted |

## Format

Each ADR follows this structure:

- **Title** — short noun phrase describing the decision
- **Status** — Proposed, Accepted, Deprecated, or Superseded
- **Context** — forces at play, problem statement, constraints
- **Decision** — what we decided and why
- **Consequences** — positive and negative outcomes of the decision
