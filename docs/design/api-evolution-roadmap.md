# API Evolution Roadmap — ParkirPintar

## Current State

All ParkirPintar services are at **v1**:

| Service | gRPC Package | REST Prefix | Status |
|---------|-------------|-------------|--------|
| Reservation Service | `reservation.v1` | `/api/v1/reservations` | Stable |
| Billing Service | `billing.v1` | `/api/v1/billing` | Stable |
| Notification Service | `notification.v1` | `/api/v1/notifications` | Stable |
| Parking Spot Service | `parking.v1` | `/api/v1/spots` | Stable |
| Auth/User Service | `auth.v1` | `/api/v1/auth`, `/api/v1/users` | Stable |

## Versioning Strategy

### Protocol Buffers (gRPC)

- Package-level versioning: `package parkir.reservation.v1;`
- Each service version lives in its own directory: `proto/reservation/v1/`, `proto/reservation/v2/`
- Proto files are the source of truth for API contracts
- Generated code goes to `pkg/gen/` with version suffixes

### REST (gRPC-Gateway)

- URL prefix versioning: `/api/v1/`, `/api/v2/`
- gRPC-Gateway generates REST endpoints from proto definitions
- Custom routes defined via `google.api.http` annotations
- JSON field names follow `snake_case` convention

### Versioning Rules

1. Major version bump (`v1` → `v2`): breaking changes
2. Minor additions within a version: additive-only (new fields, new RPCs)
3. Proto field numbers are never reused or reassigned

## Deprecation Policy

### Timeline

- **Minimum 6 months** notice before removing any v1 endpoint
- Deprecation announced via:
  - `Sunset` HTTP header on REST responses (RFC 8594)
  - `Deprecation` HTTP header with date
  - Proto field/RPC `deprecated = true` option
  - Changelog entry and developer notification

### Sunset Header Format

```
Sunset: Sat, 01 Mar 2026 00:00:00 GMT
Deprecation: true
Link: </api/v2/reservations>; rel="successor-version"
```

### Process

1. Mark endpoint/field as deprecated in proto definition
2. Add `Sunset` header to REST responses with target date
3. Notify all known consumers via email/Telegram
4. Monitor usage of deprecated endpoints via metrics
5. Remove after sunset date only if usage drops to zero (or coordinate migration)

## Planned v2 Changes

### Batch Operations

```protobuf
// reservation/v2/reservation.proto
service ReservationService {
  // Existing
  rpc CreateReservation(CreateReservationRequest) returns (CreateReservationResponse);
  
  // New in v2
  rpc BatchCreateReservations(BatchCreateReservationsRequest) returns (BatchCreateReservationsResponse);
  rpc BatchCancelReservations(BatchCancelReservationsRequest) returns (BatchCancelReservationsResponse);
}
```

- Batch endpoints accept up to 100 items per request
- Partial success semantics with per-item error reporting
- Idempotency keys per batch item

### Streaming Updates

```protobuf
// parking/v2/parking.proto
service ParkingSpotService {
  // Real-time spot availability stream
  rpc WatchSpotAvailability(WatchSpotAvailabilityRequest) returns (stream SpotUpdate);
  
  // Occupancy change stream for operators
  rpc StreamOccupancyEvents(StreamOccupancyRequest) returns (stream OccupancyEvent);
}
```

- Server-sent events (SSE) via gRPC-Gateway for REST clients
- Bidirectional streaming for operator dashboards
- Heartbeat every 30s to detect stale connections

### GraphQL Gateway (Under Consideration)

- Evaluate Apollo Federation or gqlgen as a gateway layer
- Would sit in front of existing gRPC services
- Benefits: flexible queries for mobile clients, reduced over-fetching
- Concerns: added complexity, caching strategy, N+1 query prevention
- Decision target: Q3 2025 evaluation, Q4 2025 implementation if approved

## Breaking vs Non-Breaking Changes

### Non-Breaking (allowed within v1)

- Adding new fields to messages (with new field numbers)
- Adding new RPC methods to a service
- Adding new enum values
- Adding new services
- Relaxing validation constraints (e.g., increasing max length)
- Adding optional query parameters to REST endpoints

### Breaking (requires v2)

- Removing or renaming fields
- Changing field types or numbers
- Removing RPC methods or services
- Renaming RPC methods
- Changing request/response message types for existing RPCs
- Tightening validation constraints
- Changing URL paths for REST endpoints
- Changing authentication/authorization requirements
- Changing error code semantics

## Backward Compatibility Guarantees

1. **v1 remains available** for minimum 6 months after v2 GA release
2. **No silent behavior changes** — same request always produces same semantic response
3. **Additive changes only** within a major version
4. **Default values** for new fields ensure old clients work without modification
5. **Error codes are stable** — existing error codes keep their meaning
6. **Pagination tokens** remain valid across minor updates
7. **Webhook payloads** only add new fields, never remove

## Migration Guide Template

When releasing v2, provide consumers with:

### Migration Guide: Service X v1 → v2

```markdown
## Overview
Brief description of why v2 exists and key improvements.

## Timeline
- v2 GA: [date]
- v1 deprecation notice: [date]
- v1 sunset: [date]

## Breaking Changes
| v1 | v2 | Migration Action |
|----|----|--------------------|
| `field_old` | `field_new` | Rename in client code |
| `POST /api/v1/x` | `POST /api/v2/x` | Update base URL |

## New Features in v2
- Feature A: description
- Feature B: description

## Step-by-Step Migration
1. Update proto dependencies to v2
2. Regenerate client stubs
3. Update field mappings (see table above)
4. Test against v2 staging environment
5. Switch production traffic

## Rollback Plan
If issues arise, revert to v1 endpoints (still available until sunset date).

## Support
Contact: parkir-pintar-dev@team (or Telegram group)
```

## Revision History

| Date | Change | Author |
|------|--------|--------|
| 2025-01-15 | Initial roadmap created | ParkirPintar Team |
