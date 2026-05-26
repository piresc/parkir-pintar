# Gateway Service

## Purpose & Responsibility

The Gateway service is the single REST entry point for all client-facing traffic, translating HTTP/JSON requests into gRPC calls to downstream microservices and enforcing authentication, rate limiting, and observability at the edge.

## REST API Contract

All routes are prefixed with `/api/v1` and require JWT authentication.

### Reservation Routes

| Method | Path | Handler | Downstream gRPC |
|--------|------|---------|-----------------|
| POST | `/reservations` | CreateReservation | ReservationService.CreateReservation |
| GET | `/reservations` | ListByDriver | ReservationService.ListByDriver |
| GET | `/reservations/:id` | GetReservation | ReservationService.GetReservation |
| DELETE | `/reservations/:id` | CancelReservation | ReservationService.CancelReservation |
| POST | `/reservations/:id/checkin` | CheckIn | ReservationService.CheckIn |
| POST | `/reservations/:id/checkout` | CheckOut | ReservationService.CheckOut |
| POST | `/reservations/:id/confirm` | ConfirmReservation | ReservationService.ConfirmReservation |
| POST | `/reservations/:id/complete` | CompleteCheckout | ReservationService.CompleteCheckout |

### Search Routes

| Method | Path | Handler | Downstream gRPC |
|--------|------|---------|-----------------|
| GET | `/availability` | GetAvailability | SearchService.GetAvailability |
| GET | `/floors/:floor` | GetFloorMap | SearchService.GetFloorMap |
| GET | `/spots/:id` | GetSpotDetails | SearchService.GetSpotDetails |

### Payment Routes

| Method | Path | Handler | Downstream gRPC |
|--------|------|---------|-----------------|
| GET | `/payments/:id/status` | GetPaymentStatus | PaymentService.GetPaymentStatus |

### Billing Routes

| Method | Path | Handler | Downstream gRPC |
|--------|------|---------|-----------------|
| GET | `/reservations/:id/billing` | GetReservationBilling | BillingService.GenerateInvoice |

### Analytics Routes

| Method | Path | Handler | Downstream gRPC |
|--------|------|---------|-----------------|
| GET | `/analytics/peak-hours` | GetPeakHours | AnalyticsService.GetPeakHours |
| GET | `/analytics/occupancy` | GetOccupancy | AnalyticsService.GetUsagePatterns |
| GET | `/analytics/predictions` | GetPredictions | AnalyticsService.PredictResources |

### Presence Routes

| Method | Path | Handler | Downstream gRPC |
|--------|------|---------|-----------------|
| POST | `/presence/stream` | StreamPresence | PresenceService.VerifyPresence |

## Configuration

| Key | Default | Description |
|-----|---------|-------------|
| `server.port` | 8080 | HTTP listen port |
| `server.read_timeout` | 30s | HTTP read timeout |
| `server.write_timeout` | 30s | HTTP write timeout |
| `server.shutdown_timeout` | 30s | Graceful shutdown window |
| `server.allowed_origins` | localhost:3000, localhost:3002 | CORS allowed origins |
| `grpc.client.dial_timeout` | 5s | gRPC client connection timeout |
| `grpc.client.keepalive_time` | 30s | gRPC keepalive interval |
| `grpc.rate_limit.requests_per_second` | 100 | Per-client rate limit |
| `grpc.rate_limit.burst_size` | 200 | Rate limit burst capacity |
| `redis.db` | 0 | Redis database index |
| `jwt.expiration` | 60 min | JWT token lifetime |

## Dependencies

| Dependency | Purpose |
|------------|---------|
| Reservation Service (gRPC :9091) | Reservation lifecycle operations |
| Search Service (gRPC :9092) | Availability and floor map queries |
| Billing Service (gRPC :9093) | Invoice generation |
| Payment Service (gRPC :9094) | Payment status queries |
| Analytics Service (gRPC :9095) | Peak hours, occupancy, predictions |
| Presence Service (gRPC :9096) | Sensor-based presence verification |
| Redis | Rate limiting, health checks |
| PostgreSQL | Health checks only |

Analytics and Presence are **optional** — the gateway degrades gracefully if they are unavailable, returning `503 Service Unavailable` for their endpoints.

## Key Domain Logic

The gateway is a thin translation layer with no business logic. Its responsibilities are:

1. **Authentication propagation**: Extracts JWT claims and forwards `x-user-id` and `authorization` as gRPC metadata to downstream services.
2. **gRPC-to-HTTP error mapping**: Translates gRPC status codes to appropriate HTTP status codes (e.g., `codes.NotFound` → `404`, `codes.AlreadyExists` → `409`).
3. **Middleware chain**: Recovery → Security Headers → Metrics → CORS → Rate Limiting → Tracing.

## Error Handling Approach

- gRPC errors from downstream services are mapped to HTTP status codes via `grpcCodeToHTTP()`.
- Non-gRPC errors (unexpected) are logged and returned as `500 Internal Server Error`.
- Rate limit violations return `429 Too Many Requests`.
- Missing/invalid JWT returns `401 Unauthorized`.

| gRPC Code | HTTP Status |
|-----------|-------------|
| OK | 200 |
| InvalidArgument | 400 |
| NotFound | 404 |
| AlreadyExists | 409 |
| PermissionDenied | 403 |
| Unauthenticated | 401 |
| FailedPrecondition | 412 |
| ResourceExhausted | 429 |
| Unavailable | 503 |
| DeadlineExceeded | 504 |
