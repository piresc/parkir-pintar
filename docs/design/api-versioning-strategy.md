# API Versioning Strategy — ParkirPintar

> **Status:** Approved  
> **Author:** piresc  
> **Date:** 2026-05-13  
> **Applies to:** Gateway REST API (public-facing)

---

## 1. Current Approach: URL Path Versioning

All public API endpoints use URL path versioning:

```
https://parkir-pintar.piresc.dev/api/v1/reservations
https://parkir-pintar.piresc.dev/api/v1/availability
https://parkir-pintar.piresc.dev/api/v1/payments
```

### Why URL Path Versioning

| Criteria | URL Path | Header-Based | Query Param |
|----------|----------|--------------|-------------|
| Visibility | ✅ Obvious in URL | ❌ Hidden | ⚠️ Clutters params |
| Cacheability | ✅ CDN-friendly | ❌ Requires Vary header | ⚠️ Cache key complexity |
| Simplicity | ✅ Easy to implement | ⚠️ Middleware needed | ⚠️ Parsing overhead |
| API docs | ✅ Clear separation | ❌ Harder to document | ❌ Harder to document |
| Go router support | ✅ Native path groups | ⚠️ Custom middleware | ⚠️ Custom middleware |

**Decision:** URL path versioning (`/api/v1/...`) is the primary strategy. It aligns with the Go gateway's router group pattern and is straightforward for consumers.

### Implementation in Gateway

```go
// cmd/gateway/main.go — route registration
v1 := router.Group("/api/v1")
{
    v1.GET("/reservations", h.ListReservations)
    v1.GET("/reservations/:id", h.GetReservation)
    v1.POST("/reservations", h.CreateReservation)
    // ...
}

// Future: v2 group coexists with v1
v2 := router.Group("/api/v2")
{
    v2.GET("/reservations", h.ListReservationsV2)
    // ...
}
```

---

## 2. Versioning Policy

### When to Bump the Major Version

A new API version (`v1` → `v2`) is created **only for breaking changes**. Non-breaking changes are added to the current version.

### Breaking Changes (require version bump)

| Change Type | Example |
|-------------|---------|
| Remove endpoint | `DELETE /api/v1/examples` removed |
| Remove field from response | `invoice.tax_breakdown` removed |
| Rename field | `parking_spot` → `slot` |
| Change field type | `amount: string` → `amount: int` |
| Change error response structure | Different error envelope |
| Change authentication mechanism | JWT claims restructured |
| Change pagination format | Offset-based → cursor-based |
| Remove query parameter | `?status=` filter removed |
| Change HTTP method | `PUT` → `PATCH` for updates |
| Change status code semantics | 200 → 201 for creation |

### Non-Breaking Changes (safe within current version)

| Change Type | Example |
|-------------|---------|
| Add new endpoint | `GET /api/v1/analytics` |
| Add optional field to response | `reservation.estimated_cost` added |
| Add optional query parameter | `?sort_by=created_at` |
| Add new enum value | `status: "expired"` added |
| Relax validation | Max name length 50 → 100 |
| Add optional request field | `notes` field in create body |
| Performance improvements | Faster response, same shape |
| Add new HTTP header | `X-Request-Cost` in response |

---

## 3. Deprecation Policy

### 6-Month Sunset Period

When a version is deprecated:

1. **Announcement (Day 0):** Deprecation notice in changelog, API docs, and developer communications
2. **Sunset Header (Day 0):** All responses from deprecated version include:
   ```
   Sunset: Sat, 13 Nov 2026 00:00:00 GMT
   Deprecation: true
   Link: <https://parkir-pintar.piresc.dev/api/v2/docs>; rel="successor-version"
   ```
3. **Warning Period (Months 1-5):** Deprecated version continues to function normally. Monitoring tracks usage to identify consumers who haven't migrated.
4. **Final Warning (Month 5):** Direct outreach to remaining consumers via registered contact info.
5. **Shutdown (Month 6):** Deprecated version returns `410 Gone` with migration instructions.

### Implementation: Sunset Middleware

```go
// pkg/middleware/deprecation.go
func DeprecationMiddleware(sunsetDate time.Time, successorURL string) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Sunset", sunsetDate.Format(http.TimeFormat))
        c.Header("Deprecation", "true")
        c.Header("Link", fmt.Sprintf("<%s>; rel=\"successor-version\"", successorURL))
        c.Next()
    }
}

// Usage in gateway
v1 := router.Group("/api/v1")
v1.Use(middleware.DeprecationMiddleware(
    time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
    "https://parkir-pintar.piresc.dev/api/v2/docs",
))
```

### Deprecation Monitoring

Track deprecated version usage via OpenTelemetry metrics:

```go
// Metric: api.deprecated_version.requests
// Labels: version, endpoint, consumer_id (from JWT or API key)
meter.Int64Counter("api.deprecated_version.requests").Add(ctx, 1,
    attribute.String("version", "v1"),
    attribute.String("endpoint", c.FullPath()),
    attribute.String("consumer", extractConsumerID(c)),
)
```

Alert when deprecated version still receives > 0 requests within 30 days of sunset.

---

## 4. Migration Guide Template

When releasing a new API version, publish a migration guide following this template:

```markdown
# Migration Guide: v1 → v2

## Timeline
- **v2 Available:** YYYY-MM-DD
- **v1 Deprecated:** YYYY-MM-DD
- **v1 Sunset:** YYYY-MM-DD (6 months after deprecation)

## Breaking Changes Summary

| # | Change | v1 Behavior | v2 Behavior | Impact |
|---|--------|-------------|-------------|--------|
| 1 | ... | ... | ... | ... |

## Step-by-Step Migration

### 1. Update Base URL
```
- https://parkir-pintar.piresc.dev/api/v1/
+ https://parkir-pintar.piresc.dev/api/v2/
```

### 2. [Specific Change Title]
**Before (v1):**
```json
{ "old_field": "value" }
```

**After (v2):**
```json
{ "new_field": "value" }
```

**Action required:** Update your request/response parsing for...

## Compatibility Notes
- Fields added in v2 that don't exist in v1
- Behavioral differences

## Testing
- Staging endpoint: `https://staging.parkir-pintar.piresc.dev/api/v2/`
- Test credentials available upon request

## Support
- Questions: [contact channel]
- Issues: [GitHub issues link]
```

---

## 5. Internal Service Versioning (gRPC)

Internal gRPC services use **protobuf package versioning**:

```protobuf
// proto/reservation/v1/reservation.proto
package parkir_pintar.reservation.v1;

service ReservationService {
    rpc CreateReservation(...) returns (...);
}
```

### gRPC Versioning Rules

- Proto packages are versioned independently (`reservation.v1`, `billing.v1`)
- New fields added to existing messages are always backward-compatible (protobuf guarantees)
- Breaking proto changes → new package version (`reservation.v2`)
- Old and new gRPC services can coexist on different ports or via gRPC reflection

### NATS Event Versioning

Event subjects include version:

```
parkir-pintar.reservation.v1.created
parkir-pintar.reservation.v1.checked_in
parkir-pintar.billing.v1.invoice_generated
```

Event payloads use protobuf or JSON with a `schema_version` field:

```json
{
  "schema_version": "1.0",
  "event_type": "reservation.created",
  "data": { ... }
}
```

---

## 6. Future Option: Header-Based Version Negotiation

If consumer needs diverge significantly, header-based negotiation can supplement path versioning:

```
GET /api/reservations HTTP/1.1
Accept: application/vnd.parkir-pintar.v2+json
```

### When to Consider

- Multiple API consumers need different response shapes
- Mobile vs. web clients need different field sets
- GraphQL-like field selection becomes necessary

### Implementation Sketch

```go
func VersionNegotiationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        accept := c.GetHeader("Accept")
        version := parseVersion(accept) // "v1", "v2", etc.
        if version == "" {
            version = "v1" // default
        }
        c.Set("api_version", version)
        c.Next()
    }
}
```

**Current decision:** Not implementing header-based negotiation. URL path versioning is sufficient for ParkirPintar's current consumer base (single frontend + potential mobile app). Revisit if we onboard third-party integrators.

---

## 7. Version Lifecycle Summary

```
┌──────────┐     ┌────────────┐     ┌──────────────┐     ┌─────────┐
│  Active  │────▶│ Deprecated │────▶│ Sunset (410) │────▶│ Removed │
│          │     │ (+headers) │     │              │     │         │
└──────────┘     └────────────┘     └──────────────┘     └─────────┘
                  6 months            immediate            code cleanup
```

| Version | Status | Sunset Date |
|---------|--------|-------------|
| v1 | Active | — |
| v2 | Planned | — |
