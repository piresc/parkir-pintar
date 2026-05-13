# Technology Stack Comparison

This document explains the technology choices made for ParkirPintar, comparing alternatives and providing rationale for each decision.

## Language: Go

### Comparison

| Criteria | Go | Java | Node.js | Rust |
|----------|-----|------|---------|------|
| Compilation speed | Fast | Slow | N/A (interpreted) | Slow |
| Runtime performance | High | High | Medium | Very High |
| Memory footprint | Low | High | Medium | Very Low |
| Concurrency model | Goroutines (simple) | Threads/Virtual Threads | Event loop (single-threaded) | async/Tokio (complex) |
| Learning curve | Low | Medium | Low | High |
| Ecosystem for microservices | Strong | Very Strong | Strong | Growing |
| Container image size | Small (~10-20MB) | Large (~200MB+) | Medium (~100MB) | Small (~10-20MB) |
| gRPC support | Native (Google-maintained) | Good | Good | Good |
| Deployment simplicity | Single binary | JVM required | Node runtime required | Single binary |

### Decision Rationale

**Go** was chosen for ParkirPintar because:

- **Simplicity and productivity** — Go's straightforward syntax enables fast development without sacrificing performance
- **Native concurrency** — Goroutines handle concurrent parking operations (spot allocation, payment processing) naturally
- **Small deployment footprint** — Single static binaries produce minimal Docker images, ideal for Coolify deployment
- **First-class gRPC support** — Google maintains the Go gRPC implementation, ensuring tight integration
- **Fast compilation** — Quick feedback loops during development and CI/CD
- **Strong standard library** — HTTP servers, JSON handling, and crypto built-in without heavy frameworks

### When to Reconsider

- If the team grows significantly and Java/Spring expertise dominates
- If extreme low-latency requirements emerge (consider Rust)
- If rapid prototyping with a dynamic language becomes more valuable than type safety

---

## Database: PostgreSQL

### Comparison

| Criteria | PostgreSQL | MySQL | MongoDB |
|----------|-----------|-------|---------|
| ACID compliance | Full | Full | Configurable |
| JSON support | Excellent (JSONB) | Basic (JSON type) | Native |
| Concurrency handling | MVCC | MVCC (InnoDB) | Document-level locking |
| Geospatial | PostGIS (excellent) | Basic | Good (GeoJSON) |
| Complex queries | Excellent | Good | Limited (aggregation pipeline) |
| Schema flexibility | Structured + JSONB | Structured | Schema-less |
| Ecosystem/tooling | Excellent | Excellent | Good |
| Horizontal scaling | Citus/partitioning | Vitess/Group Replication | Native sharding |

### Decision Rationale

**PostgreSQL** was chosen because:

- **Relational data model fits parking domain** — Parking lots, spots, users, and transactions have clear relationships
- **ACID transactions** — Critical for payment processing and spot allocation (no double-booking)
- **PostGIS** — Geospatial queries for "find nearest parking" features
- **JSONB** — Flexible metadata storage without sacrificing relational integrity
- **sqlc compatibility** — Excellent tooling for type-safe Go code generation
- **Proven at scale** — Handles the expected load without exotic scaling solutions

### When to Reconsider

- If document-oriented data becomes dominant (unlikely for parking domain)
- If horizontal write scaling beyond single-node becomes necessary (evaluate Citus)
- If the team has strong MySQL operational expertise and no PostGIS needs

---

## Cache: Redis

### Comparison

| Criteria | Redis | Memcached | Valkey |
|----------|-------|-----------|--------|
| Data structures | Rich (strings, hashes, sets, sorted sets, streams) | Key-value only | Rich (Redis-compatible) |
| Persistence | Optional (RDB/AOF) | None | Optional |
| Pub/Sub | Built-in | None | Built-in |
| Distributed locking | Redlock algorithm | Not built-in | Redlock-compatible |
| Lua scripting | Yes | No | Yes |
| Cluster mode | Yes | Client-side sharding | Yes |
| Maturity | Very mature | Very mature | Newer (Redis fork) |
| Licensing | Source-available (SSPL) | BSD | BSD |

### Decision Rationale

**Redis** was chosen because:

- **Distributed locking** — Critical for concurrent parking spot allocation; Redlock provides the needed guarantees
- **Rich data structures** — Sorted sets for leaderboards, hashes for session data, streams for event buffering
- **Pub/Sub** — Lightweight real-time notifications complement NATS for local cache invalidation
- **Proven ecosystem** — Extensive Go client libraries (go-redis), well-understood operationally
- **Multi-purpose** — Single system for caching, locking, rate limiting, and session storage

### When to Reconsider

- If licensing concerns arise (evaluate Valkey as drop-in replacement)
- If only simple key-value caching is needed (Memcached is simpler)
- If the project moves to a managed cloud offering where Valkey is the default

---

## Messaging: NATS

### Comparison

| Criteria | NATS | RabbitMQ | Kafka |
|----------|------|----------|-------|
| Latency | Very low | Low | Medium |
| Throughput | High | Medium | Very High |
| Operational complexity | Very low | Medium | High |
| Message persistence | JetStream | Durable queues | Log-based (default) |
| Clustering | Built-in (simple) | Clustering + Quorum queues | ZooKeeper/KRaft |
| Memory footprint | Very small (~10MB) | Medium (~100MB+) | Large (~1GB+) |
| Protocol | Custom (simple text) | AMQP | Custom binary |
| Go client | Excellent (official) | Good | Good |
| Message ordering | Per-subject | Per-queue | Per-partition |
| Exactly-once | JetStream (with dedup) | Publisher confirms | Yes (with transactions) |

### Decision Rationale

**NATS** was chosen because:

- **Operational simplicity** — Single binary, minimal configuration, easy to deploy on Coolify
- **Low resource footprint** — Ideal for a project that doesn't need Kafka-scale throughput
- **JetStream** — Provides persistence and at-least-once delivery when needed
- **Request-reply pattern** — Natural fit alongside gRPC for certain async workflows
- **Go-native** — Written in Go with an excellent official Go client
- **Sufficient for parking domain** — Event volumes (entry/exit/payment) don't require Kafka's log-based architecture

### When to Reconsider

- If event replay/audit requirements demand long-term log retention (Kafka excels here)
- If complex routing patterns emerge (RabbitMQ's exchange model is more flexible)
- If throughput exceeds millions of events/second sustained

---

## Inter-Service Communication: gRPC

### Comparison

| Criteria | gRPC | REST (HTTP/JSON) |
|----------|------|-----------------|
| Performance | High (binary protobuf, HTTP/2) | Medium (text JSON, HTTP/1.1) |
| Type safety | Strong (protobuf schema) | Weak (OpenAPI optional) |
| Streaming | Bidirectional | SSE/WebSocket (separate) |
| Code generation | Built-in (multi-language) | Third-party (OpenAPI generators) |
| Browser support | Requires grpc-web proxy | Native |
| Debugging | Harder (binary) | Easy (curl, browser) |
| Contract evolution | Protobuf backward compatibility | Manual versioning |
| Tooling | grpcurl, Evans, Buf | Postman, curl, httpie |

### Decision Rationale

**gRPC** was chosen for inter-service communication because:

- **Type-safe contracts** — Protobuf schemas prevent integration errors between services
- **Performance** — Binary serialization and HTTP/2 multiplexing reduce latency between services
- **Code generation** — Auto-generated Go clients/servers eliminate boilerplate
- **Streaming** — Server-streaming useful for real-time parking availability updates
- **Schema evolution** — Protobuf's backward compatibility rules make API evolution safe

**Note:** The API gateway still exposes REST/JSON to external clients (mobile apps, web). gRPC is internal only.

### When to Reconsider

- If debugging inter-service calls becomes a major pain point
- If the team prefers REST simplicity and performance is acceptable
- If third-party integrations require direct service-to-service REST

---

## Deployment: Docker + Coolify

### Comparison

| Criteria | Docker + Coolify | Kubernetes |
|----------|-----------------|------------|
| Operational complexity | Low | Very High |
| Learning curve | Low | Steep |
| Cost (small scale) | Low (single VPS) | High (control plane + nodes) |
| Auto-scaling | Basic | Advanced (HPA, VPA, KEDA) |
| Service discovery | Docker networking | Built-in (CoreDNS) |
| Secret management | Environment variables | Secrets API + external stores |
| Deployment UX | Git-push deploy | kubectl/Helm/ArgoCD |
| High availability | Manual (multi-node) | Built-in |
| Resource overhead | Minimal | Significant (~1-2GB for control plane) |

### Decision Rationale

**Docker + Coolify** was chosen because:

- **Right-sized for the project** — ParkirPintar doesn't need Kubernetes' complexity at current scale
- **Cost-effective** — Runs on a single VPS; no managed K8s fees
- **Simple deployment** — Git-push triggers builds and deploys via Coolify
- **Sufficient for microservices** — Docker Compose/networking handles service discovery
- **Fast iteration** — No YAML sprawl or Helm chart maintenance
- **Team size** — Solo/small team doesn't benefit from K8s operational overhead

### When to Reconsider

- If the service count exceeds 10+ and orchestration becomes painful
- If auto-scaling based on parking demand becomes critical
- If multi-region deployment is required
- If the team grows and dedicated DevOps capacity is available

---

## Infrastructure as Code: Terraform

### Comparison

| Criteria | Terraform | Pulumi | AWS CDK |
|----------|-----------|--------|---------|
| Language | HCL (declarative) | General-purpose (Go, TS, Python) | TypeScript/Python |
| State management | Remote state (S3, Terraform Cloud) | Pulumi Cloud / self-managed | CloudFormation |
| Multi-cloud | Excellent | Excellent | AWS only |
| Community/modules | Very large | Growing | AWS-focused |
| Learning curve | Medium (HCL) | Lower (if you know the language) | Lower (if you know AWS) |
| Maturity | Very mature | Mature | Mature |
| Drift detection | terraform plan | pulumi preview | Change sets |
| Provider ecosystem | 3000+ providers | Terraform providers via bridge | AWS only |

### Decision Rationale

**Terraform** was chosen because:

- **Industry standard** — Widest adoption means more examples, modules, and community support
- **Declarative** — HCL's declarative nature makes infrastructure state explicit and reviewable
- **Provider ecosystem** — Supports any cloud/service ParkirPintar might use
- **Separation of concerns** — IaC in HCL stays distinct from application code in Go
- **Mature tooling** — terraform plan, state management, and import are battle-tested
- **CI/CD integration** — Well-supported in GitHub Actions workflows

### When to Reconsider

- If the team strongly prefers writing infrastructure in Go (evaluate Pulumi with Go SDK)
- If locking into a single cloud provider (AWS CDK becomes viable)
- If OpenTofu becomes the clear community successor and Terraform licensing is a concern
