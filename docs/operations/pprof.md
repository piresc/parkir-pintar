# Runtime Profiling (pprof)

The gateway exposes Go's built-in profiler at `/debug/pprof/*` for diagnosing performance issues in production.

## Enable

Set the environment variable on the gateway container:

```
ENABLE_PPROF=true
```

Disabled by default. No endpoints are registered when off.

## Auth

All pprof routes require:
1. Valid JWT token
2. `admin` role claim

Unauthorized requests get `403 Forbidden`.

## Endpoints

| Path | Description |
|------|-------------|
| `/debug/pprof/` | Index page listing all profiles |
| `/debug/pprof/profile` | CPU profile (30s default) |
| `/debug/pprof/heap` | Heap memory allocations |
| `/debug/pprof/goroutine` | All goroutine stacks |
| `/debug/pprof/trace` | Execution trace |
| `/debug/pprof/cmdline` | Command line arguments |

## Usage

```bash
# CPU profile (30 seconds)
go tool pprof http://localhost:8082/debug/pprof/profile?seconds=30

# Heap snapshot
go tool pprof http://localhost:8082/debug/pprof/heap

# Goroutine dump (detect leaks)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8082/debug/pprof/goroutine?debug=2

# Execution trace (5 seconds)
curl -H "Authorization: Bearer $TOKEN" -o trace.out http://localhost:8082/debug/pprof/trace?seconds=5
go tool trace trace.out
```

Note: `go tool pprof` doesn't pass auth headers. For authenticated endpoints, download the profile with curl first, then analyze locally:

```bash
curl -H "Authorization: Bearer $TOKEN" -o cpu.prof http://localhost:8082/debug/pprof/profile?seconds=30
go tool pprof cpu.prof
```

## When to Use

- **High CPU:** `profile` → find hot functions
- **Memory leak:** `heap` → find allocations that aren't freed
- **Goroutine leak:** `goroutine` → find stuck goroutines growing over time
- **Latency spikes:** `trace` → visualize scheduling and blocking
