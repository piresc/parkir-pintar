# Product Requirements Document (PRD)
## ParkirPintar — Smart Parking Marketplace

**Version:** 1.0  
**Date:** April 24, 2026  
**Status:** Draft  
**Assessment:** Solution Development Assessment 2026

---

## Table of Contents

1. [Overview](#1-overview)
2. [Problem Statement](#2-problem-statement)
3. [Product Vision](#3-product-vision)
4. [Scope & Constraints](#4-scope--constraints)
5. [User Personas](#5-user-personas)
6. [Functional Requirements](#6-functional-requirements)
7. [Parking Area Specification](#7-parking-area-specification)
8. [Reservation & Booking Flow](#8-reservation--booking-flow)
9. [Pricing & Billing Rules](#9-pricing--billing-rules)
10. [Cancellation & Penalty Policy](#10-cancellation--penalty-policy)
11. [Payment Requirements](#11-payment-requirements)
12. [Location & Presence Requirements](#12-location--presence-requirements)
13. [Architecture & Technical Requirements](#13-architecture--technical-requirements)
14. [Microservices Breakdown](#14-microservices-breakdown)
15. [Reusable Components](#15-reusable-components)
16. [Data Model (ERD)](#16-data-model-erd)
17. [Testing Requirements](#17-testing-requirements)
18. [Non-Functional Requirements](#18-non-functional-requirements)
19. [Assumptions](#19-assumptions)
20. [Deliverables](#20-deliverables)
21. [Glossary](#21-glossary)

---

## 1. Overview

ParkirPintar is a lightweight, fast smart parking backend system that manages a single centralized parking area. It enables Drivers to view real-time parking availability, reserve spots, navigate to assigned spots, and pay for parking sessions — all through a mini app inside a super app (or as a standalone service).

The key differentiator: ParkirPintar does **not** lock a fixed price at booking time. Instead, billing is calculated from the **actual parking session duration**, aligned with the reservation window and applicable fee rules.

---

## 2. Problem Statement

Drivers waste time circling parking areas looking for available spots. Existing parking apps are often bloated, slow, and lock pricing upfront regardless of actual usage. There is a need for a lite, simple, and fast parking reservation system that:

- Shows real-time availability for a single parking area
- Allows quick spot reservation with minimal friction
- Bills fairly based on actual parking duration
- Prevents double-booking and spot conflicts

---

## 3. Product Vision

Deliver the fastest, simplest parking reservation experience — one tap to reserve, automatic arrival detection, and fair pay-per-use billing. ParkirPintar operates a single centralized parking inventory with no Host onboarding complexity.

---

## 4. Scope & Constraints

### In Scope
- Single parking area management (one building, centralized inventory)
- Driver-facing reservation, check-in, check-out, and billing
- Real-time availability view
- Two spot assignment modes (system-assigned and user-selected)
- Payment gateway integration (including QRIS)
- Sensor-based spot occupancy verification
- Presence streaming during active sessions

### Out of Scope
- Host onboarding or spot publishing workflows
- Multi-area search radius or multi-location support
- Driver-to-Driver marketplace features
- Mobile app frontend (backend only)
- Physical hardware integration (barriers, sensors) — presence is smartphone-based

---

## 5. User Personas

### Driver
- Primary user of the system
- Uses a smartphone with location services enabled
- Needs to find and reserve parking quickly
- Wants transparent, usage-based billing
- May drive a car or motorcycle

---

## 6. Functional Requirements

### FR-01: View Parking Availability
- The system shall display real-time availability of the parking area, broken down by floor and vehicle type (car / motorcycle).
- Availability data must reflect current reservations, active sessions, and locked spots.

### FR-02: Reserve a Spot
- **System-Assigned Mode (Fastest):** The Driver taps "Reserve" and the system immediately assigns any available spot matching the vehicle type.
- **User-Selected Mode:** The Driver browses available spots and selects a specific one. During selection, the system applies a short hold/queue mechanism to prevent conflicts.
- On reservation, the system validates capacity and availability, locks the inventory briefly to prevent double-booking, and confirms the reservation.

### FR-03: Reservation Hold & Expiry
- A confirmed reservation holds the assigned spot for **1 hour**.
- If the Driver does not check in within 1 hour after confirmation, the reservation expires automatically and the spot is released.

### FR-04: Check-In
- Check-in occurs when the Driver arrives and manually checks in; the system verifies via spot sensor.
- Billing starts at check-in time.

### FR-05: Check-Out
- The Driver initiates check-out (or is prompted).
- The system calculates the final bill based on actual session duration and applicable rules.
- Payment is processed via the payment gateway.

### FR-06: Wrong-Spot Detection & Penalty
- If the Driver parks in a spot different from the reserved/assigned spot, a penalty of **200,000 IDR** is applied.

### FR-07: Cancellation
- Cancellation rules are time-based relative to confirmation (see Section 10).

### FR-08: Notifications (Stub)
- The system shall send notifications for: reservation confirmation, upcoming expiry warning, check-in confirmation, billing summary, cancellation confirmation, penalty alerts.
- Notification service may be implemented as a stub.

---

## 7. Parking Area Specification

| Attribute         | Value                          |
|-------------------|--------------------------------|
| Building          | Single parking building        |
| Floors            | 5                              |
| Car capacity/floor| 30                             |
| Motorcycle capacity/floor | 50                    |
| **Total car capacity** | **150**                   |
| **Total motorcycle capacity** | **250**           |
| **Total capacity** | **400 spots**                 |

### Spot Identification
Each spot should be uniquely identifiable by: `Floor` + `Vehicle Type` + `Spot Number`  
Example: `F3-C-012` (Floor 3, Car, Spot 12), `F1-M-045` (Floor 1, Motorcycle, Spot 45)

---

## 8. Reservation & Booking Flow

### 8.1 System-Assigned Flow (Fastest)

```
Driver opens app
  → System detects Driver location
  → Driver selects vehicle type (car/motorcycle)
  → Driver taps "Reserve"
  → System checks availability
  → System locks inventory (Redis distributed lock)
  → System assigns first available spot matching vehicle type
  → Reservation confirmed → Booking fee charged (5,000 IDR)
  → Spot held for 1 hour
  → Driver navigates to parking area & assigned spot
  → Driver arrives → Geofence auto-detects OR manual check-in
  → Billing timer starts
  → Driver parks in assigned spot
  → ... parking session ...
  → Driver checks out
  → System calculates final bill (session duration + applicable fees)
  → Payment processed (QRIS / payment gateway)
  → Spot released back to inventory
```

### 8.2 User-Selected Flow

```
Driver opens app
  → System detects Driver location
  → Driver browses available spots (by floor, vehicle type)
  → Driver selects a specific spot
  → System applies short hold/queue on selected spot (prevents conflicts)
  → System validates availability & locks inventory
  → Reservation confirmed → Booking fee charged (5,000 IDR)
  → (same flow as system-assigned from here)
```

### 8.3 Reservation States

```
PENDING → CONFIRMED → CHECKED_IN → CHECKED_OUT
                    → EXPIRED (no-show after 1 hour)
                    → CANCELLED
```

| State        | Description                                                |
|--------------|------------------------------------------------------------|
| PENDING      | Reservation request received, inventory lock in progress   |
| CONFIRMED    | Spot assigned and held; booking fee charged                |
| CHECKED_IN   | Driver has arrived and parked; billing active              |
| CHECKED_OUT  | Session ended; final bill calculated and paid              |
| EXPIRED      | Driver did not check in within 1 hour; spot released       |
| CANCELLED    | Driver cancelled the reservation                           |

---

## 9. Pricing & Billing Rules

### 9.1 Booking Fee
| Item        | Amount       |
|-------------|-------------|
| Booking fee | 5,000 IDR per successful reservation (charged at confirmation) |

### 9.2 Parking Rate (Time-Based)
| Duration                  | Rate          |
|---------------------------|---------------|
| First hour                | 5,000 IDR     |
| Each subsequent started hour | 5,000 IDR  |

- Billing is calculated from **actual parking session time** (check-in to check-out).
- Any started hour is billed as a full hour (ceiling-based).
- Rate is **not** locked at booking time — it is computed at checkout.

### 9.3 Overnight Fee
| Condition                                      | Fee          |
|------------------------------------------------|-------------|
| Session crosses midnight (overnight parking)   | Flat 20,000 IDR |

- Applied once per overnight crossing, in addition to hourly charges.

### 9.4 Overstay Policy
- Drivers **may** stay beyond the reserved end time.
- There is **no overstay penalty**.
- Additional time beyond the reservation window is billed at the **same standard hourly rate** (5,000 IDR per started hour).

### 9.5 Billing Calculation Examples

**Example 1: Standard 2-hour parking**
- Check-in: 10:00, Check-out: 12:00
- Duration: 2 hours
- Parking fee: 2 × 5,000 = 10,000 IDR
- Booking fee: 5,000 IDR
- **Total: 15,000 IDR**

**Example 2: 1.5-hour parking (partial hour rounds up)**
- Check-in: 14:00, Check-out: 15:30
- Duration: 1.5 hours → rounds up to 2 hours
- Parking fee: 2 × 5,000 = 10,000 IDR
- Booking fee: 5,000 IDR
- **Total: 15,000 IDR**

**Example 3: Overnight parking**
- Check-in: 22:00 (Day 1), Check-out: 06:00 (Day 2)
- Duration: 8 hours
- Parking fee: 8 × 5,000 = 40,000 IDR
- Overnight fee: 20,000 IDR
- Booking fee: 5,000 IDR
- **Total: 65,000 IDR**

**Example 4: Overstay (reserved 2 hours, stayed 4 hours)**
- Reserved: 10:00–12:00, Actual check-out: 14:00
- Duration: 4 hours
- Parking fee: 4 × 5,000 = 20,000 IDR
- Overstay penalty: 0 IDR
- Booking fee: 5,000 IDR
- **Total: 25,000 IDR**

---

## 10. Cancellation & Penalty Policy

### 10.1 Cancellation Fees

| Condition                                          | Fee          |
|----------------------------------------------------|-------------|
| Cancel within 2 minutes after confirmation         | 0 IDR (free cancellation) |
| Cancel after 2 minutes, before check-in            | 5,000 IDR   |
| No-show (not checked in within 1 hour)             | Booking fee (5,000 IDR) consumed; spot released |

### 10.2 Wrong-Spot Penalty

| Condition                                          | Penalty      |
|----------------------------------------------------|-------------|
| Driver parks in a different spot than assigned      | 200,000 IDR |

---

## 11. Payment Requirements

### 11.1 Payment Gateway
- The system must integrate with a payment gateway supporting **QRIS** payments.
- Payment is processed at checkout (for parking fees) and at confirmation (for booking fees).

### 11.2 Payment Scenarios
- **Booking fee:** Charged immediately upon reservation confirmation (5,000 IDR).
- **Cancellation fee:** Charged when applicable per cancellation policy.
- **No-show:** Booking fee (5,000 IDR) already collected at confirmation is consumed; no additional fee is charged.
- **Wrong-spot penalty:** Charged when detected (200,000 IDR).
- **Parking session fee:** Charged at checkout (hourly rate + overnight fee if applicable).

### 11.3 Payment States
```
PENDING → SUCCESS
        → FAILED → RETRY
        → REFUNDED
```

### 11.4 Idempotency
- `CreateReservation` and `Checkout/Invoice` operations must be **idempotent** to prevent duplicate charges.

---

## 12. Location & Presence Requirements

### 12.1 Location Updates
- The Driver app sends location updates at least every **30 seconds** (or faster) while a session is active.
- Location data supports arrival detection and presence streaming.

### 12.2 Geofence
- The system uses spot sensors to verify vehicle presence at the assigned parking spot.
- Geofence triggers automatic check-in when the Driver enters the parking area boundary.

### 12.3 Presence Streaming
- Active sessions stream Driver presence data to the backend.
- Presence data is used for real-time occupancy tracking and wrong-spot detection.

---

## 13. Architecture & Technical Requirements

### 13.1 Communication Protocol
- All service-to-service communication must use **gRPC over HTTP/2**.

### 13.2 Language
- All services must be written in **Go (Golang)**.

### 13.3 Deployment
- The system is set up as a **mini app inside a super app** or as a **standalone service**.
- Containerized deployment (Docker).

### 13.4 Consistency & Double-Booking Prevention
- The system must prevent double-booking for the same spot and overlapping time windows.
- Use **Redis-based distributed locks** for inventory locking during reservation.

### 13.5 Idempotency
- `CreateReservation` and `Checkout/Invoice` operations must implement an **idempotency mechanism** (e.g., idempotency keys stored in Redis/DB with TTL).

### 13.6 Resilience & Availability
- **Retries:** Implement retry logic with exponential backoff for transient failures.
- **Timeouts:** Define per-service and per-RPC timeouts to prevent cascading delays.
- **Circuit Breakers:** Apply circuit breaker patterns on calls to external services (payment gateway, notification) to prevent cascade failures.
- **Graceful Degradation:** When non-core services (notification, presence) fail, core flows (reservation, billing, payment) must continue to function.

### 13.7 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Super App / Client                       │
└──────────────────────────────┬──────────────────────────────────┘
                               │ REST / gRPC-Web
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                        API Gateway Service                       │
│         (Auth Interceptor, Rate Limiting, Routing)               │
└──────┬────────┬────────┬────────┬────────┬────────┬─────────────┘
       │        │        │        │        │        │
       │ gRPC   │ gRPC   │ gRPC   │ gRPC   │ gRPC   │ gRPC
       ▼        ▼        ▼        ▼        ▼        ▼
┌────────┐ ┌────────┐ ┌──────────┐ ┌───────┐ ┌────────┐ ┌──────────────┐
│ Search │ │Reserve │ │ Billing  │ │Payment│ │Presence│ │ Notification │
│Service │ │Service │ │ Service  │ │Service│ │Service │ │  Service     │
│        │ │        │ │          │ │       │ │        │ │  (Stub)      │
└───┬────┘ └───┬────┘ └────┬─────┘ └───┬───┘ └───┬────┘ └──────────────┘
    │          │           │           │         │
    ▼          ▼           ▼           ▼         ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Data Layer                                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  ┌─────────────┐  │
│  │PostgreSQL│  │  Redis    │  │Message Queue │  │  Object     │  │
│  │ (Primary │  │ (Cache,   │  │ (Async jobs, │  │  Storage    │  │
│  │  Store)  │  │  Locks,   │  │  Events)     │  │  (if needed)│  │
│  │          │  │  Presence)│  │              │  │             │  │
│  └──────────┘  └──────────┘  └──────────────┘  └─────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 13.8 Low-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          API GATEWAY SERVICE                            │
│  ┌─────────────┐ ┌──────────────┐ ┌────────────┐ ┌──────────────────┐  │
│  │JWT/mTLS Auth│ │Rate Limiter  │ │Request     │ │gRPC-Web to gRPC  │  │
│  │Interceptor  │ │(Token Bucket)│ │Validator   │ │Transcoder        │  │
│  └─────────────┘ └──────────────┘ └────────────┘ └──────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            ▼                       ▼                       ▼
┌───────────────────┐  ┌────────────────────┐  ┌────────────────────────┐
│  SEARCH SERVICE   │  │RESERVATION SERVICE │  │   BILLING SERVICE      │
│                   │  │                    │  │                        │
│ • GetAvailability │  │ • CreateReservation│  │ • StartBilling         │
│ • GetFloorMap     │  │   (idempotent)     │  │ • CalculateFee         │
│ • GetSpotDetails  │  │ • CancelReservation│  │ • GenerateInvoice      │
│                   │  │ • CheckIn          │  │   (idempotent)         │
│ Reads from:       │  │ • CheckOut         │  │ • ApplyPenalty         │
│ - PostgreSQL      │  │ • ExpireReservation│  │ • ApplyOvernightFee    │
│ - Redis (cache)   │  │                    │  │                        │
│                   │  │ Uses:              │  │ Uses:                  │
│                   │  │ - Redis Lock       │  │ - Pricing Engine       │
│                   │  │ - PostgreSQL       │  │ - PostgreSQL           │
│                   │  │ - Event Publisher  │  │ - Event Publisher      │
└───────────────────┘  └────────────────────┘  └────────────────────────┘
            ┌───────────────────────┼───────────────────────┐
            ▼                       ▼                       ▼
┌───────────────────┐  ┌────────────────────┐  ┌────────────────────────┐
│ PAYMENT SERVICE   │  │ PRESENCE SERVICE   │  │ NOTIFICATION SERVICE   │
│                   │  │                    │  │ (Stub)                 │
│ • ProcessPayment  │  │ • StreamLocation   │  │                        │
│ • ProcessQRIS     │  │ • DetectArrival    │  │ • SendPush             │
│ • RefundPayment   │  │   (Geofence)       │  │ • SendSMS              │
│ • GetPaymentStatus│  │ • DetectWrongSpot  │  │ • SendEmail            │
│                   │  │ • GetPresence      │  │                        │
│ Integrates:       │  │                    │  │ Integrates:            │
│ - Payment Gateway │  │ Uses:              │  │ - Push Service (stub)  │
│ - QRIS Provider   │  │ - Redis (stream)   │  │ - SMS Gateway (stub)   │
│                   │  │ - Geofence Engine  │  │                        │
└───────────────────┘  └────────────────────┘  └────────────────────────┘
```

### 13.9 Service Communication Flow

```
Reserve Spot Flow:
  Client → Gateway → ReservationService.CreateReservation()
    → Redis: AcquireLock(spot_id)
    → PostgreSQL: CheckAvailability + InsertReservation
    → Redis: ReleaseLock(spot_id)
    → BillingService: ChargeBookingFee()
    → PaymentService: ProcessPayment(booking_fee)
    → NotificationService: SendConfirmation()
    → Return ReservationConfirmation to Client

Check-In Flow:
  PresenceService queries spot sensor for occupancy
    → ReservationService.CheckIn(reservation_id)
    → PostgreSQL: UpdateReservationStatus(CHECKED_IN)
    → BillingService.StartBilling(reservation_id)
    → NotificationService: SendCheckInConfirmation()

Check-Out Flow:
  Client → Gateway → ReservationService.CheckOut()
    → BillingService.CalculateFee(reservation_id)
      → PricingEngine: ComputeHourlyRate + OvernightFee + Penalties
    → BillingService.GenerateInvoice() (idempotent)
    → PaymentService.ProcessPayment(total_amount)
    → PostgreSQL: UpdateReservationStatus(CHECKED_OUT)
    → Redis: ReleaseSpot(spot_id)
    → NotificationService: SendReceipt()

Reservation Expiry Flow (Background Job):
  Scheduler checks CONFIRMED reservations older than 1 hour
    → ReservationService.ExpireReservation(reservation_id)
    → Booking fee (5,000 IDR) already collected at confirmation is not refunded
    → Redis: ReleaseSpot(spot_id)
    → NotificationService: SendExpiryNotice()
```

---

## 14. Microservices Breakdown

### 14.1 Gateway Service
| Attribute       | Detail                                                    |
|-----------------|-----------------------------------------------------------|
| Responsibility  | API entry point, authentication, rate limiting, routing   |
| Protocol (in)   | REST / gRPC-Web (client-facing)                          |
| Protocol (out)  | gRPC over HTTP/2 (to internal services)                  |
| Key Components  | Auth interceptor (JWT/mTLS), rate limiter, request validator |

### 14.2 Search Service
| Attribute       | Detail                                                    |
|-----------------|-----------------------------------------------------------|
| Responsibility  | Real-time availability queries, floor maps, spot details  |
| Data Sources    | PostgreSQL (source of truth), Redis (cache layer)         |
| Key RPCs        | `GetAvailability`, `GetFloorMap`, `GetSpotDetails`        |

### 14.3 Reservation Service
| Attribute       | Detail                                                    |
|-----------------|-----------------------------------------------------------|
| Responsibility  | Spot reservation, check-in, check-out, cancellation, expiry |
| Data Sources    | PostgreSQL, Redis (distributed locks)                     |
| Key RPCs        | `CreateReservation` (idempotent), `CancelReservation`, `CheckIn`, `CheckOut`, `ExpireReservation` |
| Critical Logic  | Double-booking prevention, inventory locking, hold timer  |

### 14.4 Billing Service
| Attribute       | Detail                                                    |
|-----------------|-----------------------------------------------------------|
| Responsibility  | Fee calculation, invoice generation, penalty application  |
| Data Sources    | PostgreSQL                                                |
| Key RPCs        | `StartBilling`, `CalculateFee`, `GenerateInvoice` (idempotent), `ApplyPenalty`, `ApplyOvernightFee` |
| Key Components  | Pricing Engine (reusable)                                 |

### 14.5 Payment Service
| Attribute       | Detail                                                    |
|-----------------|-----------------------------------------------------------|
| Responsibility  | Payment processing, QRIS integration, refunds            |
| Integrations    | Payment gateway, QRIS provider                           |
| Key RPCs        | `ProcessPayment`, `ProcessQRIS`, `RefundPayment`, `GetPaymentStatus` |

### 14.6 Presence Service
| Attribute       | Detail                                                    |
|-----------------|-----------------------------------------------------------|
| Responsibility  | Sensor-based spot occupancy verification, wrong-spot detection |
| Data Sources    | Redis (streams/pub-sub)                                   |
| Key RPCs        | `StreamLocation` (streaming RPC), `DetectArrival`, `DetectWrongSpot`, `GetPresence` |
| Update Frequency| At least every 30 seconds                                |

### 14.7 Notification Service (Stub)
| Attribute       | Detail                                                    |
|-----------------|-----------------------------------------------------------|
| Responsibility  | Push notifications, SMS, email (stub implementation)      |
| Key RPCs        | `SendPush`, `SendSMS`, `SendEmail`                        |
| Note            | Implemented as a stub; logs notification payloads         |

---

## 15. Reusable Components

| Component              | Description                                                       |
|------------------------|-------------------------------------------------------------------|
| Auth Interceptor       | gRPC interceptor for JWT validation or mTLS certificate verification |
| Pricing Engine         | Encapsulates all pricing rules: hourly rate, overnight fee, penalties, booking fee, cancellation fee |
| Redis Lock             | Distributed lock implementation for inventory locking during reservation |
| Presence Streamer      | Streaming presence service for real-time location updates via gRPC streaming |
| Config Loader          | Centralized configuration loading from environment variables / config files |
| Structured Logger      | Production-grade structured logging (e.g., zerolog / zap) with trace correlation |
| Tracing Middleware      | Distributed tracing integration (OpenTelemetry) for cross-service request tracking |
| Idempotency Middleware | Middleware to enforce idempotency on critical operations using idempotency keys |
| Circuit Breaker        | Wrapper for external service calls with circuit breaker pattern   |

---

## 16. Data Model (ERD)

### 16.1 Entity Relationship Diagram

```
┌──────────────────┐       ┌──────────────────────┐       ┌──────────────────┐
│     drivers       │       │    reservations       │       │   parking_spots   │
├──────────────────┤       ├──────────────────────┤       ├──────────────────┤
│ id (PK, UUID)    │──┐    │ id (PK, UUID)        │    ┌──│ id (PK, UUID)    │
│ name             │  │    │ driver_id (FK)       │◄───┘  │ floor_number     │
│ phone            │  │    │ spot_id (FK)         │───┘   │ spot_number      │
│ email            │  └───►│ vehicle_type         │       │ vehicle_type     │
│ vehicle_type     │       │ assignment_mode      │       │ spot_code        │
│ vehicle_plate    │       │ status               │       │ status           │
│ created_at       │       │ idempotency_key      │       │ created_at       │
│ updated_at       │       │ confirmed_at         │       │ updated_at       │
└──────────────────┘       │ expires_at           │       └──────────────────┘
                           │ checked_in_at        │
                           │ checked_out_at       │       ┌──────────────────┐
                           │ cancelled_at         │       │   billing_records │
                           │ created_at           │       ├──────────────────┤
                           │ updated_at           │    ┌──│ id (PK, UUID)    │
                           └──────────┬───────────┘    │  │ reservation_id   │
                                      │                │  │ (FK, UNIQUE)     │
                                      └────────────────┘  │ booking_fee      │
                                                          │ parking_fee      │
┌──────────────────┐       ┌──────────────────────┐       │ overnight_fee    │
│    payments       │       │     penalties         │       │ cancellation_fee │
├──────────────────┤       ├──────────────────────┤       │ penalty_amount   │
│ id (PK, UUID)    │       │ id (PK, UUID)        │       │ total_amount     │
│ billing_id (FK)  │       │ reservation_id (FK)  │       │ duration_minutes │
│ amount           │       │ penalty_type         │       │ billed_hours     │
│ payment_method   │       │ amount               │       │ is_overnight     │
│ payment_gateway  │       │ description          │       │ idempotency_key  │
│ transaction_ref  │       │ created_at           │       │ status           │
│ idempotency_key  │       └──────────────────────┘       │ created_at       │
│ status           │                                      │ updated_at       │
│ paid_at          │                                      └──────────────────┘
│ created_at       │
│ updated_at       │       ┌──────────────────────┐
└──────────────────┘       │  presence_logs        │
                           ├──────────────────────┤
                           │ id (PK, UUID)        │
                           │ reservation_id (FK)  │
                           │ latitude             │
                           │ longitude            │
                           │ accuracy             │
                           │ recorded_at          │
                           └──────────────────────┘
```

### 16.2 Table Definitions

#### `drivers`
| Column        | Type         | Constraints          | Description                    |
|---------------|-------------|----------------------|--------------------------------|
| id            | UUID        | PK                   | Unique driver identifier       |
| name          | VARCHAR(255)| NOT NULL             | Driver's display name          |
| phone         | VARCHAR(20) | NOT NULL, UNIQUE     | Phone number                   |
| email         | VARCHAR(255)| UNIQUE               | Email address                  |
| vehicle_type  | ENUM        | NOT NULL             | 'car' or 'motorcycle'         |
| vehicle_plate | VARCHAR(20) | NOT NULL             | Vehicle license plate          |
| created_at    | TIMESTAMP   | NOT NULL, DEFAULT NOW| Record creation time           |
| updated_at    | TIMESTAMP   | NOT NULL, DEFAULT NOW| Last update time               |

#### `parking_spots`
| Column        | Type         | Constraints          | Description                    |
|---------------|-------------|----------------------|--------------------------------|
| id            | UUID        | PK                   | Unique spot identifier         |
| floor_number  | INT         | NOT NULL, 1-5        | Floor number                   |
| spot_number   | INT         | NOT NULL             | Spot number on the floor       |
| vehicle_type  | ENUM        | NOT NULL             | 'car' or 'motorcycle'         |
| spot_code     | VARCHAR(10) | NOT NULL, UNIQUE     | Human-readable code (e.g., F3-C-012) |
| status        | ENUM        | NOT NULL, DEFAULT 'available' | 'available', 'reserved', 'occupied' |
| created_at    | TIMESTAMP   | NOT NULL, DEFAULT NOW| Record creation time           |
| updated_at    | TIMESTAMP   | NOT NULL, DEFAULT NOW| Last update time               |

#### `reservations`
| Column          | Type         | Constraints          | Description                    |
|-----------------|-------------|----------------------|--------------------------------|
| id              | UUID        | PK                   | Unique reservation identifier  |
| driver_id       | UUID        | FK → drivers.id, NOT NULL | Driver who made the reservation |
| spot_id         | UUID        | FK → parking_spots.id, NOT NULL | Assigned parking spot |
| vehicle_type    | ENUM        | NOT NULL             | 'car' or 'motorcycle'         |
| assignment_mode | ENUM        | NOT NULL             | 'system_assigned' or 'user_selected' |
| status          | ENUM        | NOT NULL, DEFAULT 'pending' | 'pending', 'confirmed', 'checked_in', 'checked_out', 'expired', 'cancelled' |
| idempotency_key | VARCHAR(64) | NOT NULL, UNIQUE     | Client-provided idempotency key |
| confirmed_at    | TIMESTAMP   | NULLABLE             | When reservation was confirmed |
| expires_at      | TIMESTAMP   | NULLABLE             | 1 hour after confirmed_at      |
| checked_in_at   | TIMESTAMP   | NULLABLE             | When driver checked in         |
| checked_out_at  | TIMESTAMP   | NULLABLE             | When driver checked out        |
| cancelled_at    | TIMESTAMP   | NULLABLE             | When reservation was cancelled |
| created_at      | TIMESTAMP   | NOT NULL, DEFAULT NOW| Record creation time           |
| updated_at      | TIMESTAMP   | NOT NULL, DEFAULT NOW| Last update time               |

#### `billing_records`
| Column           | Type          | Constraints          | Description                    |
|------------------|--------------|----------------------|--------------------------------|
| id               | UUID         | PK                   | Unique billing record ID       |
| reservation_id   | UUID         | FK → reservations.id, UNIQUE | One billing per reservation |
| booking_fee      | DECIMAL(12,2)| NOT NULL, DEFAULT 0  | Booking fee (5,000 IDR)       |
| parking_fee      | DECIMAL(12,2)| NOT NULL, DEFAULT 0  | Hourly parking charges         |
| overnight_fee    | DECIMAL(12,2)| NOT NULL, DEFAULT 0  | Overnight flat fee (20,000 IDR)|
| cancellation_fee | DECIMAL(12,2)| NOT NULL, DEFAULT 0  | Cancellation fee if applicable |
| penalty_amount   | DECIMAL(12,2)| NOT NULL, DEFAULT 0  | Wrong-spot penalty             |
| total_amount     | DECIMAL(12,2)| NOT NULL, DEFAULT 0  | Sum of all fees                |
| duration_minutes | INT          | DEFAULT 0            | Actual parking duration        |
| billed_hours     | INT          | DEFAULT 0            | Ceiling of duration in hours   |
| is_overnight     | BOOLEAN      | NOT NULL, DEFAULT FALSE | Whether session crossed midnight |
| idempotency_key  | VARCHAR(64)  | NOT NULL, UNIQUE     | Idempotency key for invoice    |
| status           | ENUM         | NOT NULL, DEFAULT 'pending' | 'pending', 'calculated', 'invoiced', 'paid' |
| created_at       | TIMESTAMP    | NOT NULL, DEFAULT NOW| Record creation time           |
| updated_at       | TIMESTAMP    | NOT NULL, DEFAULT NOW| Last update time               |

#### `payments`
| Column          | Type          | Constraints          | Description                    |
|-----------------|--------------|----------------------|--------------------------------|
| id              | UUID         | PK                   | Unique payment identifier      |
| billing_id      | UUID         | FK → billing_records.id | Associated billing record   |
| amount          | DECIMAL(12,2)| NOT NULL             | Payment amount                 |
| payment_method  | ENUM         | NOT NULL             | 'qris', 'credit_card', 'debit', 'ewallet' |
| payment_gateway | VARCHAR(50)  | NOT NULL             | Gateway provider name          |
| transaction_ref | VARCHAR(100) | NULLABLE             | External transaction reference |
| idempotency_key | VARCHAR(64)  | NOT NULL, UNIQUE     | Idempotency key for payment    |
| status          | ENUM         | NOT NULL, DEFAULT 'pending' | 'pending', 'success', 'failed', 'refunded' |
| paid_at         | TIMESTAMP    | NULLABLE             | When payment was completed     |
| created_at      | TIMESTAMP    | NOT NULL, DEFAULT NOW| Record creation time           |
| updated_at      | TIMESTAMP    | NOT NULL, DEFAULT NOW| Last update time               |

#### `penalties`
| Column          | Type          | Constraints          | Description                    |
|-----------------|--------------|----------------------|--------------------------------|
| id              | UUID         | PK                   | Unique penalty identifier      |
| reservation_id  | UUID         | FK → reservations.id | Associated reservation         |
| penalty_type    | ENUM         | NOT NULL             | 'wrong_spot'                  |
| amount          | DECIMAL(12,2)| NOT NULL             | Penalty amount                 |
| description     | TEXT         | NULLABLE             | Human-readable description     |
| created_at      | TIMESTAMP    | NOT NULL, DEFAULT NOW| Record creation time           |

#### `presence_logs`
| Column          | Type          | Constraints          | Description                    |
|-----------------|--------------|----------------------|--------------------------------|
| id              | UUID         | PK                   | Unique log entry identifier    |
| reservation_id  | UUID         | FK → reservations.id | Associated active reservation  |
| floor_number    | INT          | NOT NULL             | Floor of assigned spot         |
| spot_number     | INT          | NOT NULL             | Spot number on floor           |
| sensor_status   | VARCHAR(20)  | NOT NULL             | Sensor reading (occupied/empty)|
| recorded_at     | TIMESTAMP    | NOT NULL             | When location was recorded     |

### 16.3 Key Indexes

```sql
-- Prevent double-booking: unique active reservation per spot
CREATE UNIQUE INDEX idx_reservations_active_spot
  ON reservations (spot_id)
  WHERE status IN ('confirmed', 'checked_in');

-- Fast lookup by driver
CREATE INDEX idx_reservations_driver ON reservations (driver_id, status);

-- Fast availability queries
CREATE INDEX idx_parking_spots_availability ON parking_spots (vehicle_type, status, floor_number);

-- Reservation expiry job
CREATE INDEX idx_reservations_expiry ON reservations (status, expires_at)
  WHERE status = 'confirmed';

-- Billing lookup
CREATE INDEX idx_billing_reservation ON billing_records (reservation_id);

-- Payment lookup
CREATE INDEX idx_payments_billing ON payments (billing_id, status);

-- Presence time-series
CREATE INDEX idx_presence_reservation_time ON presence_logs (reservation_id, recorded_at);
```

---

## 17. Testing Requirements

### 17.1 Unit Tests

| Test Area              | Scenarios                                                    |
|------------------------|--------------------------------------------------------------|
| Pricing Rules          | First hour = 5,000 IDR; subsequent hours = 5,000 IDR each; partial hour rounds up; overnight fee = 20,000 IDR; no overstay penalty |
| Overlap Detection      | Same spot, overlapping time windows must be rejected; non-overlapping allowed; edge cases (exact boundary times) |
| Idempotency            | Duplicate `CreateReservation` with same key returns same result; duplicate `GenerateInvoice` with same key returns same result; different keys create different records |

### 17.2 Integration Tests

| Test Area                    | Scenarios                                              |
|------------------------------|--------------------------------------------------------|
| Reservation → Billing Flow   | Create reservation → check-in → check-out → verify billing calculation matches expected amount |

### 17.3 End-to-End Test Scenarios

| #  | Scenario                              | Description                                                                                     |
|----|---------------------------------------|-------------------------------------------------------------------------------------------------|
| 1  | Happy Path Reservation                | Driver reserves (system-assigned) → checks in → parks → checks out → pays → spot released      |
| 2  | Double-Book Prevention                | Two drivers attempt to reserve the same spot simultaneously → only one succeeds                 |
| 3  | User-Selected Spot Contention/Queue   | Multiple drivers select the same spot → hold/queue mechanism ensures only one gets it           |
| 4  | Reservation Expiry (No-Show)          | Driver reserves but does not check in within 1 hour → reservation expires → booking fee (5,000 IDR) consumed → spot released |
| 5  | Wrong-Spot Penalty                    | Driver reserves spot A but parks in spot B → 200,000 IDR penalty applied                       |
| 6  | Cancellation — Free (within 2 min)    | Driver cancels within 2 minutes of confirmation → 0 IDR fee → spot released                    |
| 7  | Cancellation — Paid (after 2 min)     | Driver cancels after 2 minutes but before check-in → 5,000 IDR fee → spot released             |
| 8  | Extended Stay Billing (No Penalty)    | Driver stays 2 hours beyond reservation → billed at standard rate → no overstay penalty        |
| 9  | Overnight Fee                         | Driver parks from 22:00 to 06:00 → hourly charges + 20,000 IDR overnight fee                  |
| 10 | Payment Checkout — Success            | QRIS payment processed successfully → payment status = success → receipt generated             |
| 11 | Payment Checkout — Failure            | Payment gateway returns failure → payment status = failed → retry available                    |

---

## 18. Non-Functional Requirements

### 18.1 Performance
| Metric                    | Target                          |
|---------------------------|---------------------------------|
| Reservation response time | < 500ms (p95)                  |
| Availability query        | < 200ms (p95)                  |
| Location update ingestion | < 100ms per update             |
| Concurrent reservations   | Support 100+ simultaneous      |

### 18.2 Reliability
| Metric                    | Target                          |
|---------------------------|---------------------------------|
| Uptime                    | 99.9%                          |
| Data durability           | No reservation or payment data loss |
| Graceful degradation      | Core flows work when non-core services are down |

### 18.3 Security
- JWT or mTLS for service authentication
- Rate limiting on all public endpoints
- Input validation on all API requests
- No sensitive data in logs
- CORS with specific allowed origins

### 18.4 Observability
- Structured logging (JSON format) with correlation IDs
- Distributed tracing (OpenTelemetry)
- Health check endpoints (`/health`, `/ready`) per service
- Metrics export (Prometheus-compatible)

---

## 19. Assumptions

1. The parking area is a single building with a fixed layout (5 floors, 30 car + 50 motorcycle spots per floor). The layout does not change at runtime.
2. Each Driver has one vehicle type per session (car or motorcycle). A Driver cannot reserve spots for multiple vehicles simultaneously.
3. Each parking spot is equipped with an occupancy sensor that polls every 30 seconds.
4. Spot verification is based on sensor readings; the backend queries the sensor gateway to confirm vehicle presence.
5. The payment gateway and QRIS provider are external third-party services; the Payment Service integrates via their APIs (stubbed for testing).
6. The Notification Service is a stub — it logs notification payloads but does not send actual messages.
7. Wrong-spot detection relies on spot sensor data; if the assigned spot sensor shows empty after check-in, a warning is flagged.
8. All monetary values are in IDR (Indonesian Rupiah).
9. The system operates in a single timezone (WIB / UTC+7) for overnight fee calculation.
10. The super app handles Driver registration and authentication; ParkirPintar receives a valid JWT token from the super app.
11. There is no physical barrier integration — check-in and check-out are app-driven (sensor verification + manual).
12. Database is PostgreSQL; caching and locking use Redis; message queuing uses Redis Streams or a similar lightweight queue.
13. For testing purposes, payment gateway responses are simulated/stubbed.

---

## 20. Deliverables

| #  | Deliverable                          | Format / Location                    |
|----|--------------------------------------|--------------------------------------|
| 1  | Solution Diagrams (HLD, LLD, ERD)   | `README.md`                          |
| 2  | Configuration Files                  | Repository root / `config/` directory |
| 3  | Microservice Source Code (Go)        | `services/` directory                |
| 4  | Proto Definitions (gRPC)             | `proto/` directory                   |
| 5  | Docker / Docker Compose              | Repository root                      |
| 6  | Database Migrations                  | `migrations/` directory              |
| 7  | Unit Tests                           | `*_test.go` files alongside source   |
| 8  | Integration Tests                    | `tests/integration/` directory       |
| 9  | End-to-End Tests                     | `tests/e2e/` directory               |
| 10 | README with setup & run instructions | `README.md`                          |
| 11 | Assumptions documentation            | `README.md`                          |
| 12 | Third-party library justifications   | `README.md`                          |

---

## 21. Glossary

| Term              | Definition                                                        |
|-------------------|-------------------------------------------------------------------|
| Driver            | End user who parks a vehicle                                      |
| Spot              | A single parking space identified by floor, type, and number      |
| Reservation       | A booking that holds a spot for a Driver                          |
| Check-In          | The moment a Driver arrives and begins the parking session        |
| Check-Out         | The moment a Driver ends the parking session                      |
| Geofence          | A virtual geographic boundary used to detect Driver arrival       |
| QRIS              | Quick Response Code Indonesian Standard — a QR-based payment method |
| Idempotency Key   | A unique client-provided key ensuring an operation is processed only once |
| System-Assigned   | Spot assignment mode where the system picks the best available spot |
| User-Selected     | Spot assignment mode where the Driver chooses a specific spot     |
| Overnight         | A parking session that crosses midnight                           |
| No-Show           | A Driver who does not check in within the reservation hold time   |
| Overstay          | Parking beyond the reserved end time (no penalty in ParkirPintar) |
| HLD               | High-Level Design                                                 |
| LLD               | Low-Level Design                                                  |
| ERD               | Entity Relationship Diagram                                       |
| gRPC              | Google Remote Procedure Call — high-performance RPC framework     |
| mTLS              | Mutual Transport Layer Security                                   |

---

*End of PRD*
