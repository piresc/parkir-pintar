# Chaos / Resilience Testing

This package contains resilience tests that use [Toxiproxy](https://github.com/Shopify/toxiproxy) to inject network faults and verify that ParkirPintar services degrade gracefully under adverse conditions.

## Prerequisites

- **Toxiproxy server** running on `localhost:8474`
- **Redis** on `localhost:6379`
- **NATS** on `localhost:4222`
- **PostgreSQL** on `localhost:5432`
- **gRPC reservation service** on `localhost:50051`

## Running the Tests

```bash
# Start toxiproxy (see Docker Compose below)
docker compose -f tests/chaos/docker-compose.yml up -d

# Run chaos tests (gated by build tag)
go test -tags chaos -v -timeout 120s ./tests/chaos/
```

Tests are excluded from normal `go test ./...` runs because of the `//go:build chaos` constraint.

## What Each Test Validates

| Test | Fault Injected | Validates |
|------|---------------|-----------|
| `TestCircuitBreakerWithRedisLatency` | 2s latency on Redis | Circuit breaker opens after threshold failures, rejects fast, recovers after toxic removal |
| `TestGracefulDegradationNATSDown` | NATS proxy disabled entirely | System returns errors quickly without hanging; recovers when NATS is back |
| `TestDatabaseConnectionPoolExhaustion` | 1 KB/s bandwidth limit on Postgres | Connection attempts fail with clear errors under pool pressure; pool recovers |
| `TestGRPCTimeoutHandling` | 3s latency on gRPC | Client deadlines are respected, circuit breaker opens, fast rejection once open |

## Adding New Chaos Scenarios

1. Define a new test function in `chaos_test.go` (or a new `*_test.go` file).
2. Follow the pattern:
   ```go
   func TestNewScenario(t *testing.T) {
       client := newToxiClient()

       // 1. Create proxy
       proxy, err := client.CreateProxy("name", listenAddr, upstreamAddr)
       require.NoError(t, err)
       defer proxy.Delete()

       // 2. Verify baseline
       // ...

       // 3. Inject fault
       proxy.AddToxic("toxic_name", "type", "direction", 1.0, toxiproxy.Attributes{...})

       // 4. Exercise system under fault
       // ...

       // 5. Assert degradation behavior
       // ...

       // 6. Remove fault and verify recovery
       proxy.RemoveToxic("toxic_name")
   }
   ```
3. Available toxic types: `latency`, `bandwidth`, `slow_close`, `timeout`, `slicer`, `limit_data`, `reset_peer`.

## Docker Compose for Toxiproxy

```yaml
# tests/chaos/docker-compose.yml
version: "3.8"

services:
  toxiproxy:
    image: ghcr.io/shopify/toxiproxy:2.9.0
    ports:
      - "8474:8474"   # API port
      - "26379:26379" # Redis proxy
      - "24222:24222" # NATS proxy
      - "25432:25432" # PostgreSQL proxy
      - "50052:50052" # gRPC proxy
    networks:
      - chaos

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    networks:
      - chaos

  nats:
    image: nats:2-alpine
    ports:
      - "4222:4222"
    command: ["--jetstream"]
    networks:
      - chaos

  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: parkir
      POSTGRES_PASSWORD: parkir
      POSTGRES_DB: parkir_pintar
    networks:
      - chaos

networks:
  chaos:
    driver: bridge
```

## Toxiproxy Proxy Setup

After starting the compose stack, create the proxies via the API or in test setup:

```bash
# Create proxies (or let the tests create them automatically)
curl -X POST http://localhost:8474/proxies \
  -d '{"name":"redis_chaos","listen":"0.0.0.0:26379","upstream":"redis:6379"}'

curl -X POST http://localhost:8474/proxies \
  -d '{"name":"nats_chaos","listen":"0.0.0.0:24222","upstream":"nats:4222"}'

curl -X POST http://localhost:8474/proxies \
  -d '{"name":"postgres_chaos","listen":"0.0.0.0:25432","upstream":"postgres:5432"}'
```
