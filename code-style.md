# Nebengjek Clean Code Style Guide

A repeatable coding pattern extracted from the nebengjek microservices project. Use this as a blueprint when implementing new services.

---

## Table of Contents

1. [Project Structure](#1-project-structure)
2. [Service Architecture (Clean Architecture)](#2-service-architecture)
3. [Folder Convention Per Service](#3-folder-convention-per-service)
4. [Interface Definitions](#4-interface-definitions)
5. [Handler Layer](#5-handler-layer)
6. [Usecase Layer](#6-usecase-layer)
7. [Repository Layer](#7-repository-layer)
8. [Gateway Layer](#8-gateway-layer)
9. [Domain Models](#9-domain-models)
10. [Error Handling](#10-error-handling)
11. [Configuration](#11-configuration)
12. [Middleware](#12-middleware)
13. [Dependency Injection](#13-dependency-injection)
14. [Logging](#14-logging)
15. [Testing](#15-testing)
16. [Observability](#16-observability)
17. [Docker & Deployment](#17-docker--deployment)
18. [Naming Conventions](#18-naming-conventions)
19. [Key Dependencies](#19-key-dependencies)

---

## 1. Project Structure

```
project-root/
├── cmd/                        # Service entrypoints
│   ├── service-a/
│   │   ├── main.go
│   │   └── Dockerfile
│   └── service-b/
│       ├── main.go
│       └── Dockerfile
├── config/                     # Environment config files (.env)
├── db/
│   └── migrations/             # SQL migration files
├── docker-compose.yml
├── docs/                       # Documentation
├── go.mod
├── go.sum
├── internal/                   # Shared internal packages
│   ├── pkg/
│   │   ├── config/             # Config loading utilities
│   │   ├── database/           # DB & Redis client wrappers
│   │   ├── health/             # Health check handler
│   │   ├── http/               # Shared HTTP client
│   │   ├── jwt/                # JWT utilities
│   │   ├── logger/             # Structured logging (slog)
│   │   ├── middleware/         # HTTP middleware
│   │   ├── models/             # Shared domain models
│   │   │   ├── core/           # Config, common types
│   │   │   ├── user/
│   │   │   ├── ride/
│   │   │   ├── match/
│   │   │   ├── location/
│   │   │   ├── notification/
│   │   │   └── websocket/
│   │   ├── nats/               # NATS client, producer, constants
│   │   ├── newrelic/           # APM integration
│   │   ├── server/             # Graceful server wrapper
│   │   └── tracing/            # Distributed tracing
│   └── utils/                  # Utility functions (geo, http, msisdn)
├── services/                   # Service business logic
│   ├── service-a/
│   └── service-b/
└── sonar-project.properties
```

---

## 2. Service Architecture

Each service follows **Clean Architecture** with clear dependency direction:

```
Handler (HTTP/NATS) → Usecase (Business Logic) → Repository (Data) / Gateway (External)
```

- **Handler**: Receives requests, validates input, calls usecase, returns response
- **Usecase**: Orchestrates business logic, depends on repository and gateway interfaces
- **Repository**: Data persistence (PostgreSQL, Redis)
- **Gateway**: Outbound communication (NATS publish, HTTP calls to other services)

Dependencies flow inward. Inner layers define interfaces; outer layers implement them.

---

## 3. Folder Convention Per Service

```
services/<service-name>/
├── usecase.go                  # Usecase interface definition
├── repository.go               # Repository interface definition
├── gateway.go                  # Gateway interface definition (or gateways.go)
├── errors/
│   └── errors.go               # Domain-specific sentinel errors
├── handler/
│   ├── routes.go               # Route registration + composite handler
│   ├── http/
│   │   ├── <domain>.go         # HTTP handler implementation
│   │   └── <domain>_test.go
│   └── nats/
│       ├── nats.go             # NATS consumer handler
│       └── nats_test.go
├── usecase/
│   ├── init.go                 # Usecase struct + constructor
│   ├── <domain>.go             # Business logic methods
│   └── <domain>_test.go
├── repository/
│   ├── init.go                 # Repository struct + constructor
│   ├── <domain>.go             # Data access methods
│   └── <domain>_test.go
├── gateway/
│   ├── <transport>.go          # Gateway implementation (nats.go, http_gateway.go)
│   └── <transport>_test.go
└── mocks/
    ├── mock_usecase.go         # Generated mock
    ├── mock_repository.go      # Generated mock
    └── mock_gateway.go         # Generated mock
```

---

## 4. Interface Definitions

Interfaces live at the **service package root** (not inside implementation folders). Each has a `go:generate` directive for mock generation.

```go
// services/<service>/usecase.go
package servicename

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/org/project/services/servicename ServiceUC
type ServiceUC interface {
    CreateItem(ctx context.Context, item *models.Item) error
    GetItem(ctx context.Context, id string) (*models.Item, error)
}
```

```go
// services/<service>/repository.go
package servicename

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/org/project/services/servicename ServiceRepo
type ServiceRepo interface {
    Create(ctx context.Context, item *models.Item) error
    GetByID(ctx context.Context, id string) (*models.Item, error)
}
```

```go
// services/<service>/gateway.go
package servicename

//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/org/project/services/servicename ServiceGW
type ServiceGW interface {
    PublishEvent(ctx context.Context, event *models.Event) error
}
```

---

## 5. Handler Layer

### HTTP Handler Pattern

```go
// handler/http/<domain>.go
type DomainHandler struct {
    domainUC servicename.ServiceUC
}

func NewDomainHandler(domainUC servicename.ServiceUC) *DomainHandler {
    return &DomainHandler{domainUC: domainUC}
}

func (h *DomainHandler) CreateItem(c echo.Context) error {
    // 1. Bind request
    var req models.CreateRequest
    if err := c.Bind(&req); err != nil {
        return utils.BadRequestResponse(c, "Invalid request payload")
    }

    // 2. Validate input
    if req.Name == "" {
        return utils.BadRequestResponse(c, "Name is required")
    }

    // 3. Call usecase
    result, err := h.domainUC.CreateItem(c.Request().Context(), &req)
    if err != nil {
        if errors.Is(err, domainerrors.ErrAlreadyExists) {
            return utils.ErrorResponseHandler(c, http.StatusConflict, "Item already exists")
        }
        return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to create item")
    }

    // 4. Return standardized response
    return utils.SuccessResponse(c, http.StatusCreated, "Item created successfully", result)
}
```

### NATS JetStream Handler Pattern

```go
// handler/nats/nats.go
type ServiceHandler struct {
    serviceUC  servicename.ServiceUC
    natsClient *natspkg.Client
    nrApp      *newrelic.Application
    cfg        *core.Config
}

func (h *ServiceHandler) InitNATSConsumers() error {
    consumerConfigs := natspkg.DefaultConsumerConfigs()
    config := consumerConfigs["consumer_name"]

    if err := h.natsClient.CreateConsumer(config); err != nil {
        return fmt.Errorf("failed to create consumer: %w", err)
    }

    if err := h.natsClient.ConsumeMessages("STREAM", "consumer_name", h.handleEventJS); err != nil {
        return fmt.Errorf("failed to start consuming: %w", err)
    }
    return nil
}

func (h *ServiceHandler) handleEventJS(msg jetstream.Msg) error {
    ctx := context.Background()

    // New Relic transaction
    if h.nrApp != nil {
        txn := h.nrApp.StartTransaction("NATS/event.name")
        defer txn.End()
        ctx = newrelic.NewContext(ctx, txn)
    }

    return h.handleEvent(ctx, msg.Data())
}
```

### Route Registration (Composite Handler)

```go
// handler/routes.go
type Handler struct {
    httpHandler  *httpHandler.DomainHandler
    natsHandler  *natsHandler.ServiceHandler
    cfg          *core.Config
}

func NewHandler(uc servicename.ServiceUC, natsClient *natspkg.Client, cfg *core.Config) *Handler {
    return &Handler{
        httpHandler:  httpHandler.NewDomainHandler(uc),
        natsHandler:  natsHandler.NewServiceHandler(uc, natsClient, cfg),
        cfg:          cfg,
    }
}

func (h *Handler) RegisterRoutes(e *echo.Echo, mw *middleware.Middleware) {
    // Internal routes (service-to-service with API key)
    internal := e.Group("/internal", mw.APIKeyHandler("service-name"))
    internal.POST("/items", h.httpHandler.CreateItem)
    internal.GET("/items/:id", h.httpHandler.GetItem)
}

func (h *Handler) InitNATSConsumers() error {
    return h.natsHandler.InitNATSConsumers()
}
```

---

## 6. Usecase Layer

```go
// usecase/init.go
type serviceUC struct {
    cfg         *core.Config
    serviceRepo servicename.ServiceRepo
    serviceGW   servicename.ServiceGW
}

func NewServiceUC(cfg *core.Config, repo servicename.ServiceRepo, gw servicename.ServiceGW) (servicename.ServiceUC, error) {
    return &serviceUC{
        cfg:         cfg,
        serviceRepo: repo,
        serviceGW:   gw,
    }, nil
}
```

```go
// usecase/<domain>.go
func (uc *serviceUC) CreateItem(ctx context.Context, req *models.CreateRequest) error {
    // 1. Business validation
    if err := uc.validateRequest(req); err != nil {
        return err
    }

    // 2. Build domain object
    item := &models.Item{
        ID:        uuid.New(),
        Name:      req.Name,
        CreatedAt: time.Now(),
    }

    // 3. Persist via repository
    if err := uc.serviceRepo.Create(ctx, item); err != nil {
        return fmt.Errorf("failed to create item: %w", err)
    }

    // 4. Publish event via gateway
    if err := uc.serviceGW.PublishEvent(ctx, item); err != nil {
        return fmt.Errorf("failed to publish event: %w", err)
    }

    return nil
}
```

---

## 7. Repository Layer

```go
// repository/init.go
type ServiceRepo struct {
    cfg         *core.Config
    db          *sqlx.DB
    redisClient *database.RedisClient
}

func NewServiceRepo(cfg *core.Config, db *sqlx.DB, redisClient *database.RedisClient) *ServiceRepo {
    return &ServiceRepo{
        cfg:         cfg,
        db:          db,
        redisClient: redisClient,
    }
}
```

```go
// repository/<domain>.go

// Query with context and New Relic tracing
func (r *ServiceRepo) GetByID(ctx context.Context, id string) (*models.Item, error) {
    txn := newrelic.FromContext(ctx)
    dbCtx := newrelic.NewContext(ctx, txn)

    query := `SELECT id, name, created_at, updated_at FROM items WHERE id = $1`

    var item models.Item
    err := r.db.QueryRowContext(dbCtx, query, id).Scan(
        &item.ID, &item.Name, &item.CreatedAt, &item.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("item not found")
        }
        return nil, fmt.Errorf("failed to get item: %w", err)
    }
    return &item, nil
}

// Transaction pattern
func (r *ServiceRepo) Create(ctx context.Context, item *models.Item) error {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    query := `INSERT INTO items (id, name, created_at, updated_at)
              VALUES (:id, :name, :created_at, :updated_at)`
    _, err = tx.NamedExecContext(ctx, query, item)
    if err != nil {
        return fmt.Errorf("failed to insert item: %w", err)
    }

    if err = tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    return nil
}

// Redis caching pattern
func (r *ServiceRepo) GetCached(ctx context.Context, key string) (*models.Item, error) {
    data, err := r.redisClient.Get(ctx, key)
    if err != nil {
        return nil, err
    }

    var item models.Item
    if err := json.Unmarshal([]byte(data), &item); err != nil {
        return nil, fmt.Errorf("failed to unmarshal: %w", err)
    }
    return &item, nil
}
```

---

## 8. Gateway Layer

```go
// gateway/nats.go
type ServiceGW struct {
    natsClient *nats.Conn
}

func NewServiceGW(natsClient *nats.Conn) *ServiceGW {
    return &ServiceGW{natsClient: natsClient}
}

func (gw *ServiceGW) PublishEvent(ctx context.Context, item *models.Item) error {
    data, err := json.Marshal(item)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }

    if err := gw.natsClient.Publish("STREAM.subject.action", data); err != nil {
        return fmt.Errorf("failed to publish event: %w", err)
    }
    return nil
}
```

---

## 9. Domain Models

Models live in `internal/pkg/models/<domain>/` and use triple struct tags.

```go
// internal/pkg/models/<domain>/<domain>.go
package domain

type Status string

const (
    StatusPending   Status = "PENDING"
    StatusActive    Status = "ACTIVE"
    StatusCompleted Status = "COMPLETED"
)

type Item struct {
    ID        uuid.UUID `json:"id" db:"id" bson:"_id"`
    Name      string    `json:"name" db:"name" bson:"name"`
    Status    Status    `json:"status" db:"status" bson:"status"`
    CreatedAt time.Time `json:"created_at" db:"created_at" bson:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at" bson:"updated_at"`
}
```

**Rules:**
- Use `uuid.UUID` for IDs
- Use typed constants for status enums (`type Status string`)
- Triple struct tags: `json`, `db` (sqlx), `bson` (optional)
- Use `snake_case` for all tag values
- Use `omitempty` for optional fields

---

## 10. Error Handling

### Sentinel Errors (per service)

```go
// services/<service>/errors/errors.go
package errors

import "errors"

var (
    ErrItemNotFound    = errors.New("item not found")
    ErrAlreadyExists   = errors.New("item already exists")
    ErrInvalidInput    = errors.New("invalid input")
)
```

### Error Wrapping

Always wrap errors with context using `fmt.Errorf` and `%w`:

```go
return fmt.Errorf("failed to create item: %w", err)
return fmt.Errorf("invalid ID format: %w", err)
```

### Error Matching in Handlers

```go
if errors.Is(err, domainerrors.ErrItemNotFound) {
    return utils.ErrorResponseHandler(c, http.StatusNotFound, "Item not found")
}
```

---

## 11. Configuration

### Central Config Struct

```go
// internal/pkg/models/core/config.go
type Config struct {
    App      AppConfig
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    NATS     NATSConfig
    JWT      JWTConfig
    APIKey   APIKeyConfig
    Services ServicesConfig
    NewRelic NewRelicConfig
    Logger   LoggerConfig
}
```

### Env Loading

```go
// internal/pkg/config/config.go
func InitConfig(configPath string) *core.Config {
    env := GetEnv("APP_ENV", "local")
    if env == "local" {
        godotenv.Load(configPath)
    }
    return loadConfigFromEnv()
}

func GetEnv(key, defaultValue string) string { ... }
func GetEnvAsInt(key string, defaultValue int) int { ... }
func GetEnvAsBool(key string, defaultValue bool) bool { ... }
func GetEnvAsFloat(key string, defaultValue float64) float64 { ... }
```

### Per-Service Config File

Each service has its own `config/<service>.env` file.

---

## 12. Middleware

### Unified Middleware (Request ID + Logging + Panic Recovery)

```go
type Middleware struct {
    config *core.Config
    logger *slog.Logger
    tracer tracing.Tracer
    auth   *auth.AuthMiddleware
}

func NewMiddleware(config *core.Config, logger *slog.Logger, tracer tracing.Tracer) *Middleware {
    return &Middleware{config: config, logger: logger, tracer: tracer, auth: auth.NewAuthMiddleware(config)}
}

func (m *Middleware) Handler() echo.MiddlewareFunc { ... }
```

### Auth Patterns

- **API Key** for service-to-service: `X-API-Key` header
- **JWT** for user-facing: Bearer token with `user_id` and `role` claims

```go
// Service-to-service
internal := e.Group("/internal", mw.APIKeyHandler("service-name"))

// User-facing
protected := e.Group("/api", mw.JWTHandler())
```

---

## 13. Dependency Injection

Manual constructor injection. No DI framework.

### Bootstrap Sequence (main.go)

```go
func main() {
    // 1. Load config
    configs := config.InitConfig("config/service.env")

    // 2. Initialize observability
    nrApp := nrpkg.InitNewRelic(configs)
    slogLogger := slogpkg.NewSlogLogger(...)
    tracer := newrelictracer.NewTracer(...)

    // 3. Initialize infrastructure
    postgresClient, _ := database.NewPostgresClient(configs.Database)
    redisClient, _ := database.NewRedisClient(configs.Redis)
    natsClient, _ := nats.NewClient(configs.NATS.URL)

    // 4. Initialize layers (bottom-up)
    repo := repository.NewServiceRepo(configs, postgresClient.GetDB(), redisClient)
    gw := gateway.NewServiceGW(natsClient.GetClient())
    uc, _ := usecase.NewServiceUC(configs, repo, gw)
    handler := handler.NewHandler(uc, natsClient, configs)

    // 5. Initialize NATS consumers
    handler.InitNATSConsumers()

    // 6. Initialize HTTP server
    e := echo.New()
    mw := middleware.NewMiddleware(configs, slogLogger, tracer)
    e.Use(mw.Handler())
    handler.RegisterRoutes(e, mw)

    // 7. Graceful shutdown
    go func() { e.Start(":" + configs.Server.Port) }()
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    e.Shutdown(ctx)
}
```

---

## 14. Logging

Custom wrapper over `log/slog` with structured fields:

```go
// Usage
logger.Info("Item created", logger.String("item_id", item.ID.String()))
logger.Error("Failed to create", logger.String("item_id", id), logger.ErrorField(err))
logger.InfoCtx(ctx, "Processing request", logger.String("request_id", reqID))
```

**Rules:**
- Always use structured fields (no string interpolation in messages)
- Use `logger.ErrorField(err)` for errors
- Use context-aware variants (`InfoCtx`, `ErrorCtx`) when context is available
- Log at boundaries: handler entry, usecase errors, repository failures

---

## 15. Testing

### Test Structure

- Unit tests co-located with source: `<file>_test.go`
- Integration tests: `<file>_integration_test.go`
- Mocks in `mocks/` directory (generated)

### Testing Libraries

| Library | Purpose |
|---------|---------|
| `testify/assert` | Assertions |
| `testify/require` | Fatal assertions |
| `gomock` | Interface mocking |
| `go-sqlmock` | SQL mocking |
| `redismock` | Redis mocking |

### Test Pattern (Arrange-Act-Assert)

```go
func TestCreateItem_Success(t *testing.T) {
    // Arrange
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMockServiceRepo(ctrl)
    mockGW := mocks.NewMockServiceGW(ctrl)
    cfg := &core.Config{}
    uc, err := usecase.NewServiceUC(cfg, mockRepo, mockGW)
    require.NoError(t, err)

    mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
    mockGW.EXPECT().PublishEvent(gomock.Any(), gomock.Any()).Return(nil)

    // Act
    err = uc.CreateItem(context.Background(), &models.CreateRequest{Name: "test"})

    // Assert
    assert.NoError(t, err)
}
```

### HTTP Handler Test Pattern

```go
func TestHandler_CreateItem_Success(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockUC := mocks.NewMockServiceUC(ctrl)
    handler := NewDomainHandler(mockUC)

    mockUC.EXPECT().CreateItem(gomock.Any(), gomock.Any()).Return(nil)

    e := echo.New()
    body, _ := json.Marshal(map[string]interface{}{"name": "test"})
    req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
    req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    err := handler.CreateItem(c)

    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, rec.Code)
}
```

### Repository Test Pattern (sqlmock)

```go
func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
    mockDB, mock, err := sqlmock.New()
    assert.NoError(t, err)
    return sqlx.NewDb(mockDB, "sqlmock"), mock
}

func TestRepo_Create_Success(t *testing.T) {
    db, mock := setupMockDB(t)
    repo := repository.NewServiceRepo(&core.Config{}, db, nil)

    mock.ExpectExec(regexp.QuoteMeta("INSERT INTO items")).
        WithArgs(sqlmock.AnyArg(), "test", sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))

    err := repo.Create(context.Background(), &models.Item{Name: "test"})
    assert.NoError(t, err)
    assert.NoError(t, mock.ExpectationsWereMet())
}
```

### Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "hello", false},
        {"empty input", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validate(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## 16. Observability

- **APM**: New Relic with context propagation
- **Tracing**: Custom tracer interface wrapping New Relic
- **Metrics**: New Relic transactions per handler
- **Health checks**: `/health` endpoint on every service

```go
// Context propagation pattern
txn := newrelic.FromContext(ctx)
dbCtx := newrelic.NewContext(ctx, txn)
// Pass dbCtx to database calls
```

---

## 17. Docker & Deployment

### Multi-stage Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o service ./cmd/<service>/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/service .
ENTRYPOINT ["/app/service"]
```

### docker-compose Pattern

- All services on shared `backend` bridge network
- Health checks on all containers
- `restart: unless-stopped`
- Infrastructure: PostgreSQL, Redis, NATS (JetStream)
- Each service depends on infrastructure via `depends_on`

---

## 18. Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Package names | lowercase, short | `usecase`, `repository`, `gateway` |
| Interface names | PascalCase with suffix | `ServiceUC`, `ServiceRepo`, `ServiceGW` |
| Struct names | PascalCase | `UserHandler`, `RideRepo` |
| Constructor | `New` + struct name | `NewUserHandler()`, `NewRideRepo()` |
| Methods | PascalCase (exported) | `CreateUser()`, `GetByID()` |
| Files | snake_case | `user_handler.go`, `ride_test.go` |
| DB columns | snake_case | `created_at`, `user_id` |
| JSON fields | snake_case | `"ride_id"`, `"created_at"` |
| Env vars | UPPER_SNAKE_CASE | `DB_HOST`, `SERVER_PORT` |
| NATS subjects | dot.separated | `match.accepted.rides` |
| Error vars | `Err` + PascalCase | `ErrUserNotFound` |
| Config structs | PascalCase + `Config` | `DatabaseConfig`, `RedisConfig` |

---

## 19. Key Dependencies

| Package | Purpose |
|---------|---------|
| `labstack/echo/v4` | HTTP framework |
| `nats-io/nats.go` | NATS JetStream messaging |
| `jmoiron/sqlx` + `lib/pq` | PostgreSQL |
| `go-redis/redis/v8` | Redis |
| `golang-jwt/jwt/v4` | JWT auth |
| `google/uuid` | UUID generation |
| `newrelic/go-agent/v3` | APM/tracing |
| `joho/godotenv` | Env file loading |
| `stretchr/testify` | Test assertions |
| `golang/mock` | Mock generation |
| `DATA-DOG/go-sqlmock` | SQL mocking |
| `go-redis/redismock/v8` | Redis mocking |

---

## Quick Start Checklist for New Service

1. Create `cmd/<service>/main.go` and `cmd/<service>/Dockerfile`
2. Create `config/<service>.env`
3. Create `services/<service>/` with interface files at root
4. Implement layers: `repository/` → `gateway/` → `usecase/` → `handler/`
5. Define domain models in `internal/pkg/models/<domain>/`
6. Add sentinel errors in `services/<service>/errors/`
7. Generate mocks: `go generate ./services/<service>/...`
8. Write tests for each layer
9. Add service to `docker-compose.yml`
10. Add health check endpoint
11. Register NATS consumers if needed
12. Add to CI/CD pipeline
