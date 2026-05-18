# Parkir Pintar Codebase Fixes — Master Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 55+ issues identified in the full codebase review, organized into 3 independent phases that can be executed in parallel.

**Architecture:** Each phase targets an independent concern (security, concurrency, infrastructure). Within each phase, tasks are ordered by dependency but are otherwise independent across phases.

**Tech Stack:** Go 1.25, React 19, PostgreSQL, Redis, NATS JetStream, gRPC, Gin, Terraform, Docker

---

## Phase Structure

| Phase | File | Focus | Tasks |
|-------|------|-------|-------|
| 1 | `2026-05-18-phase1-security.md` | Security & Auth fixes | 8 tasks |
| 2 | `2026-05-18-phase2-concurrency.md` | Concurrency & Data Integrity | 10 tasks |
| 3 | `2026-05-18-phase3-reliability.md` | Reliability, Infra & Quality | 12 tasks |

## Execution Order

Phases 1, 2, and 3 are **independent** — they can be executed in parallel by separate agents. Within each phase, tasks are sequential.

## Verification

After all phases complete:
1. `go build ./...` — all services compile
2. `go test ./...` — all tests pass
3. `golangci-lint run` — no new lint violations
4. `docker compose build` — all images build successfully
5. `cd frontend && npm run build` — frontend compiles

## Out of Scope (Follow-up)

These lower-priority issues are not covered in this plan and should be addressed in a follow-up:

- Frontend accessibility (focus trapping, ARIA labels, form label association)
- Frontend UX (double-click protection, AbortController, optimistic updates)
- Frontend cleanup (remove unused Redux deps, extract QRISPlaceholder)
- Proto enum types (string → proto enum migration)
- Pagination on ListByDriver
- Analytics request filtering parameters
- Monitoring: pin image versions, configure real Alertmanager receiver
- Remove dead `StatusPending` from reservation model
- Add presence service tests
- Standardize telemetry init across all services
- Extract `getEnv` helper to shared package
- Make config path and rate limit CleanupInterval configurable
