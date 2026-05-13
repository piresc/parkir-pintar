# 2. gRPC for Internal Service Communication

## Status

Accepted

## Context

With a microservices architecture (ADR-0001), services need a synchronous communication mechanism for request/response interactions — e.g., reservation-service querying presence-service for real-time spot availability, or billing-service fetching reservation details.

Requirements:
- Strong typing and contract enforcement between services
- Low latency for internal calls
- Support for streaming (e.g., real-time occupancy updates)
- Code generation for multiple languages (Go primarily, but future flexibility)

Alternatives considered:

1. **REST (HTTP/JSON)** — widely understood, easy to debug, but lacks native streaming, no built-in contract enforcement beyond OpenAPI specs, JSON serialization overhead
2. **gRPC (HTTP/2 + Protocol Buffers)** — typed contracts via proto files, bidirectional streaming, efficient binary serialization, native code generation

## Decision

We will use **gRPC** with Protocol Buffers for all synchronous inter-service communication. REST (HTTP/JSON) will be used only at the API gateway layer for external/public-facing endpoints.

Proto definitions will live in a shared `proto/` directory at the repository root, with generated code committed per service.

## Consequences

### Positive

- Type safety: proto contracts catch breaking changes at compile time
- Performance: binary serialization is significantly faster than JSON for high-throughput internal calls
- Streaming: native support for server-streaming (e.g., presence updates) and bidirectional streaming
- Code generation: consistent client/server stubs across services, reducing boilerplate
- HTTP/2 multiplexing: efficient connection reuse between services

### Negative

- Proto management overhead: changes to `.proto` files require regeneration and coordination across services
- Debugging difficulty: binary payloads are harder to inspect than JSON (mitigated by grpcurl and reflection)
- Browser incompatibility: gRPC not directly callable from browsers (not an issue for internal comms, gateway handles external)
- Learning curve: team needs familiarity with protobuf IDL and gRPC patterns
- Tooling: requires protoc compiler and language-specific plugins in CI/CD
