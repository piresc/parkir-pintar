# API Reference

## Health Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Build info (service name, version, go version) |
| GET | `/health/live` | Liveness probe — always 200 |
| GET | `/health/ready` | Readiness probe — checks all dependencies |
| GET | `/health/detailed` | Per-dependency status with check durations |

## Example Domain Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/examples` | List examples (query: `limit`, `offset`) |
| GET | `/api/v1/examples/:id` | Get example by ID |
| POST | `/api/v1/examples` | Create example |
| PUT | `/api/v1/examples/:id` | Update example |
| DELETE | `/api/v1/examples/:id` | Delete example |

## Standard Response Format

### Success
```json
{
  "status": "success",
  "data": { ... }
}
```

### Error
```json
{
  "status": "error",
  "error": "error message",
  "request_id": "optional-transaction-id"
}
```

## Authentication

- **JWT**: `Authorization: Bearer <token>` — validates HMAC-SHA256 signed tokens
- **API Key**: `X-API-Key: <key>` — validates against configured service keys
