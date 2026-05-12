# Kompetensi 3 — Software Quality (Level 4) Gap Analysis

## Overview

**Project:** parkir-pintar (Go microservices parking application)  
**Level:** 4 — Advanced  
**Focus:** Non-functional testing (performance, load, security) and CI/CD integration  
**Date:** 2025-05-12

---

## Sub-Kompetensi 1: Prepare Automated Testing Environment

### Level 4 Requirement

> Mampu menyiapkan kelengkapan non-functional test (performance test, load test, security test)

**Assessment:** Melakukan persiapan pelaksanaan non-functional test

### What Currently EXISTS ✅

| Category | Evidence | Path |
|----------|----------|------|
| Load test framework | Custom Go load tests with 100 concurrent users, sustained waves, mixed operations, spot exhaustion scenarios | `tests/e2e/load_test.go` |
| Race condition testing | 6 race tests with `-race` flag, 50 goroutines per test | `tests/e2e/race_test.go` |
| SAST (Static Security) | gosec integrated in CI pipeline | `.github/workflows/ci.yml` |
| Secret scanning | gitleaks integrated in CI pipeline | `.github/workflows/ci.yml` |
| SonarCloud quality gate | Coverage + code quality analysis | `.github/workflows/ci.yml` |
| Test infrastructure | Testcontainers-go for real Postgres/Redis/NATS | `tests/e2e/` (22 files) |
| Docker Compose stack | Full 7-service integration environment | `tests/e2e_docker/` (6 files) |
| Makefile targets | `test-load`, `test-race`, `test-race-e2e`, `test-e2e`, `test-e2e-docker` | `Makefile` |
| ZAP variable defined | `ZAP_DOCKER_VERSION` variable exists | `Makefile` or CI config |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No DAST (Dynamic Application Security Testing) | Security vulnerabilities in running application not detected | **HIGH** |
| No HTTP-level load testing tool (k6/Vegeta/Gatling) | Cannot simulate realistic HTTP traffic patterns with proper reporting | **HIGH** |
| No Go benchmarks (`func BenchmarkXxx`) | Cannot measure and track function-level performance regressions | **MEDIUM** |
| No chaos/resilience testing (toxiproxy) | Cannot verify system behavior under network failures | **MEDIUM** |
| No API contract tests (buf breaking) | gRPC schema breaking changes not caught automatically | **MEDIUM** |
| No explicit coverage threshold gate | Coverage enforcement relies solely on SonarCloud (no local gate) | **LOW** |
| No mutation testing | Test quality/effectiveness not measured | **LOW** |

### ACTION ITEMS

#### 1. Set up OWASP ZAP DAST environment

```bash
# File: tests/security/zap-config.yaml
# File: tests/security/zap-baseline.conf
# File: scripts/run-zap-scan.sh
```

- Create ZAP configuration for API scanning against running services
- Define target endpoints (gateway service HTTP API)
- Configure authentication context (JWT tokens)
- Create baseline scan rules (suppress known false positives)
- Use `zaproxy/zap-stable` Docker image

#### 2. Set up k6 load testing framework

```bash
# Directory: tests/load/
# Files:
#   tests/load/scenarios/parking-flow.js
#   tests/load/scenarios/spot-availability.js
#   tests/load/scenarios/payment-processing.js
#   tests/load/config.js
#   tests/load/thresholds.json
```

- Install k6 or use Docker image `grafana/k6`
- Define load profiles: smoke, average, stress, spike, soak
- Set performance thresholds (p95 < 500ms, error rate < 1%)
- Create realistic user scenarios matching production traffic patterns

#### 3. Add Go benchmarks

```bash
# Files:
#   services/parking-service/internal/handler/benchmark_test.go
#   services/billing-service/internal/handler/benchmark_test.go
#   services/spot-service/internal/handler/benchmark_test.go
```

- Write `BenchmarkXxx` functions for hot paths
- Add `Makefile` target: `test-bench`
- Use `benchstat` for comparison between runs

#### 4. Set up chaos testing with toxiproxy

```bash
# File: tests/resilience/toxiproxy_test.go
# File: tests/resilience/docker-compose.toxiproxy.yml
```

- Add toxiproxy container to test infrastructure
- Test: database connection timeout handling
- Test: NATS disconnection and reconnection
- Test: Redis unavailability fallback behavior

#### 5. Add buf breaking change detection

```bash
# File: buf.yaml (update)
# File: buf.gen.yaml (update)
```

- Configure `buf breaking` against main branch
- Detect backward-incompatible gRPC schema changes

### Assessment Criteria (How to Demonstrate)

- [ ] Show OWASP ZAP configuration and scan results against running services
- [ ] Show k6 load test scripts with defined thresholds and scenarios
- [ ] Show Go benchmark results with `go test -bench=.` output
- [ ] Show toxiproxy resilience test setup and passing tests
- [ ] Show all non-functional test environments can be spun up with single commands
- [ ] Document test environment requirements and setup instructions

---

## Sub-Kompetensi 2: Write Automated Test & CI/CD Integration

### Level 4 Requirement

> Mampu mengimplementasikan automated non-functional test  
> Mampu mengintegrasikan automated non-functional test dalam proses CI/CD

**Assessment:** Membuat automated non-functional test. Mengintegrasikan non-functional test ke proses CI/CD

### What Currently EXISTS ✅

| Category | Evidence | Path |
|----------|----------|------|
| Automated load tests | 6 scenarios: concurrent parking, sustained waves, mixed operations, spot exhaustion, billing under load, payment gateway stress | `tests/e2e/load_test.go` |
| Automated race tests | 6 race condition tests with 50 goroutines each | `tests/e2e/race_test.go` |
| Property-based tests | Spot inventory, billing calculation, state machine properties | `tests/e2e/property_*_test.go` |
| CI security scanning | gosec SAST + gitleaks secret scanning runs on every PR | `.github/workflows/ci.yml` |
| CI quality gate | SonarCloud analysis with coverage reporting | `.github/workflows/ci.yml` |
| Race detection in CI | Unit tests run with `-race` flag in CI | `.github/workflows/ci.yml` |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| Load tests NOT in CI/CD pipeline | Performance regressions not caught before merge | **CRITICAL** |
| E2E tests NOT in CI/CD pipeline | Integration failures not caught automatically | **CRITICAL** |
| No DAST stage in CI/CD | Security vulnerabilities in running app not detected in pipeline | **HIGH** |
| No k6 load test execution in CI | HTTP-level performance not validated in pipeline | **HIGH** |
| No benchmark regression detection in CI | Function-level performance regressions not tracked | **MEDIUM** |
| No non-functional test reporting/artifacts | No historical performance data or trend analysis | **MEDIUM** |
| No explicit coverage threshold enforcement | Can merge with low coverage if SonarCloud is slow/unavailable | **LOW** |

### ACTION ITEMS

#### 1. Add E2E tests to CI/CD pipeline (CRITICAL)

```yaml
# File: .github/workflows/ci.yml (update)
# Add new job: e2e-tests

e2e-tests:
  runs-on: ubuntu-latest
  needs: [build]
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    - name: Run E2E tests (testcontainers)
      run: make test-e2e
      env:
        TESTCONTAINERS_RYUK_DISABLED: "true"
```

#### 2. Add load tests to CI/CD pipeline (CRITICAL)

```yaml
# File: .github/workflows/ci.yml (update)
# Add new job: load-tests

load-tests:
  runs-on: ubuntu-latest
  needs: [e2e-tests]
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    - name: Run load tests
      run: make test-load
    - name: Upload load test results
      uses: actions/upload-artifact@v4
      with:
        name: load-test-results
        path: tests/results/
```

#### 3. Add DAST (OWASP ZAP) stage to CI/CD

```yaml
# File: .github/workflows/security.yml (new)
# OR add to existing ci.yml

dast-scan:
  runs-on: ubuntu-latest
  needs: [e2e-tests]
  steps:
    - uses: actions/checkout@v4
    - name: Start services
      run: docker compose -f docker-compose.yml up -d
    - name: Wait for services
      run: scripts/wait-for-services.sh
    - name: Run ZAP API scan
      uses: zaproxy/action-api-scan@v0.7.0
      with:
        target: 'http://localhost:8080/api/v1'
        rules_file_name: 'tests/security/zap-rules.tsv'
        cmd_options: '-a -j'
    - name: Upload ZAP report
      uses: actions/upload-artifact@v4
      with:
        name: zap-report
        path: report_html.html
```

#### 4. Add k6 load tests to CI/CD

```yaml
# File: .github/workflows/ci.yml (update)

k6-load-test:
  runs-on: ubuntu-latest
  needs: [e2e-tests]
  steps:
    - uses: actions/checkout@v4
    - name: Start services
      run: docker compose up -d
    - name: Run k6 smoke test
      uses: grafana/k6-action@v0.3.1
      with:
        filename: tests/load/scenarios/parking-flow.js
        flags: --out json=results.json
    - name: Check thresholds
      run: |
        if grep -q '"thresholds":.*"failed":true' results.json; then
          echo "Performance thresholds exceeded!"
          exit 1
        fi
```

#### 5. Add benchmark regression detection

```yaml
# File: .github/workflows/ci.yml (update)

benchmarks:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - name: Run benchmarks
      run: go test ./... -bench=. -benchmem -count=5 | tee bench-current.txt
    - name: Compare with baseline
      uses: benchmark-action/github-action-benchmark@v1
      with:
        tool: 'go'
        output-file-path: bench-current.txt
        alert-threshold: '150%'
        fail-on-alert: true
```

#### 6. Add coverage threshold enforcement

```yaml
# File: .github/workflows/ci.yml (update)
# Add step after test execution:

- name: Check coverage threshold
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    echo "Total coverage: ${COVERAGE}%"
    if (( $(echo "$COVERAGE < 70" | bc -l) )); then
      echo "Coverage ${COVERAGE}% is below 70% threshold"
      exit 1
    fi
```

#### 7. Implement k6 load test scripts

```javascript
// File: tests/load/scenarios/parking-flow.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },   // ramp up
    { duration: '1m', target: 50 },    // sustained load
    { duration: '30s', target: 100 },  // stress
    { duration: '30s', target: 0 },    // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  // Entry → Park → Exit → Pay flow
  const entryRes = http.post(`${__ENV.BASE_URL}/api/v1/parking/entry`, ...);
  check(entryRes, { 'entry status 201': (r) => r.status === 201 });
  sleep(1);
}
```

#### 8. Write OWASP ZAP automation script

```bash
# File: scripts/run-zap-scan.sh
#!/bin/bash
set -e

ZAP_IMAGE="zaproxy/zap-stable:${ZAP_DOCKER_VERSION:-2.14.0}"
TARGET_URL="${TARGET_URL:-http://host.docker.internal:8080}"

docker run --rm \
  --network host \
  -v $(pwd)/tests/security:/zap/wrk:rw \
  ${ZAP_IMAGE} \
  zap-api-scan.py \
  -t "${TARGET_URL}/api/v1/openapi.json" \
  -f openapi \
  -r zap-report.html \
  -c zap-rules.tsv
```

### Assessment Criteria (How to Demonstrate)

- [ ] Show CI/CD pipeline with non-functional test stages (load, security, performance)
- [ ] Show successful pipeline run with all non-functional tests passing
- [ ] Show load test results with defined thresholds (p95 latency, error rate)
- [ ] Show DAST scan report from CI/CD (OWASP ZAP HTML report)
- [ ] Show benchmark comparison detecting performance regression
- [ ] Show pipeline failing when performance thresholds are exceeded
- [ ] Show pipeline failing when security vulnerabilities are found
- [ ] Show test artifacts (reports, metrics) stored as CI/CD artifacts
- [ ] Demonstrate that non-functional tests run automatically on every PR/merge

---

## Summary: Priority Implementation Roadmap

### Phase 1 — Critical (Week 1)

| # | Task | Files |
|---|------|-------|
| 1 | Add E2E tests to CI pipeline | `.github/workflows/ci.yml` |
| 2 | Add load tests to CI pipeline | `.github/workflows/ci.yml` |
| 3 | Add coverage threshold gate | `.github/workflows/ci.yml` |

### Phase 2 — High Priority (Week 2)

| # | Task | Files |
|---|------|-------|
| 4 | Set up k6 load testing | `tests/load/scenarios/*.js`, `tests/load/config.js` |
| 5 | Implement OWASP ZAP DAST | `tests/security/`, `.github/workflows/security.yml` |
| 6 | Add k6 to CI pipeline | `.github/workflows/ci.yml` |
| 7 | Add ZAP to CI pipeline | `.github/workflows/ci.yml` or `security.yml` |

### Phase 3 — Medium Priority (Week 3)

| # | Task | Files |
|---|------|-------|
| 8 | Write Go benchmarks | `services/*/internal/handler/benchmark_test.go` |
| 9 | Add benchmark regression CI | `.github/workflows/ci.yml` |
| 10 | Set up toxiproxy resilience tests | `tests/resilience/` |
| 11 | Add buf breaking detection | `buf.yaml`, `.github/workflows/ci.yml` |

### Phase 4 — Nice to Have (Week 4)

| # | Task | Files |
|---|------|-------|
| 12 | Mutation testing (go-mutesting) | `.github/workflows/ci.yml` |
| 13 | Performance trend dashboard | GitHub Pages or external |
| 14 | Soak testing (long-running) | `tests/load/scenarios/soak.js` |

---

## Current Score Estimate

| Sub-Kompetensi | Current | Target | Gap |
|----------------|---------|--------|-----|
| Sub 1: Prepare non-functional test environment | 60% | 100% | Missing DAST setup, k6, benchmarks |
| Sub 2: Write & integrate non-functional tests in CI/CD | 40% | 100% | Load/E2E not in CI, no DAST in CI |

**Overall Kompetensi 3 Readiness: ~50%**

The existing load tests and security scanning provide a solid foundation. The critical gap is **CI/CD integration** — the non-functional tests exist but don't run automatically in the pipeline. Closing this gap (Phase 1-2) would bring readiness to ~85%.
