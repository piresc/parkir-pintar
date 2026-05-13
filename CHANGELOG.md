# Changelog

All notable changes to ParkirPintar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-05-13

### Added

- Microservices architecture with dedicated services: gateway, user, parking, payment, notification
- gRPC communication for synchronous inter-service calls with protobuf contracts
- NATS JetStream event-driven messaging for asynchronous workflows
- Distributed locking via Redis for concurrent parking spot allocation
- OpenTelemetry (OTel) observability: distributed tracing, metrics, and structured logging
- CI/CD pipeline with GitHub Actions (lint, test, build, deploy stages)
- Terraform Infrastructure as Code (IaC) for provisioning cloud resources
- DAST scanning with OWASP ZAP integrated into CI pipeline
- Load testing with k6 for performance validation
- PostgreSQL per-service databases with sqlc for type-safe queries
- Redis for caching, session management, and distributed locks
- JWT-based authentication with refresh token rotation
- Role-based access control (RBAC) for multi-tenant parking management
- Real-time parking availability tracking
- QR code-based entry/exit for parking facilities
- Payment processing integration
- Notification service (email, push) via NATS events
- Docker containerization with multi-stage builds
- Coolify-based deployment configuration
- Database migrations with golang-migrate
- API documentation via protobuf service definitions
- Health check endpoints for all services
- Graceful shutdown handling across all services

### Changed

- N/A (initial release)

### Fixed

- N/A (initial release)

### Security

- TLS enforcement on all external endpoints
- Input validation at API gateway level
- SQL injection prevention via parameterized queries (sqlc)
- Rate limiting on authentication and public endpoints
- Secret management via environment variables (no hardcoded secrets)
- Container hardening: non-root users, minimal base images
- Dependency vulnerability scanning via Dependabot
- DAST scanning for runtime vulnerability detection

[Unreleased]: https://github.com/piresc/parkir-pintar/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/piresc/parkir-pintar/releases/tag/v1.0.0
