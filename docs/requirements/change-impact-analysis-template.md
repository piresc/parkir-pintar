# Change Impact Analysis Template

**Project:** ParkirPintar — Smart Parking Backend System  
**Version:** 1.0  
**Date:** 2026-05-13  
**Status:** Approved

---

## Template

### Change Request Header

| Field | Value |
|-------|-------|
| **Change Request ID** | CR-XXX |
| **Title** | [Short descriptive title] |
| **Requested By** | [Stakeholder name/role] |
| **Date Submitted** | YYYY-MM-DD |
| **Priority** | Critical / High / Medium / Low |
| **Target Release** | [Version or sprint] |
| **Status** | Draft / Under Review / Approved / Rejected / Implemented |

### 1. Change Description

**Summary:**  
[1-2 paragraph description of what is being changed and why]

**Business Justification:**  
[Why this change is needed — user feedback, regulatory, performance, etc.]

**Scope:**  
- [ ] New feature
- [ ] Feature modification
- [ ] Bug fix
- [ ] Infrastructure change
- [ ] Performance optimization
- [ ] Security patch

---

### 2. Affected Services Checklist

| Service | Affected? | Impact Level | Description of Changes |
|---------|-----------|--------------|----------------------|
| Gateway (:8080) | ☐ Yes / ☐ No | LOW / MED / HIGH | |
| Search (:9092) | ☐ Yes / ☐ No | LOW / MED / HIGH | |
| Reservation (:9091) | ☐ Yes / ☐ No | LOW / MED / HIGH | |
| Billing (:9093) | ☐ Yes / ☐ No | LOW / MED / HIGH | |
| Payment (:9094) | ☐ Yes / ☐ No | LOW / MED / HIGH | |
| Presence (:9095) | ☐ Yes / ☐ No | LOW / MED / HIGH | |
| Notification (:9096) | ☐ Yes / ☐ No | LOW / MED / HIGH | |

---

### 3. Database Migration Required?

| Question | Answer |
|----------|--------|
| Schema migration needed? | ☐ Yes / ☐ No |
| New tables? | ☐ Yes / ☐ No — List: |
| Column additions? | ☐ Yes / ☐ No — List: |
| Column removals? | ☐ Yes / ☐ No — List: |
| Index changes? | ☐ Yes / ☐ No — List: |
| Data migration needed? | ☐ Yes / ☐ No |
| Migration reversible? | ☐ Yes / ☐ No |
| Estimated migration time | [duration] |
| Downtime required? | ☐ Yes / ☐ No |

**Migration files:**
- `db/migrations/XXXXXX_description.up.sql`
- `db/migrations/XXXXXX_description.down.sql`

---

### 4. Proto/API Changes Required?

| Question | Answer |
|----------|--------|
| Proto file changes? | ☐ Yes / ☐ No |
| New RPC methods? | ☐ Yes / ☐ No — List: |
| Modified messages? | ☐ Yes / ☐ No — List: |
| Breaking changes? | ☐ Yes / ☐ No |
| Version bump needed? | ☐ Yes / ☐ No (v1 → v2?) |
| REST endpoint changes? | ☐ Yes / ☐ No |
| New NATS subjects? | ☐ Yes / ☐ No — List: |
| Event payload changes? | ☐ Yes / ☐ No |

**Affected proto files:**
- [ ] `proto/search/v1/search.proto`
- [ ] `proto/reservation/v1/reservation.proto`
- [ ] `proto/billing/v1/billing.proto`
- [ ] `proto/payment/v1/payment.proto`
- [ ] `proto/presence/v1/presence.proto`
- [ ] `proto/notification/v1/notification.proto`

---

### 5. Resource Impact

| Resource | Before | After | Delta | Notes |
|----------|--------|-------|-------|-------|
| CPU (per service) | | | | |
| Memory (per service) | | | | |
| Storage (monthly growth) | | | | |
| Network (daily egress) | | | | |
| DB connections | | | | |
| Redis memory | | | | |
| NATS message volume | | | | |

**Infrastructure changes needed:**
- [ ] Additional container resources
- [ ] New infrastructure component
- [ ] Configuration changes
- [ ] Scaling policy update

---

### 6. Risk Assessment

#### Probability × Impact Matrix

| | Low Impact | Medium Impact | High Impact | Critical Impact |
|---|-----------|---------------|-------------|-----------------|
| **High Probability** | Medium Risk | High Risk | Critical Risk | Critical Risk |
| **Medium Probability** | Low Risk | Medium Risk | High Risk | Critical Risk |
| **Low Probability** | Low Risk | Low Risk | Medium Risk | High Risk |

#### Identified Risks

| # | Risk | Probability | Impact | Risk Level | Mitigation |
|---|------|-------------|--------|------------|------------|
| 1 | | L / M / H | L / M / H / C | | |
| 2 | | L / M / H | L / M / H / C | | |
| 3 | | L / M / H | L / M / H / C | | |

---

### 7. Rollback Plan

| Step | Action | Responsible | Duration |
|------|--------|-------------|----------|
| 1 | | | |
| 2 | | | |
| 3 | | | |

**Rollback triggers:**
- [ ] Error rate exceeds X%
- [ ] Latency P95 exceeds Xms
- [ ] Data inconsistency detected
- [ ] Customer-reported critical issue

**Rollback time estimate:** [minutes/hours]

**Data rollback strategy:**
- [ ] Database down migration
- [ ] Data restore from backup
- [ ] No data rollback needed (backward compatible)

---

### 8. Testing Requirements

| Test Type | Required? | Scope | Status |
|-----------|-----------|-------|--------|
| Unit tests | ☐ Yes / ☐ No | [affected packages] | |
| Integration tests | ☐ Yes / ☐ No | [service pairs] | |
| Load tests | ☐ Yes / ☐ No | [scenarios] | |
| Race condition tests | ☐ Yes / ☐ No | [concurrent paths] | |
| Contract tests (proto) | ☐ Yes / ☐ No | [changed protos] | |
| End-to-end tests | ☐ Yes / ☐ No | [user flows] | |
| Security tests | ☐ Yes / ☐ No | [auth/authz changes] | |
| Performance benchmarks | ☐ Yes / ☐ No | [critical paths] | |

---

### 9. Stakeholder Notification List

| Stakeholder | Notification Type | When | Channel |
|-------------|-------------------|------|---------|
| Product Owner | Approval request | Before implementation | Email/Slack |
| DevOps/SRE | Deployment coordination | Before deploy | Slack |
| QA Team | Test plan review | Before testing | Jira/Slack |
| Billing Team | If billing logic changes | Before deploy | Email |
| Security Team | If auth/security changes | Before implementation | Email |
| End Users | If UX/API changes | After deploy | Release notes |

---

### 10. Approval Signatures

| Role | Name | Decision | Date | Notes |
|------|------|----------|------|-------|
| Technical Lead | | Approved / Rejected | | |
| Product Owner | | Approved / Rejected | | |
| DevOps Lead | | Approved / Rejected | | |
| Security (if applicable) | | Approved / Rejected | | |

---

---

## Example: Filled-In Template — Payment Flow Migration

### Change Request Header

| Field | Value |
|-------|-------|
| **Change Request ID** | CR-007 |
| **Title** | Two-Phase Payment Flow Migration |
| **Requested By** | Product Owner |
| **Date Submitted** | 2026-05-07 |
| **Priority** | High |
| **Target Release** | v1.2.0 |
| **Status** | Implemented |

### 1. Change Description

**Summary:**  
Migrate from single-phase payment (pay at checkout only) to two-phase payment flow where drivers pay a booking fee at reservation confirmation and the remaining parking fee at checkout. This ensures commitment from drivers and reduces no-show rates.

**Business Justification:**  
High no-show rate (estimated 15%) causes revenue loss and blocks spots for other drivers. Booking fee creates financial commitment, reducing no-shows to estimated <5%.

**Scope:**  
- [x] Feature modification
- [ ] New feature

---

### 2. Affected Services Checklist

| Service | Affected? | Impact Level | Description of Changes |
|---------|-----------|--------------|----------------------|
| Gateway (:8080) | ☑ Yes | LOW | New route: `POST /api/v1/reservations/:id/confirm` |
| Search (:9092) | ☐ No | — | No changes |
| Reservation (:9091) | ☑ Yes | HIGH | New `ConfirmReservation` RPC, modified state machine (pending → confirmed requires payment) |
| Billing (:9093) | ☑ Yes | HIGH | New `StartBilling` RPC, booking fee tracking, split fee calculation |
| Payment (:9094) | ☑ Yes | MEDIUM | Process booking fee payment (existing `ProcessPayment` reused) |
| Presence (:9095) | ☐ No | — | No changes |
| Notification (:9096) | ☑ Yes | LOW | New event subscription: `reservation.confirmed` with payment details |

---

### 3. Database Migration Required?

| Question | Answer |
|----------|--------|
| Schema migration needed? | ☑ Yes |
| New tables? | ☐ No |
| Column additions? | ☑ Yes — `billing_records.booking_fee`, `reservations.confirmed_at` |
| Column removals? | ☐ No |
| Index changes? | ☐ No |
| Data migration needed? | ☑ Yes — Backfill `booking_fee = 0` for existing records |
| Migration reversible? | ☑ Yes — Column additions are backward compatible |
| Estimated migration time | < 5 seconds (small table) |
| Downtime required? | ☐ No |

**Migration files:**
- `db/migrations/000008_add_booking_fee.up.sql`
- `db/migrations/000008_add_booking_fee.down.sql`

---

### 4. Proto/API Changes Required?

| Question | Answer |
|----------|--------|
| Proto file changes? | ☑ Yes |
| New RPC methods? | ☑ Yes — `ConfirmReservation`, `StartBilling` |
| Modified messages? | ☑ Yes — `ReservationResponse` (added `confirmed_at`), `BillingResponse` (added `booking_fee`) |
| Breaking changes? | ☐ No (additive only) |
| Version bump needed? | ☐ No (backward compatible) |
| REST endpoint changes? | ☑ Yes — New `POST /api/v1/reservations/:id/confirm` |
| New NATS subjects? | ☐ No (reuses `reservation.confirmed`) |
| Event payload changes? | ☑ Yes — `reservation.confirmed` now includes `booking_fee` field |

**Affected proto files:**
- [x] `proto/reservation/v1/reservation.proto`
- [x] `proto/billing/v1/billing.proto`

---

### 5. Resource Impact

| Resource | Before | After | Delta | Notes |
|----------|--------|-------|-------|-------|
| CPU (Reservation) | 0.1 cores | 0.12 cores | +20% | Additional payment call per reservation |
| Memory (Billing) | 64 MB | 72 MB | +8 MB | Booking fee state tracking |
| Storage (monthly) | 500 MB | 520 MB | +20 MB | Additional billing_records rows |
| Network (daily) | 150 MB | 165 MB | +15 MB | Extra gRPC calls (Reservation → Payment) |
| DB connections | 25 | 25 | 0 | No change (pooled) |
| Redis memory | 64 MB | 64 MB | 0 | No change |
| NATS message volume | 1000/day | 1200/day | +200/day | Confirmation events now carry payment data |

**Infrastructure changes needed:**
- [ ] No infrastructure changes required (existing capacity sufficient)

---

### 6. Risk Assessment

| # | Risk | Probability | Impact | Risk Level | Mitigation |
|---|------|-------------|--------|------------|------------|
| 1 | Payment gateway timeout during confirmation | Medium | High | High | Idempotency key ensures retry safety; circuit breaker prevents cascade |
| 2 | Inconsistent state if payment succeeds but DB update fails | Low | Critical | High | Wrap in transaction; payment is idempotent; reconciliation job |
| 3 | Existing clients break on new required confirmation step | Medium | Medium | Medium | Backward compatible: old flow still works, confirmation is additive |
| 4 | Booking fee amount disputes | Low | Low | Low | Clear fee display in API response before confirmation |

---

### 7. Rollback Plan

| Step | Action | Responsible | Duration |
|------|--------|-------------|----------|
| 1 | Revert Gateway route (remove `/confirm` endpoint) | DevOps | 2 min |
| 2 | Deploy previous Reservation service image | DevOps | 3 min |
| 3 | Deploy previous Billing service image | DevOps | 3 min |
| 4 | Run down migration (remove `booking_fee` column) | DBA | 1 min |
| 5 | Verify all services healthy | On-call | 5 min |

**Rollback triggers:**
- [x] Error rate exceeds 5% on `/confirm` endpoint
- [x] Payment processing latency P95 exceeds 5000ms
- [x] Data inconsistency: confirmed reservations without billing records

**Rollback time estimate:** 15 minutes

**Data rollback strategy:**
- [x] Database down migration (column removal is safe, data preserved in payments table)

---

### 8. Testing Requirements

| Test Type | Required? | Scope | Status |
|-----------|-----------|-------|--------|
| Unit tests | ☑ Yes | `internal/reservation/usecase/`, `internal/billing/usecase/` | ✅ Done |
| Integration tests | ☑ Yes | Reservation → Billing → Payment flow | ✅ Done |
| Load tests | ☑ Yes | Confirmation endpoint under 100 concurrent users | ✅ Done |
| Race condition tests | ☑ Yes | Concurrent confirmations for same reservation | ✅ Done |
| Contract tests (proto) | ☑ Yes | `reservation.proto`, `billing.proto` | ✅ Done |
| End-to-end tests | ☑ Yes | Full reserve → confirm → check-in → check-out | ✅ Done |
| Security tests | ☐ No | No auth changes | N/A |
| Performance benchmarks | ☑ Yes | `BenchmarkConfirmReservation` | ✅ Done |

---

### 9. Stakeholder Notification List

| Stakeholder | Notification Type | When | Channel |
|-------------|-------------------|------|---------|
| Product Owner | Feature approval | 2026-05-06 | Design doc review |
| DevOps/SRE | Deployment plan | 2026-05-07 | Slack #deployments |
| QA Team | Test plan | 2026-05-07 | Jira ticket |
| Billing Team | Fee logic change | 2026-05-06 | Email + meeting |
| Mobile App Team | New API endpoint | 2026-05-07 | API changelog |

---

### 10. Approval Signatures

| Role | Name | Decision | Date | Notes |
|------|------|----------|------|-------|
| Technical Lead | [Approved] | Approved | 2026-05-07 | Idempotency approach validated |
| Product Owner | [Approved] | Approved | 2026-05-06 | Booking fee amount confirmed at Rp 5,000 |
| DevOps Lead | [Approved] | Approved | 2026-05-07 | No infrastructure changes needed |

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-05-13 | Engineering Team | Initial template + payment flow example |
