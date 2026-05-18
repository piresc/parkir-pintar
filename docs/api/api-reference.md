# ParkirPintar API Reference

## Overview

ParkirPintar exposes a RESTful API through the API Gateway service. All internal communication uses gRPC over HTTP/2, but clients interact exclusively via REST.

**Base URL:**

```
https://parkir-pintar.piresc.dev/api/v1
```

**Local Development:**

```
http://localhost:8080/api/v1
```

---

## Authentication

All API endpoints (except health checks) require a valid JWT token in the `Authorization` header.

```
Authorization: Bearer <jwt_token>
```

The JWT is issued by the super app and validated by ParkirPintar's Gateway using HMAC-SHA256 signature verification.

### Token Claims

| Claim | Type | Description |
|-------|------|-------------|
| `sub` | string (UUID) | Driver ID |
| `exp` | int64 | Expiration timestamp (Unix) |
| `iat` | int64 | Issued-at timestamp (Unix) |

---

## Common Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | Bearer JWT token |
| `Content-Type` | Yes (POST/PUT) | `application/json` |
| `X-Idempotency-Key` | Recommended (POST) | UUID v4 for write operations |
| `X-Request-ID` | Optional | Client-generated request correlation ID |

---

## Rate Limiting

- **Limit:** 100 requests per minute per IP address
- **Algorithm:** Token bucket
- **State:** Stored in Redis

Rate limit headers are included in every response:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests per window |
| `X-RateLimit-Remaining` | Remaining requests in current window |
| `X-RateLimit-Reset` | Unix timestamp when the window resets |

When rate limited, the API returns `429 Too Many Requests`.

---

## Idempotency

For write operations (`POST`), include an `X-Idempotency-Key` header with a UUID v4 value. If the same key is sent again:

- The server returns the original response without re-executing the operation
- Keys are stored with a TTL of 24 hours
- Idempotency is enforced at both the gRPC middleware level (Redis SETNX) and the database level (UNIQUE constraint)

```bash
curl -X POST https://parkir-pintar.piresc.dev/api/v1/reservations \
  -H "Authorization: Bearer <token>" \
  -H "X-Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"vehicle_type": "car"}'
```

---

## Response Format

### Success Response

```json
{
  "status": "success",
  "data": { ... }
}
```

### Error Response

```json
{
  "status": "error",
  "error": "human-readable error message",
  "code": "ERROR_CODE",
  "request_id": "trace-correlation-id"
}
```

---

## Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `INVALID_REQUEST` | Malformed request body or invalid parameters |
| 401 | `UNAUTHORIZED` | Missing or invalid JWT token |
| 403 | `FORBIDDEN` | Token valid but insufficient permissions |
| 404 | `NOT_FOUND` | Resource does not exist |
| 409 | `CONFLICT` | Resource state conflict (e.g., spot already reserved) |
| 422 | `UNPROCESSABLE_ENTITY` | Valid JSON but business rule violation |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Unexpected server error |
| 503 | `SERVICE_UNAVAILABLE` | Downstream service unavailable |

---

## Endpoints

### Search

#### GET /search/availability

Get parking availability summary by vehicle type.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `vehicle_type` | string | Yes | `car` or `motorcycle` |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "vehicle_type": "car",
    "total_spots": 150,
    "available_spots": 87,
    "floors": [
      {
        "floor_number": 1,
        "total": 30,
        "available": 18
      },
      {
        "floor_number": 2,
        "total": 30,
        "available": 22
      }
    ]
  }
}
```

---

#### GET /search/floors/:floor

Get floor map with all spots and their current status.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `floor` | int | Floor number (1-5) |

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `vehicle_type` | string | No | Filter by `car` or `motorcycle` |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "floor_number": 3,
    "spots": [
      {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "spot_code": "F3-C-001",
        "vehicle_type": "car",
        "status": "available"
      },
      {
        "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "spot_code": "F3-C-002",
        "vehicle_type": "car",
        "status": "reserved"
      }
    ]
  }
}
```

**Spot Status Values:** `available`, `reserved`, `occupied`, `maintenance`

---

#### GET /search/spots/:id

Get detailed information about a specific parking spot.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Parking spot ID |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "spot_code": "F3-C-001",
    "floor_number": 3,
    "spot_number": 1,
    "vehicle_type": "car",
    "status": "available",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-05-13T10:30:00Z"
  }
}
```

---

### Reservations

#### POST /reservations

Create a new parking reservation. The system assigns the best available spot automatically, or the client can specify a preferred spot.

**Request Body:**

```json
{
  "vehicle_type": "car",
  "spot_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "assignment_mode": "system_assigned"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `vehicle_type` | string | Yes | `car` or `motorcycle` |
| `spot_id` | UUID | No | Specific spot (for `user_selected` mode) |
| `assignment_mode` | string | No | `system_assigned` (default) or `user_selected` |

**Response (201):**

```json
{
  "status": "success",
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "driver_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
    "spot_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "spot_code": "F3-C-001",
    "vehicle_type": "car",
    "assignment_mode": "system_assigned",
    "status": "confirmed",
    "confirmed_at": "2025-05-13T10:30:00Z",
    "expires_at": "2025-05-13T11:30:00Z",
    "booking_fee": 5000,
    "idempotency_key": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Notes:**
- A booking fee of 5,000 IDR is charged immediately upon confirmation
- The reservation expires after 1 hour if the driver does not check in
- Uses distributed locking to prevent double-booking

---

#### GET /reservations/:id

Get reservation details by ID.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Reservation ID |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "driver_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
    "spot_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "spot_code": "F3-C-001",
    "vehicle_type": "car",
    "assignment_mode": "system_assigned",
    "status": "confirmed",
    "confirmed_at": "2025-05-13T10:30:00Z",
    "expires_at": "2025-05-13T11:30:00Z",
    "checked_in_at": null,
    "checked_out_at": null,
    "cancelled_at": null,
    "created_at": "2025-05-13T10:30:00Z",
    "updated_at": "2025-05-13T10:30:00Z"
  }
}
```

**Reservation Status Values:** `confirmed`, `checked_in`, `checked_out`, `expired`, `cancelled`

---

#### POST /reservations/:id/cancel

Cancel an active reservation.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Reservation ID |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "status": "cancelled",
    "cancelled_at": "2025-05-13T10:35:00Z",
    "cancellation_fee": 2500,
    "message": "Reservation cancelled. Cancellation fee applied (cancelled after 2 minutes)."
  }
}
```

**Cancellation Policy:**
- Cancelled within 2 minutes of confirmation: no additional fee
- Cancelled after 2 minutes: 50% of booking fee (2,500 IDR) penalty

---

#### POST /reservations/:id/check-in

Check in to the reserved parking spot.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Reservation ID |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "status": "checked_in",
    "checked_in_at": "2025-05-13T10:45:00Z",
    "spot_code": "F3-C-001"
  }
}
```

---

#### POST /reservations/:id/check-out

Check out from the parking spot. Triggers billing calculation and payment processing.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Reservation ID |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "status": "checked_out",
    "checked_out_at": "2025-05-13T13:45:00Z",
    "billing": {
      "billing_id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
      "duration_minutes": 180,
      "billed_hours": 3,
      "parking_fee": 15000,
      "overnight_fee": 0,
      "booking_fee": 5000,
      "total_amount": 20000,
      "is_overnight": false
    },
    "payment": {
      "payment_id": "e5f6a7b8-9012-3456-cdef-789012345678",
      "status": "success",
      "payment_method": "qris",
      "amount": 20000
    }
  }
}
```

**Pricing:**
- Parking fee: ceil(hours) × 5,000 IDR (minimum 1 hour)
- Overnight fee: 20,000 IDR if check-in and check-out span different calendar days (WIB)
- Booking fee: 5,000 IDR (charged at reservation time)

---

#### GET /reservations/driver/:driverId

Get all reservations for a specific driver.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `driverId` | UUID | Driver ID |

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `status` | string | No | Filter by status |
| `limit` | int | No | Max results (default: 20) |
| `offset` | int | No | Pagination offset (default: 0) |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "reservations": [
      {
        "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        "spot_code": "F3-C-001",
        "vehicle_type": "car",
        "status": "checked_out",
        "confirmed_at": "2025-05-13T10:30:00Z",
        "checked_out_at": "2025-05-13T13:45:00Z"
      }
    ],
    "total": 5,
    "limit": 20,
    "offset": 0
  }
}
```

---

### Billing

#### GET /billing/:reservationId

Get billing details for a reservation.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `reservationId` | UUID | Reservation ID |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
    "reservation_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "booking_fee": 5000,
    "parking_fee": 15000,
    "overnight_fee": 0,
    "cancellation_fee": 0,
    "penalty_amount": 0,
    "total_amount": 20000,
    "duration_minutes": 180,
    "billed_hours": 3,
    "is_overnight": false,
    "status": "invoiced",
    "idempotency_key": "billing-f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "created_at": "2025-05-13T13:45:00Z",
    "updated_at": "2025-05-13T13:45:01Z"
  }
}
```

**Billing Status Values:** `pending`, `calculated`, `invoiced`, `paid`

---

### Payments

#### POST /payments

Process a payment for a billing record.

**Request Body:**

```json
{
  "billing_id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
  "amount": 20000,
  "payment_method": "qris"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `billing_id` | UUID | Yes | Associated billing record ID |
| `amount` | int64 | Yes | Payment amount in IDR |
| `payment_method` | string | Yes | `qris`, `bank_transfer`, `e_wallet` |

**Response (201):**

```json
{
  "status": "success",
  "data": {
    "id": "e5f6a7b8-9012-3456-cdef-789012345678",
    "billing_id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
    "amount": 20000,
    "payment_method": "qris",
    "payment_gateway": "midtrans",
    "transaction_ref": "TXN-20250513-ABC123",
    "status": "success",
    "paid_at": "2025-05-13T13:45:02Z",
    "created_at": "2025-05-13T13:45:01Z"
  }
}
```

**Payment Status Values:** `pending`, `success`, `failed`, `refunded`

---

#### GET /payments/:id

Get payment details by ID.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Payment ID |

**Response (200):**

```json
{
  "status": "success",
  "data": {
    "id": "e5f6a7b8-9012-3456-cdef-789012345678",
    "billing_id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
    "amount": 20000,
    "payment_method": "qris",
    "payment_gateway": "midtrans",
    "transaction_ref": "TXN-20250513-ABC123",
    "status": "success",
    "paid_at": "2025-05-13T13:45:02Z",
    "created_at": "2025-05-13T13:45:01Z",
    "updated_at": "2025-05-13T13:45:02Z"
  }
}
```

---

## Health Endpoints (No Auth Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Build info (service name, version, commit, build time) |
| GET | `/health/live` | Liveness probe (always 200) |
| GET | `/health/ready` | Readiness probe (checks PostgreSQL, Redis, NATS) |
| GET | `/health/detailed` | Per-dependency status with check duration |

---

## Status Codes Summary

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created (new resource) |
| 400 | Bad request (invalid input) |
| 401 | Unauthorized (missing/invalid token) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Resource not found |
| 409 | Conflict (state violation) |
| 422 | Unprocessable entity (business rule violation) |
| 429 | Rate limited |
| 500 | Internal server error |
| 503 | Service unavailable |

---

## Webhook Events (NATS JetStream)

For internal consumers, the following events are published:

| Event | Stream | Description |
|-------|--------|-------------|
| `reservation.confirmed` | RESERVATIONS | New reservation created |
| `reservation.checked_in` | RESERVATIONS | Driver checked in |
| `reservation.checked_out` | RESERVATIONS | Driver checked out |
| `reservation.expired` | RESERVATIONS | Reservation expired (no-show) |
| `reservation.cancelled` | RESERVATIONS | Reservation cancelled |
| `billing.calculated` | BILLING | Fee calculation completed |
| `billing.invoiced` | BILLING | Invoice generated |
| `payment.success` | PAYMENTS | Payment processed successfully |
| `payment.failed` | PAYMENTS | Payment processing failed |
| `presence.arrival` | PRESENCE | Spot sensor confirmed vehicle occupancy |
| `presence.wrong_spot` | PRESENCE | Driver detected at wrong spot |
