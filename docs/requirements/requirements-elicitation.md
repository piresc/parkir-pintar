# Requirements Elicitation Document

**Project:** ParkirPintar — Smart Parking Backend System  
**Version:** 1.0  
**Date:** 2026-05-13  
**Author:** Engineering Team  
**Status:** Approved

---

## 1. Stakeholder Identification

| ID | Stakeholder Group | Role | Key Concerns | Representative |
|----|-------------------|------|--------------|----------------|
| SH-01 | Parking Operators | Manage parking facility (5 floors, 400 spots) | Occupancy visibility, revenue tracking, penalty enforcement | Operations Manager |
| SH-02 | Drivers | End users who reserve and use parking spots | Easy booking, fair pricing, real-time availability | Mobile App Users |
| SH-03 | System Administrators | Maintain infrastructure and services | Uptime, observability, deployment, scaling | DevOps Engineer |
| SH-04 | Billing Team | Manage invoicing, fee calculation, payment reconciliation | Accurate billing, penalty handling, payment gateway reliability | Finance Manager |
| SH-05 | Product Owner | Define features and priorities | Feature completeness, user satisfaction, time-to-market | Product Manager |
| SH-06 | Security Team | Ensure data protection and access control | Authentication, authorization, data encryption, audit trails | Security Engineer |

---

## 2. Elicitation Techniques Used

| Technique | Applied To | Artifacts Produced |
|-----------|-----------|-------------------|
| **Stakeholder Interviews** | SH-01, SH-02, SH-04 | Interview transcripts, requirement statements |
| **Observation** | SH-02 (driver behavior at parking facilities) | User journey maps, pain points |
| **Prototyping** | SH-02 (mobile app flow) | Wireframes for reservation and payment flows |
| **Document Analysis** | Existing parking systems, competitor analysis | Feature comparison matrix |
| **Workshop (JAD)** | All stakeholders | Consolidated requirements list, priority matrix |
| **Use Case Modeling** | SH-01, SH-02 | Use case diagrams, sequence diagrams |

---

## 3. Interview Questions per Stakeholder Group

### 3.1 Parking Operators (SH-01)

1. How do you currently track spot occupancy across floors?
2. What is your process when a driver parks in the wrong spot?
3. How do you handle overnight parking and associated fees?
4. What reports do you need for daily/weekly/monthly operations?
5. How do you manage peak hours and full-capacity situations?
6. What is your current penalty enforcement process?
7. How do you communicate with drivers about violations?

### 3.2 Drivers (SH-02)

1. What frustrates you most about finding parking?
2. How far in advance do you typically need to reserve a spot?
3. What payment methods do you prefer (QRIS, e-wallet, credit card)?
4. How important is knowing exact spot location vs. just floor availability?
5. Would you prefer system-assigned spots or choosing your own?
6. What information do you need during check-in and check-out?
7. How do you expect to be notified about reservation status changes?

### 3.3 System Administrators (SH-03)

1. What is your acceptable downtime window for maintenance?
2. What monitoring and alerting tools do you currently use?
3. How do you handle database migrations in production?
4. What is your disaster recovery process?
5. How do you manage secrets and configuration across environments?
6. What are your scaling triggers and thresholds?

### 3.4 Billing Team (SH-04)

1. What is the pricing model (hourly, daily, overnight)?
2. How are penalties calculated (wrong spot, no-show, cancellation)?
3. What payment gateways need to be supported?
4. How do you handle refunds and disputes?
5. What reconciliation reports do you need?
6. How should booking fees relate to final parking fees?
7. What is the overnight fee threshold (hours after which overnight applies)?

---

## 4. Functional Requirements

### 4.1 Spot Search & Availability

| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|
| FR-001 | System shall display real-time spot availability per floor filtered by vehicle type (car/motorcycle) | Must | SH-02 |
| FR-002 | System shall provide a floor map showing individual spot status (available, occupied, reserved) | Must | SH-01, SH-02 |
| FR-003 | System shall return spot details including floor number, spot code, and vehicle type compatibility | Must | SH-02 |
| FR-004 | System shall cache availability data with invalidation on reservation state changes | Should | SH-03 |

### 4.2 Reservation Management

| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|
| FR-005 | System shall allow drivers to create reservations with system-assigned or user-selected spots | Must | SH-02 |
| FR-006 | System shall enforce idempotent reservation creation using idempotency keys | Must | SH-03 |
| FR-007 | System shall prevent double-booking via distributed locks (Redis SETNX) and database constraints | Must | SH-01, SH-03 |
| FR-008 | System shall support reservation confirmation with booking fee payment | Must | SH-04 |
| FR-009 | System shall automatically expire unconfirmed reservations after timeout (15 minutes) | Must | SH-01 |
| FR-010 | System shall allow reservation cancellation with appropriate cancellation fee | Must | SH-02, SH-04 |
| FR-011 | System shall support check-in marking for confirmed reservations | Must | SH-01 |
| FR-012 | System shall support check-out with automatic fee calculation and payment processing | Must | SH-02, SH-04 |
| FR-013 | System shall allow drivers to list their reservations filtered by status | Should | SH-02 |

### 4.3 Billing & Invoicing

| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|
| FR-014 | System shall calculate parking fees based on actual session duration (hourly rate) | Must | SH-04 |
| FR-015 | System shall generate idempotent invoices for each reservation | Must | SH-04 |
| FR-016 | System shall apply penalties for wrong-spot parking and cancellations | Must | SH-01, SH-04 |
| FR-017 | System shall apply overnight flat fee when parking exceeds threshold | Must | SH-04 |
| FR-018 | System shall track booking fee, parking fee, overnight fee, and penalty amounts separately | Should | SH-04 |

### 4.4 Payment Processing

| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|
| FR-019 | System shall process payments via configurable payment gateway | Must | SH-04 |
| FR-020 | System shall support QRIS payment method | Must | SH-02 |
| FR-021 | System shall support multiple payment methods (QRIS, credit card, debit, e-wallet) | Should | SH-02 |
| FR-022 | System shall support payment refunds for cancelled reservations | Must | SH-02, SH-04 |
| FR-023 | System shall provide payment status queries | Must | SH-02 |
| FR-024 | System shall ensure idempotent payment processing | Must | SH-03 |

### 4.5 Presence & Location

| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|
| FR-025 | System shall receive streaming location updates from driver mobile app | Must | SH-02 |
| FR-026 | System shall detect driver arrival within parking geofence | Must | SH-01 |
| FR-027 | System shall detect wrong-spot parking based on GPS coordinates | Must | SH-01 |
| FR-028 | System shall provide latest known location for active reservations | Should | SH-01 |

### 4.6 Notifications

| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|
| FR-029 | System shall send push notifications for reservation status changes | Must | SH-02 |
| FR-030 | System shall send SMS notifications for critical events (payment confirmation, penalties) | Should | SH-02 |
| FR-031 | System shall send email notifications for invoices and receipts | Should | SH-02 |
| FR-032 | System shall support notification templates with dynamic data | Could | SH-01 |

### 4.7 Analytics (Future)

| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|
| FR-033 | System shall track occupancy metrics per floor over time | Could | SH-01 |
| FR-034 | System shall generate revenue reports per period | Could | SH-04 |
| FR-035 | System shall track peak usage patterns | Could | SH-01 |

---

## 5. Non-Functional Requirements

### 5.1 Performance

| ID | Requirement | Target | Measurement |
|----|-------------|--------|-------------|
| NFR-001 | API response time (P95) | < 200ms | Prometheus histogram |
| NFR-002 | API response time (P99) | < 500ms | Prometheus histogram |
| NFR-003 | Search query response (cached) | < 50ms | Redis cache hit latency |
| NFR-004 | Reservation creation (end-to-end) | < 1000ms | Distributed trace duration |
| NFR-005 | Payment processing | < 3000ms | External gateway + internal processing |
| NFR-006 | Event delivery latency (NATS) | < 100ms | Consumer lag metric |

### 5.2 Scalability

| ID | Requirement | Target | Strategy |
|----|-------------|--------|----------|
| NFR-007 | Concurrent users | 1000 | Stateless services, horizontal scaling |
| NFR-008 | Requests per second (gateway) | 100 req/s | Rate limiting + auto-scaling |
| NFR-009 | Database connections per service | 25 max | Connection pooling (pgx) |
| NFR-010 | Message throughput (NATS) | 10,000 msg/s | JetStream with file storage |
| NFR-011 | Parking capacity | 400 spots (5 floors) | Schema supports multi-facility expansion |

### 5.3 Security

| ID | Requirement | Implementation |
|----|-------------|----------------|
| NFR-012 | Transport encryption | TLS 1.3 (Traefik termination) |
| NFR-013 | Authentication | JWT (HMAC-SHA256) validated at Gateway |
| NFR-014 | Authorization | Driver ID from token claims, resource ownership checks |
| NFR-015 | Rate limiting | Per-IP token bucket (100 req/min) |
| NFR-016 | Input validation | Protobuf schema + handler-level validation |
| NFR-017 | SQL injection prevention | Parameterized queries via sqlx |
| NFR-018 | CORS policy | Explicit origin allowlist (no wildcard) |
| NFR-019 | Secret management | Environment variables, never in code |
| NFR-020 | SSRF protection | Internal network range blocking in HTTP client |

### 5.4 Availability & Reliability

| ID | Requirement | Target | Mechanism |
|----|-------------|--------|-----------|
| NFR-021 | System availability | 99.9% (8.76h downtime/year) | Health checks, auto-restart |
| NFR-022 | Data durability | No data loss on single node failure | PostgreSQL WAL, NATS file storage |
| NFR-023 | Graceful degradation | Non-critical failures don't cascade | Circuit breaker, fallback responses |
| NFR-024 | Recovery time (RTO) | < 5 minutes | Container restart, health probes |
| NFR-025 | Recovery point (RPO) | < 1 minute | WAL streaming, event replay |

### 5.5 Observability

| ID | Requirement | Implementation |
|----|-------------|----------------|
| NFR-026 | Distributed tracing | OpenTelemetry → Tempo |
| NFR-027 | Metrics collection | OTel Meter → Prometheus |
| NFR-028 | Centralized logging | slog → OTel Log Bridge → Loki |
| NFR-029 | Alerting | Prometheus alerting rules (8 rules) → Alertmanager |
| NFR-030 | Dashboard | Grafana with cross-linked datasources |

---

## 6. Requirements Prioritization (MoSCoW Method)

### Must Have (Release 1.0)

| ID | Requirement |
|----|-------------|
| FR-001 | Real-time spot availability |
| FR-005 | Reservation creation (system/user-assigned) |
| FR-007 | Double-booking prevention |
| FR-008 | Reservation confirmation with payment |
| FR-009 | Auto-expiry of unconfirmed reservations |
| FR-011 | Check-in |
| FR-012 | Check-out with billing |
| FR-014 | Fee calculation |
| FR-015 | Invoice generation |
| FR-019 | Payment processing |
| FR-020 | QRIS support |
| FR-022 | Refunds |
| FR-025 | Location streaming |
| FR-026 | Arrival detection |
| FR-027 | Wrong-spot detection |
| FR-029 | Push notifications |

### Should Have (Release 1.1)

| ID | Requirement |
|----|-------------|
| FR-002 | Floor map visualization |
| FR-004 | Cache invalidation |
| FR-010 | Cancellation with fee |
| FR-013 | Reservation listing |
| FR-016 | Penalty application |
| FR-017 | Overnight fee |
| FR-021 | Multiple payment methods |
| FR-028 | Latest location query |
| FR-030 | SMS notifications |
| FR-031 | Email notifications |

### Could Have (Release 2.0)

| ID | Requirement |
|----|-------------|
| FR-032 | Notification templates |
| FR-033 | Occupancy analytics |
| FR-034 | Revenue reports |
| FR-035 | Peak usage patterns |

### Won't Have (This Release)

- Multi-facility support (single parking area for now)
- Dynamic pricing based on demand
- Loyalty/rewards program
- Third-party parking aggregator integration
- EV charging spot management

---

## 7. Traceability Matrix

| Requirement | Implementing Service | Proto/Handler | Database Table |
|-------------|---------------------|---------------|----------------|
| FR-001 | Search Service (:9092) | `search.v1.SearchService/GetAvailability` | `parking_spots` (read) |
| FR-002 | Search Service (:9092) | `search.v1.SearchService/GetFloorMap` | `parking_spots` (read) |
| FR-003 | Search Service (:9092) | `search.v1.SearchService/GetSpotDetails` | `parking_spots` (read) |
| FR-005 | Reservation Service (:9091) | `reservation.v1.ReservationService/CreateReservation` | `reservations`, `parking_spots` |
| FR-006 | Reservation Service (:9091) | Idempotency middleware | `reservations` (unique constraint) |
| FR-007 | Reservation Service (:9091) | Redis SETNX + SELECT FOR UPDATE | `parking_spots`, Redis |
| FR-008 | Reservation Service (:9091) | `reservation.v1.ReservationService/ConfirmReservation` | `reservations` |
| FR-009 | Reservation Service (:9091) | `internal/reservation/worker/expiry.go` | `reservations` |
| FR-010 | Reservation Service (:9091) | `reservation.v1.ReservationService/CancelReservation` | `reservations` |
| FR-011 | Reservation Service (:9091) | `reservation.v1.ReservationService/CheckIn` | `reservations` |
| FR-012 | Reservation Service (:9091) | `reservation.v1.ReservationService/CheckOut` | `reservations` |
| FR-014 | Billing Service (:9093) | `billing.v1.BillingService/CalculateFee` | `billing_records` |
| FR-015 | Billing Service (:9093) | `billing.v1.BillingService/GenerateInvoice` | `billing_records` |
| FR-016 | Billing Service (:9093) | `billing.v1.BillingService/ApplyPenalty` | `billing_records`, `penalties` |
| FR-017 | Billing Service (:9093) | `billing.v1.BillingService/ApplyOvernightFee` | `billing_records` |
| FR-019 | Payment Service (:9094) | `payment.v1.PaymentService/ProcessPayment` | `payments` |
| FR-020 | Payment Service (:9094) | `payment.v1.PaymentService/ProcessQRIS` | `payments` |
| FR-022 | Payment Service (:9094) | `payment.v1.PaymentService/RefundPayment` | `payments` |
| FR-023 | Payment Service (:9094) | `payment.v1.PaymentService/GetPaymentStatus` | `payments` |
| FR-025 | Presence Service (:9095) | `presence.v1.PresenceService/StreamLocation` | `presence_logs` |
| FR-026 | Presence Service (:9095) | `presence.v1.PresenceService/DetectArrival` | `presence_logs` |
| FR-027 | Presence Service (:9095) | `presence.v1.PresenceService/DetectWrongSpot` | `presence_logs` |
| FR-029 | Notification Service (:9096) | `notification.v1.NotificationService/SendPush` | None (stub) |
| FR-030 | Notification Service (:9096) | `notification.v1.NotificationService/SendSMS` | None (stub) |
| FR-031 | Notification Service (:9096) | `notification.v1.NotificationService/SendEmail` | None (stub) |

---

## 8. Sign-Off

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Product Owner | _________________ | _________________ | ____/____/2026 |
| Technical Lead | _________________ | _________________ | ____/____/2026 |
| Operations Manager | _________________ | _________________ | ____/____/2026 |
| Finance Manager | _________________ | _________________ | ____/____/2026 |
| Security Engineer | _________________ | _________________ | ____/____/2026 |

### Approval Notes

- All Must Have requirements approved for Release 1.0
- Should Have requirements approved for Release 1.1 backlog
- Could Have requirements deferred to Release 2.0 planning
- Non-functional requirements accepted as baseline targets

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-05-13 | Engineering Team | Initial requirements elicitation |
