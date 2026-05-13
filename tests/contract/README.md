# Contract Testing

This package contains API contract tests that verify service responses and event payloads match their expected schemas. The proto definitions serve as the single source of truth for contracts.

## Philosophy

Contract tests sit between unit tests and integration tests. They verify that:

1. **gRPC responses** match the proto schema (field presence, types, valid enum values)
2. **NATS event payloads** follow the agreed-upon JSON structure
3. **Breaking changes** are caught before deployment

These tests don't require running services — they validate the serialization contract using the generated proto types directly.

## Running the Tests

```bash
# Run contract tests (gated by build tag)
go test -tags contract -v ./tests/contract/
```

Tests are excluded from normal `go test ./...` runs because of the `//go:build contract` constraint.

## What's Tested

### gRPC Response Contracts

| Test | Validates |
|------|-----------|
| `TestReservationResponseContract` | Field presence, types, timestamp format, enum values for ReservationResponse |
| `TestCheckOutResponseContract` | Billing fields, nested reservation, monetary amount encoding |

### NATS Event Contracts

| Test | Validates |
|------|-----------|
| `TestNATSReservationEventContract` | Required fields, event_type values, timestamp format |
| `TestNATSEventSubjectNaming` | Subject naming convention follows `<domain>.<event>` pattern |

## Adding New Contract Tests

1. Identify the proto message or event payload to test.
2. Create a test that:
   - Constructs a valid message with all fields populated
   - Serializes it to JSON (using `protojson` for proto messages)
   - Parses the JSON into a generic map
   - Asserts field presence, types, and valid values
3. Use subtests (`t.Run`) to group related assertions.

### Example: Adding a Payment Response Contract

```go
func TestPaymentResponseContract(t *testing.T) {
    resp := &paymentv1.PaymentResponse{
        Id:     "pay-001",
        Status: "completed",
        Amount: 50000,
    }

    data, err := protojson.Marshal(resp)
    require.NoError(t, err)

    var fields map[string]interface{}
    json.Unmarshal(data, &fields)

    t.Run("required_fields", func(t *testing.T) {
        assert.Contains(t, fields, "id")
        assert.Contains(t, fields, "status")
        assert.Contains(t, fields, "amount")
    })
}
```

## Proto Source of Truth

All contracts are derived from the proto definitions in:

- `proto/reservation/v1/reservation.proto`
- `proto/billing/v1/billing.proto`
- `proto/payment/v1/payment.proto`
- `proto/presence/v1/presence.proto`
- `proto/notification/v1/notification.proto`
- `proto/search/v1/search.proto`

When a proto file changes, the corresponding contract tests should be updated to reflect the new schema.

## CI Integration

Contract tests can be run in CI as a separate step:

```yaml
- name: Contract Tests
  run: go test -tags contract -v ./tests/contract/
```

They're fast (no external dependencies) and catch schema drift early.
