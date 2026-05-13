# OpenAPI Specification Generation

This document describes how to generate OpenAPI (Swagger) specifications from the ParkirPintar proto files.

## Overview

ParkirPintar uses pure gRPC services defined in `proto/`. The proto files do **not** currently include gRPC-Gateway HTTP annotations (`google.api.http`), so OpenAPI generation requires either adding annotations or using a Connect-based approach.

## Option 1: gRPC-Gateway + protoc-gen-openapiv2

This is the standard approach for generating OpenAPI v2 (Swagger) specs from proto files with HTTP annotations.

### Prerequisites

```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Adding HTTP Annotations

To generate OpenAPI specs, add `google.api.http` annotations to your service RPCs. Example for `search/v1/search.proto`:

```protobuf
import "google/api/annotations.proto";

service SearchService {
  rpc GetAvailability(GetAvailabilityRequest) returns (AvailabilityResponse) {
    option (google.api.http) = {
      get: "/api/v1/availability"
    };
  }
  rpc GetFloorMap(GetFloorMapRequest) returns (FloorMapResponse) {
    option (google.api.http) = {
      get: "/api/v1/floors/{floor_number}/map"
    };
  }
  rpc GetSpotDetails(GetSpotDetailsRequest) returns (SpotDetailsResponse) {
    option (google.api.http) = {
      get: "/api/v1/spots/{spot_id}"
    };
  }
}
```

You'll also need the `google/api/annotations.proto` and `google/api/http.proto` dependencies. These are available from [googleapis](https://github.com/googleapis/googleapis).

### Generation Command

```bash
protoc \
  -I proto/ \
  -I third_party/googleapis \
  --openapiv2_out=docs/api \
  --openapiv2_opt=logtostderr=true \
  --openapiv2_opt=generate_unbound_methods=true \
  proto/search/v1/search.proto \
  proto/payment/v1/payment.proto \
  proto/billing/v1/billing.proto \
  proto/reservation/v1/reservation.proto \
  proto/presence/v1/presence.proto \
  proto/notification/v1/notification.proto
```

The `generate_unbound_methods=true` option generates OpenAPI entries even for RPCs without HTTP annotations (they'll use the default POST mapping).

### Using buf

With buf, add a `buf.gen.yaml` at the project root:

```yaml
version: v2
plugins:
  - remote: buf.build/grpc-ecosystem/openapiv2
    out: docs/api
    opt:
      - generate_unbound_methods=true
```

Then run:

```bash
buf generate proto/
```

## Option 2: buf + Connect (gRPC-compatible HTTP API)

[Connect](https://connectrpc.com/) provides a gRPC-compatible protocol that works over HTTP/1.1 and HTTP/2 without needing gRPC-Gateway annotations. If you adopt Connect, you can use its tooling to expose HTTP+JSON APIs directly.

### Prerequisites

```bash
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

### Generation

Connect doesn't generate OpenAPI specs directly, but you can use [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi):

```bash
go install github.com/sudorandom/protoc-gen-connect-openapi/cmd/protoc-gen-connect-openapi@latest
```

Then generate OpenAPI v3 specs:

```bash
protoc \
  -I proto/ \
  --connect-openapi_out=docs/api \
  proto/search/v1/search.proto \
  proto/payment/v1/payment.proto \
  proto/billing/v1/billing.proto \
  proto/reservation/v1/reservation.proto \
  proto/presence/v1/presence.proto \
  proto/notification/v1/notification.proto
```

Or with buf (`buf.gen.yaml`):

```yaml
version: v2
plugins:
  - local: protoc-gen-connect-openapi
    out: docs/api
    opt:
      - format=yaml
```

```bash
buf generate proto/
```

## Recommendation

Since ParkirPintar currently uses pure gRPC without HTTP annotations, the quickest path to OpenAPI docs is:

1. **For documentation only**: Use `protoc-gen-openapiv2` with `generate_unbound_methods=true` — no proto changes needed.
2. **For a REST gateway**: Add `google.api.http` annotations and deploy grpc-gateway as a reverse proxy.
3. **For modern HTTP+JSON**: Adopt Connect, which gives you HTTP APIs without annotations and generates OpenAPI v3.

## CI Integration

The `proto-check` job in `.github/workflows/ci.yml` runs `buf lint` and `buf breaking` on every PR. OpenAPI generation can be added as a step if needed:

```yaml
- name: Generate OpenAPI specs
  run: buf generate proto/
```
