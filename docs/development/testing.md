# ParkirPintar Testing Strategy

## Overview

This document defines the comprehensive testing strategy for ParkirPintar, a microservices-based smart parking system built with Go. The strategy follows the test pyramid model and integrates with CI/CD pipelines for continuous quality assurance.

## Test Pyramid

```
        ┌─────────┐
        │  E2E    │  10% - Full system validation
        │  Tests  │
       ┌┴─────────┴┐
       │Integration │  20% - Service boundaries & data stores
       │   Tests    │
      ┌┴────────────┴┐
      │  Unit Tests   │  70% - Business logic & utilities
      └───────────────┘
```

| Layer       | Proportion | Scope                              | Speed    | Frequency       |
|-------------|------------|------------------------------------|----------|-----------------|
| Unit        | 70%        | Functions, methods, handlers       | < 1s     | Every PR        |
| Integration | 20%        | DB, Redis, NATS, gRPC boundaries   | < 60s    | Every PR        |
| E2E         | 10%        | Full system flows                  | < 5min   | Push to main    |

## Unit Testing

### Framework & Tools

| Tool              | Purpose                                    |
|-------------------|--------------------------------------------|
| `testing`         | Go standard library test runner            |
| `testify/assert`  | Assertion helpers                          |
| `testify/mock`    | Interface mocking                          |
| `testify/suite`   | Test suite organization                    |
| `testing/quick`   | Property-based testing                     |

### Patterns

#### Table-Driven Tests

All unit tests follow Go's table-driven pattern for comprehensive input coverage:

```go
func TestCalculateParkingFee(t *testing.T) {
    tests := []struct {
        name     string
        duration time.Duration
        vehicle  VehicleType
        want     int64
        wantErr  bool
    }{
        {"car_1_hour", 1 * time.Hour, VehicleCar, 5000, false},
        {"motorcycle_1_hour", 1 * time.Hour, VehicleMotorcycle, 2000, false},
        {"zero_duration", 0, VehicleCar, 0, true},
        {"negative_duration", -1 * time.Hour, VehicleCar, 0, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CalculateParkingFee(tt.duration, tt.vehicle)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

#### Mock Generation

Mocks are generated for all service interfaces using `testify/mock`:

```go
// internal/parking/usecase/mock_parking_repo.go
type MockParkingRepository struct {
    mock.Mock
}

func (m *MockParkingRepository) FindAvailableSlots(ctx context.Context, locationID string) ([]Slot, error) {
    args := m.Called(ctx, locationID)
    return args.Get(0).([]Slot), args.Error(1)
}
```

#### Property-Based Testing

Used for domain logic where input space is large:

```go
func TestParkingFeeAlwaysPositive(t *testing.T) {
    f := func(minutes uint16, vehicleType uint8) bool {
        if minutes == 0 {
            return true // skip zero
        }
        vt := VehicleType(vehicleType % 3)
        fee, err := CalculateParkingFee(time.Duration(minutes)*time.Minute, vt)
        return err == nil && fee > 0
    }
    if err := quick.Check(f, nil); err != nil {
        t.Error(err)
    }
}
```

### Unit Test Scope by Layer

| Layer      | What to Test                                    | What to Mock                    |
|------------|------------------------------------------------|---------------------------------|
| Handler    | Request parsing, response formatting, errors   | Usecase interfaces              |
| Usecase    | Business logic, orchestration, validation      | Repository & external services  |
| Repository | Query building, result mapping                 | Database (use integration)      |
| Domain     | Value objects, entities, domain rules          | Nothing (pure logic)            |

## Integration Testing

### Infrastructure

Integration tests run against real dependencies using Docker containers managed by `testcontainers-go` or a shared `docker-compose.test.yml`.

#### Docker Compose Test Stack

```yaml
# docker-compose.test.yml
services:
  postgres-test:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: parkir_pintar_test
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
    ports:
      - "5433:5432"
    tmpfs:
      - /var/lib/postgresql/data

  redis-test:
    image: redis:7-alpine
    ports:
      - "6380:6379"

  nats-test:
    image: nats:2-alpine
    ports:
      - "4223:4222"
```

### Build Tags

Integration tests use build tags to separate from unit tests:

```go
//go:build integration

package repository_test

func TestParkingRepository_CreateSession(t *testing.T) {
    db := setupTestDB(t)
    repo := NewParkingRepository(db)
    // ...
}
```

### What Integration Tests Cover

| Component          | Tests                                                    |
|--------------------|----------------------------------------------------------|
| PostgreSQL repos   | CRUD operations, migrations, constraints, transactions   |
| Redis cache        | Cache hit/miss, TTL expiry, distributed locks            |
| NATS messaging     | Publish/subscribe, JetStream persistence, replay         |
| gRPC services      | Service-to-service calls, error propagation              |
| Gateway            | HTTP→gRPC translation, middleware chain                  |

### Test Database Management

- Each test suite gets a fresh schema via migrations
- `tmpfs` mount for PostgreSQL eliminates disk I/O
- Tests run in parallel with isolated schemas where possible
- Cleanup via `t.Cleanup()` hooks

## End-to-End (E2E) Testing

### Architecture

E2E tests validate complete user flows against the full docker-compose stack.

```
┌──────────────┐     ┌──────────┐     ┌─────────────┐
│  E2E Test    │────▶│  Traefik │────▶│  Gateway    │
│  (Go + HTTP) │     │  (LB)    │     │  Service    │
└──────────────┘     └──────────┘     └──────┬──────┘
                                              │ gRPC
                                    ┌─────────┼─────────┐
                                    ▼         ▼         ▼
                              ┌─────────┐ ┌────────┐ ┌─────────┐
                              │ Parking │ │Payment │ │ User    │
                              │ Service │ │Service │ │ Service │
                              └─────────┘ └────────┘ └─────────┘
```

### Build Tag

```go
//go:build e2e_docker

package e2e_test

func TestFullParkingFlow(t *testing.T) {
    client := newE2EClient(t)

    // 1. Check in
    session := client.CheckIn(t, CheckInRequest{
        LocationID: "loc-001",
        PlateNumber: "B 1234 ABC",
        VehicleType: "car",
    })
    assert.NotEmpty(t, session.ID)

    // 2. Verify slot occupied
    slots := client.GetAvailableSlots(t, "loc-001")
    assert.Less(t, len(slots), initialSlotCount)

    // 3. Check out with payment
    receipt := client.CheckOut(t, session.ID)
    assert.Greater(t, receipt.Amount, int64(0))
    assert.Equal(t, "completed", receipt.Status)
}
```

### E2E Test Scenarios

| Flow                    | Steps                                                      |
|-------------------------|------------------------------------------------------------|
| Happy path parking      | Check-in → occupancy update → check-out → payment         |
| Concurrent check-in     | Multiple vehicles, verify no double-booking                |
| Payment failure         | Check-out → payment fails → retry → success               |
| Session timeout         | Check-in → exceed max duration → auto-notification        |
| Operator management     | Create location → add slots → verify availability          |

## Performance Testing

### Tool: k6

Performance tests are written in JavaScript for k6 and stored in `tests/performance/`.

### Test Profiles

| Profile    | VUs  | Duration | Threshold                    | When           |
|------------|------|----------|------------------------------|----------------|
| Smoke      | 5    | 30s      | p95 < 500ms, errors < 1%    | Every PR (CI)  |
| Load       | 50   | 5min     | p95 < 1s, errors < 0.5%     | Push to main   |
| Stress     | 200  | 10min    | p99 < 3s, errors < 2%       | Weekly schedule |
| Soak       | 30   | 30min    | No memory leaks, stable p95  | Weekly schedule |

### Key Scenarios

```javascript
// tests/performance/check-in-flow.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '1m', target: 50 },
        { duration: '3m', target: 50 },
        { duration: '1m', target: 0 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<1000'],
        http_req_failed: ['rate<0.01'],
    },
};

export default function () {
    const res = http.post(`${__ENV.BASE_URL}/api/v1/parking/check-in`, JSON.stringify({
        location_id: 'loc-001',
        plate_number: `B ${Math.floor(Math.random() * 9999)} TST`,
        vehicle_type: 'car',
    }), { headers: { 'Content-Type': 'application/json' } });

    check(res, {
        'status is 201': (r) => r.status === 201,
        'has session_id': (r) => JSON.parse(r.body).session_id !== '',
    });
    sleep(1);
}
```

## Security Testing

| Tool         | Type | What It Checks                          | When              |
|--------------|------|-----------------------------------------|-------------------|
| `gosec`      | SAST | Go-specific security issues             | Every PR          |
| `govulncheck`| SCA  | Known vulnerabilities in dependencies   | Every PR          |
| `gitleaks`   | Secret| Hardcoded secrets in source code       | Every PR (pre-commit) |
| Trivy        | Container | Container image vulnerabilities     | Every image build |

### gosec Configuration

```yaml
# .gosec.yml
global:
  nosec: false
  audit: enabled
includes:
  - G101  # Hardcoded credentials
  - G201  # SQL injection
  - G301  # Poor file permissions
  - G401  # Weak crypto
  - G501  # Blacklisted imports
```

## Code Quality

### golangci-lint

Primary linter aggregator with the following enabled linters:

| Linter        | Purpose                              |
|---------------|--------------------------------------|
| `errcheck`    | Unchecked errors                     |
| `govet`       | Suspicious constructs                |
| `staticcheck` | Advanced static analysis             |
| `gosec`       | Security issues                      |
| `gocyclo`     | Cyclomatic complexity (max 15)       |
| `dupl`        | Code duplication                     |
| `misspell`    | Spelling errors in comments          |
| `gocritic`    | Opinionated style checks             |
| `revive`      | Extensible linter                    |

### SonarCloud

Quality gate configuration:

| Metric                    | Threshold |
|---------------------------|-----------|
| Coverage                  | ≥ 80%    |
| Duplicated lines          | < 3%     |
| Maintainability rating    | A        |
| Reliability rating        | A        |
| Security rating           | A        |
| Security hotspots reviewed| 100%     |

## Coverage Targets

| Layer/Package              | Target | Rationale                              |
|----------------------------|--------|----------------------------------------|
| `internal/*/usecase`       | 90%    | Critical business logic                |
| `internal/*/handler`       | 85%    | Request/response handling              |
| `internal/*/repository`    | 75%    | Covered more by integration tests      |
| `pkg/*`                    | 90%    | Shared utilities must be reliable      |
| `cmd/*`                    | 60%    | Startup/wiring code                    |
| **Overall**                | **80%**| Balanced quality target                |

### Coverage Enforcement

- Coverage is measured with `go test -coverprofile`
- CI fails if overall coverage drops below 80%
- Coverage reports uploaded to SonarCloud and Codecov
- Per-package coverage tracked in PR comments

## Test Data Management

### Factories

Test data factories provide consistent, valid test objects:

```go
// internal/testutil/factory.go
func NewParkingSession(opts ...SessionOption) *ParkingSession {
    s := &ParkingSession{
        ID:          uuid.New().String(),
        LocationID:  "loc-test-001",
        PlateNumber: "B 1234 TST",
        VehicleType: VehicleCar,
        CheckInTime: time.Now(),
        Status:      SessionActive,
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

### Fixtures

SQL fixtures for integration tests:

```
tests/fixtures/
├── locations.sql       # Parking locations seed data
├── slots.sql           # Slot configurations
├── users.sql           # Test users with different roles
└── sessions.sql        # Historical parking sessions
```

### Seed Data

- Development: `make seed-dev` loads realistic test data
- Integration tests: fixtures loaded per-test via `t.Helper()` functions
- E2E tests: seed script runs as init container in docker-compose

## CI Integration

### Test Execution Matrix

| Trigger          | Unit | Integration | E2E  | Performance | Security | Lint  |
|------------------|------|-------------|------|-------------|----------|-------|
| PR opened/sync   | ✅   | ✅          | ❌   | Smoke only  | ✅       | ✅    |
| Push to main     | ✅   | ✅          | ✅   | Load        | ✅       | ✅    |
| Weekly schedule  | ✅   | ✅          | ✅   | Stress+Soak | ✅       | ✅    |
| Release tag      | ✅   | ✅          | ✅   | Load        | Full     | ✅    |

### CI Pipeline Stages

```
┌────────┐   ┌──────────┐   ┌─────────────┐   ┌──────────┐   ┌────────┐
│  Lint  │──▶│  Build   │──▶│  Unit Test  │──▶│Integration│──▶│  E2E   │
└────────┘   └──────────┘   └─────────────┘   └──────────┘   └────────┘
                                                                    │
                                                              ┌─────▼─────┐
                                                              │ Deploy    │
                                                              │ (staging) │
                                                              └───────────┘
```

### GitHub Actions Workflow (Summary)

```yaml
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: golangci/golangci-lint-action@v4

  test-unit:
    runs-on: ubuntu-latest
    steps:
      - run: go test -race -coverprofile=coverage.out ./...

  test-integration:
    runs-on: ubuntu-latest
    services:
      postgres: { image: postgres:16-alpine }
      redis: { image: redis:7-alpine }
      nats: { image: nats:2-alpine }
    steps:
      - run: go test -race -tags=integration ./...

  test-e2e:
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - run: docker compose -f docker-compose.yml up -d
      - run: go test -tags=e2e_docker ./tests/e2e/...

  security:
    runs-on: ubuntu-latest
    steps:
      - run: gosec ./...
      - run: govulncheck ./...
      - run: gitleaks detect
```

## Mutation Testing (Future)

Planned adoption of `go-mutesting` or `gremlins` for mutation testing:

- **Goal**: Validate test suite effectiveness by introducing code mutations
- **Target**: Critical usecase layer first
- **Metric**: Mutation score > 70%
- **Timeline**: After achieving 80% coverage baseline
- **Integration**: Scheduled weekly run, results in dashboard

## Test Environment Management

| Environment | Purpose              | Data           | Lifecycle              |
|-------------|----------------------|----------------|------------------------|
| Local       | Developer testing    | Docker Compose | Developer-managed      |
| CI          | Automated testing    | Ephemeral      | Per-pipeline run       |
| Staging     | Pre-production E2E   | Sanitized copy | Persistent, reset weekly|
| Production  | Smoke tests only     | Real data      | Read-only tests        |

### Environment Parity

- All environments use identical Docker images
- Database migrations applied consistently
- Feature flags control environment-specific behavior
- Secrets injected via environment variables (never in images)

---

*Last updated: 2026-05-13*
*Owner: ParkirPintar Engineering*
