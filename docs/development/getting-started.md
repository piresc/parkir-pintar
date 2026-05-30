# Getting Started

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.10+ | Service runtime |
| Docker & Docker Compose | Latest | Infrastructure services (PostgreSQL, Redis, NATS) |
| buf | v1.x | Protobuf code generation |
| protoc | v3.x | Protocol Buffers compiler |
| golang-migrate | v4.x | Database schema migrations |
| golangci-lint | Latest | Linting |
| k6 | Latest | Load testing (optional) |

Install all Go-based tools in one command:

```bash
make tools
```

This installs `golangci-lint`, `gosec`, `govulncheck`, `migrate`, `protoc-gen-go`, and `protoc-gen-go-grpc`.

## Local Setup

### 1. Clone and configure

```bash
git clone <repo-url> parkir-pintar
cd parkir-pintar
cp config/.env.example config/.env
```

Edit `config/.env` with your local credentials:

```
DB_USERNAME=postgres
DB_PASSWORD=secret
REDIS_PASSWORD=secret
JWT_SECRET=a-32-char-minimum-secret-string
```

### 2. Start infrastructure

```bash
make docker-up
```

This starts PostgreSQL (`:5432`), Redis (`:6379`), and NATS (`:4222` / monitoring `:8222`) via Docker Compose. All services include health checks — the command waits for readiness.

### 3. Run migrations

```bash
make migrate-up
```

Applies all SQL migrations from `db/migrations/` to the database. All service schemas are created in a single init migration.

### 4. Run services

**Full stack (Docker):**

```bash
docker compose up -d
```

Builds and runs all services: gateway (`:11000`), search (`:11001`), reservation (`:11002`), billing (`:11003`), payment (`:11004`), presence (`:11005`), analytics (`:11006`).

**Single service (local binary):**

```bash
make build-service SVC=gateway
./bin/gateway
```

Or run directly:

```bash
go run ./cmd/gateway
```

## Running Tests

### Unit tests

```bash
make test
```

Runs all tests in short mode with race detector enabled.

### Unit tests with coverage

```bash
make test-coverage
```

Generates HTML coverage report at `.coverage/coverage.html`.

### Integration tests

Requires infrastructure running (`make docker-up`):

```bash
make test-integration
```

Uses build tag `integration` — tests in files tagged `//go:build integration`.

### E2E tests

Requires full stack running:

```bash
docker compose up -d
go test -tags=e2e_docker ./tests/e2e/...
```

### Benchmarks

```bash
make bench                          # all services
make bench-service SVC=reservation  # single service
```

### Load tests

```bash
make load-test          # smoke (10 VUs, 30s)
make load-test-stress   # stress profile
make load-test-spike    # spike profile
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `help` | Show all available targets |
| `build` | Build all service binaries to `./bin/` |
| `build-service SVC=<name>` | Build a single service binary |
| `test` | Run unit tests with race detector (short mode) |
| `test-coverage` | Run tests with coverage report |
| `test-integration` | Run integration tests (requires Docker) |
| `bench` | Run all benchmarks |
| `bench-service SVC=<name>` | Run benchmarks for one service |
| `lint` | Run golangci-lint |
| `fmt` | Format code and check for unformatted files |
| `vet` | Run go vet |
| `security` | Run gosec + govulncheck |
| `proto` | Generate protobuf/gRPC code via buf |
| `docker-up` | Start infrastructure (Postgres, Redis, NATS) |
| `docker-down` | Stop and remove infrastructure containers |
| `docker-logs` | Tail Docker Compose logs |
| `migrate-up` | Apply all database migrations |
| `migrate-down` | Rollback last migration |
| `migrate-create NAME=<desc>` | Create a new migration file |
| `load-test` | Run k6 load test |
| `load-test-stress` | Run k6 stress test |
| `load-test-spike` | Run k6 spike test |
| `ci` | Run full CI pipeline locally (fmt, vet, lint, security, test, build) |
| `clean` | Remove build artifacts and caches |
| `tools` | Install all development tools |
| `mod` | Tidy and verify Go modules |
| `swagger` | Serve Swagger UI locally (http://localhost:8090) |
| `generate` | Run go generate |

## Proto Generation Workflow

Protobuf definitions live in `proto/`. Code generation uses [buf](https://buf.build/) configured in `buf.gen.yaml`:

```bash
make proto
```

This runs `buf generate`, which invokes:
- `protoc-gen-go` — generates Go message types (output co-located with module)
- `protoc-gen-go-grpc` — generates gRPC service stubs

Generated files are placed alongside their corresponding Go packages using the `module=parkir-pintar` option.

**Workflow when modifying protos:**

1. Edit `.proto` files in `proto/`
2. Run `make proto`
3. Implement the new/updated service interface in the corresponding `internal/<service>/` package
4. Run `make build` to verify compilation

## Common Development Workflows

### Adding a new API endpoint

1. Define the endpoint in the gateway router (`cmd/gateway/`)
2. If it calls a gRPC service, add the RPC to the relevant `.proto` file
3. Run `make proto` to regenerate stubs
4. Implement the handler in `internal/<service>/handler/`
5. Add business logic in `internal/<service>/usecase/`
6. Add repository methods if data access is needed
7. Write unit tests, run `make test`
8. Run `make ci` before pushing

### Adding a new microservice

1. Create `cmd/<service>/main.go` with service bootstrap
2. Create `cmd/<service>/Dockerfile`
3. Add the service to the `SERVICES` list in the Makefile
4. Add the service to `docker-compose.yml`
5. Define proto files in `proto/<service>/`
6. Run `make proto` and implement the service
7. Create migrations: `make migrate-create NAME=init_schema`

### Adding a database migration

```bash
make migrate-create NAME=add_vehicle_type_column
```

This creates timestamped up/down SQL files in `db/migrations/`. Write the migration SQL, then apply:

```bash
make migrate-up
```

---

*Owner: ParkirPintar Engineering*
