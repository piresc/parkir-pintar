# Contributing to ParkirPintar

## Development Setup

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- protoc (Protocol Buffers compiler)
- golangci-lint v2.12+

### Getting Started

```bash
git clone git@github.com:piresc/parkir-pintar.git
cd parkir-pintar
go mod download
```

### Running Locally

```bash
docker compose up -d   # starts Postgres, Redis, NATS, all services
```

Services are exposed on ports 11000-11006:
| Service       | Port  |
|---------------|-------|
| Gateway       | 11000 |
| Search        | 11001 |
| Reservation   | 11002 |
| Billing       | 11003 |
| Payment       | 11004 |
| Presence      | 11005 |
| Notification  | 11006 |

## Coding Style

### Go Conventions

- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Use `context.Context` as the first parameter in all public functions that do I/O.
- Return errors explicitly; never ignore them (except where excluded in `.golangci.yml`).
- Keep interfaces small and focused (1-3 methods).
- Use `log/slog` for structured logging — no `fmt.Println` or `log.Printf`.

### Naming

- Package names: short, lowercase, no underscores (e.g., `usecase`, `model`, `repository`).
- Exported types: `PascalCase` with Godoc comments.
- Test functions: `Test[Function]_Should[Result]_When[Condition]`.
- Benchmark functions: `Benchmark[Function]_[Scenario]`.

### Error Handling

- Use `pkg/apperror` for domain errors with HTTP/gRPC status codes.
- Wrap errors with context: `fmt.Errorf("reservation: create: %w", err)`.
- Never log and return the same error — do one or the other.

### Testing

- AAA pattern: Arrange → Act → Assert.
- Use `testify/mock` at interface boundaries.
- Call `mock.AssertExpectations(t)` on all mocks.
- Use `t.Context()` (Go 1.24+) for test contexts.
- Property-based tests use `testing/quick` or `rapid`.
- Unit tests: `*_test.go` in the same package.
- Integration tests: `tests/integration/` with build tags.
- E2E tests: `tests/e2e_docker/` with `//go:build e2e_docker`.

### Proto / gRPC

- Proto files live in `proto/` with service-specific subdirectories.
- Generate with: `protoc --go_out=. --go-grpc_out=. proto/**/*.proto`
- After generation, copy files back if needed (see protoc quirk in project docs).

## Git Workflow

### Branching

- `main` — production-ready, auto-deploys to staging via CI/CD.
- Feature branches: `feat/short-description`.
- Bug fixes: `fix/short-description`.

### Commits

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add payment timeout worker
fix: resolve race condition in spot assignment
ci: add govulncheck to pipeline
docs: update API documentation
refactor: extract billing calculation logic
test: add property-based tests for reservation model
```

### Pull Requests

1. Create a feature branch from `main`.
2. Make changes, ensure `go test -race ./...` passes.
3. Push and open a PR via `gh pr create`.
4. CI must pass (lint, test, security scan, vulnerability check).
5. Squash-merge into `main`.

## CI Pipeline

The CI pipeline runs on every push to `main` and on PRs:

1. **Secret Scan** — gitleaks
2. **Lint** — golangci-lint (continue-on-error)
3. **Test** — race detector + coverage
4. **Security Scan** — gosec
5. **Vulnerability Check** — govulncheck
6. **SonarCloud** — code quality analysis
7. **Build & Push** — Docker image to GHCR (push to main only)

## Architecture

ParkirPintar uses a microservices architecture with per-service PostgreSQL schemas:

- **Gateway** — HTTP REST API, routes to gRPC services
- **Search** — Spot availability queries (Redis-cached read model)
- **Reservation** — Booking lifecycle with distributed locking
- **Billing** — Fee calculation, invoicing
- **Payment** — Payment processing (QRIS, e-wallet, etc.)
- **Presence** — GPS-based check-in/check-out
- **Notification** — Event-driven notifications via NATS

### Communication

- Synchronous: gRPC between services
- Asynchronous: NATS JetStream for events (spot status changes, billing triggers)
- Caching: Redis with 5s TTL + singleflight deduplication

### Observability

Full OpenTelemetry pipeline:
- **Traces** → Alloy → Tempo
- **Metrics** → Alloy → Prometheus
- **Logs** → Alloy → Loki
- **Dashboards** → Grafana

## Database Migrations

Migrations live in `db/migrations/` using sequential numbering:

```
000001_init.up.sql / 000001_init.down.sql
000002_parkir_pintar.up.sql / 000002_parkir_pintar.down.sql
...
```

Every `.up.sql` must have a corresponding `.down.sql` for rollback.
