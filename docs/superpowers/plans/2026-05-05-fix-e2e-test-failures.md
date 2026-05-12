# Fix E2E Test Failures — Billing/Payment Integration, Vehicle Validation, Test Driver Seeding

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the three root causes that make Layer 2 (Docker Compose) E2E tests fail and leave billing/payment non-functional in production deployments: (1) billing & payment services are stubbed instead of wired via gRPC, (2) user-selected reservations do not validate vehicle-type compatibility, and (3) E2E tests send string driver IDs where the database requires UUIDs.

**Architecture:** Add thin gRPC-client adapter structs (mirroring the existing in-process adapters in `tests/e2e/adapters_test.go`) that translate the `reservation.BillingClient` / `reservation.PaymentClient` interfaces into gRPC calls to the standalone billing and payment microservices. Add a vehicle-type guard in the reservation usecase. Seed a test driver via `docker exec` in the Docker Compose test setup.

**Tech Stack:** Go 1.25, gRPC, Protocol Buffers, Docker Compose, PostgreSQL, testify

---

## File Map

| File | Responsibility |
|------|---------------|
| `internal/reservation/client/billing_client.go` | **NEW** — gRPC adapter implementing `usecase.BillingClient` by calling `billingv1.BillingServiceClient` |
| `internal/reservation/client/payment_client.go` | **NEW** — gRPC adapter implementing `usecase.PaymentClient` by calling `paymentv1.PaymentServiceClient` |
| `cmd/reservation/main.go` | **MODIFY** — dial billing & payment gRPC targets, pass real clients to usecase, remove stub types |
| `docker-compose.yml` | **MODIFY** — add `GRPC_BILLING_TARGET` and `GRPC_PAYMENT_TARGET` env vars to reservation service; add `depends_on` billing & payment |
| `internal/reservation/usecase/usecase.go` | **MODIFY** — add vehicle-type compatibility check in `CreateReservation` for `user_selected` mode |
| `internal/reservation/usecase/usecase_test.go` | **MODIFY** — add unit test covering the new vehicle-type mismatch rejection |
| `tests/e2e_docker/setup_test.go` | **MODIFY** — insert a known test driver into Postgres via `docker exec` before tests run |
| `tests/e2e_docker/happy_path_test.go` | **MODIFY** — replace `"test-driver-001"` with the seeded UUID |
| `tests/e2e_docker/cancellation_test.go` | **MODIFY** — replace `"test-driver-001"` with the seeded UUID |
| `tests/e2e_docker/double_book_test.go` | **MODIFY** — replace `"test-driver-001"` with the seeded UUID |
| `tests/e2e_docker/payment_test.go` | **MODIFY** — replace `"test-driver-001"` with the seeded UUID |
| `tests/e2e_docker/middleware_test.go` | **MODIFY** — replace `"test-driver-001"` with the seeded UUID where used |

---

### Task 1: Create gRPC Billing Client Adapter

**Files:**
- Create: `internal/reservation/client/billing_client.go`

**Context:**
The `reservation` usecase declares a `BillingClient` interface (`internal/reservation/usecase/usecase.go:25-33`). In the Docker Compose deployment, `cmd/reservation/main.go` currently satisfies this interface with a do-nothing `stubBillingClient`. We need a real adapter that speaks gRPC to the `billing` service (port 9090 inside the Docker network).

The adapter mirrors the pattern in `tests/e2e/adapters_test.go:34-74`, but instead of wrapping an in-process `billinguc.Usecase`, it wraps a `billingv1.BillingServiceClient` generated from protobuf.

- [ ] **Step 1: Create `internal/reservation/client/billing_client.go`**

```go
// Package client provides gRPC client adapters that let the reservation service
// call downstream billing and payment microservices.
package client

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	billingmodel "parkir-pintar/internal/billing/model"
	billingv1 "parkir-pintar/proto/billing/v1"
)

// BillingClient adapts a billingv1.BillingServiceClient to the
// reservation.BillingClient interface.
type BillingClient struct {
	client billingv1.BillingServiceClient
}

// NewBillingClient creates a new BillingClient adapter.
func NewBillingClient(client billingv1.BillingServiceClient) *BillingClient {
	return &BillingClient{client: client}
}

// StartBilling calls the billing service to create a billing record with the booking fee.
func (c *BillingClient) StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) error {
	_, err := c.client.StartBilling(ctx, &billingv1.StartBillingRequest{
		ReservationId:  reservationID,
		BookingFee:     bookingFee,
		IdempotencyKey: idempotencyKey,
	})
	return err
}

// CalculateFee calls the billing service to compute parking fees.
func (c *BillingClient) CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error) {
	resp, err := c.client.CalculateFee(ctx, &billingv1.CalculateFeeRequest{
		ReservationId: reservationID,
		CheckInAt:     timestamppb.New(checkInAt),
		CheckOutAt:    timestamppb.New(checkOutAt),
	})
	if err != nil {
		return nil, err
	}
	return protoToBillingRecord(resp), nil
}

// GenerateInvoice calls the billing service to finalise a billing record into an invoice.
func (c *BillingClient) GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	resp, err := c.client.GenerateInvoice(ctx, &billingv1.GenerateInvoiceRequest{
		ReservationId:  reservationID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return protoToBillingRecord(resp), nil
}

// ApplyPenalty calls the billing service to record a penalty.
func (c *BillingClient) ApplyPenalty(ctx context.Context, reservationID string, penaltyType string, amount int64, description string) error {
	_, err := c.client.ApplyPenalty(ctx, &billingv1.ApplyPenaltyRequest{
		ReservationId: reservationID,
		PenaltyType:   penaltyType,
		Amount:        amount,
		Description:   description,
	})
	return err
}

// protoToBillingRecord maps a protobuf BillingResponse to the domain BillingRecord.
func protoToBillingRecord(r *billingv1.BillingResponse) *billingmodel.BillingRecord {
	if r == nil {
		return nil
	}
	return &billingmodel.BillingRecord{
		ID:              r.GetId(),
		ReservationID:   r.GetReservationId(),
		BookingFee:      r.GetBookingFee(),
		ParkingFee:      r.GetParkingFee(),
		OvernightFee:    r.GetOvernightFee(),
		CancellationFee: r.GetCancellationFee(),
		PenaltyAmount:   r.GetPenaltyAmount(),
		TotalAmount:     r.GetTotalAmount(),
		DurationMinutes: int(r.GetDurationMinutes()),
		BilledHours:     int(r.GetBilledHours()),
		IsOvernight:     r.GetIsOvernight(),
		IdempotencyKey:  r.GetIdempotencyKey(),
		Status:          r.GetStatus(),
	}
}
```

- [ ] **Step 2: Verify the new package compiles**

Run:
```bash
export PATH=$PATH:/usr/local/go/bin
go build ./internal/reservation/client/...
```

Expected: `ok` (no output, exit code 0)

- [ ] **Step 3: Commit**

```bash
git add internal/reservation/client/billing_client.go
git commit -m "feat(reservation): add gRPC billing client adapter"
```

---

### Task 2: Create gRPC Payment Client Adapter

**Files:**
- Create: `internal/reservation/client/payment_client.go`

**Context:**
Same pattern as Task 1, but for the `PaymentClient` interface (`internal/reservation/usecase/usecase.go:35-40`). It wraps `paymentv1.PaymentServiceClient` and only needs `ProcessPayment` for the checkout flow.

- [ ] **Step 1: Create `internal/reservation/client/payment_client.go`**

```go
package client

import (
	"context"

	paymentv1 "parkir-pintar/proto/payment/v1"
)

// PaymentClient adapts a paymentv1.PaymentServiceClient to the
// reservation.PaymentClient interface.
type PaymentClient struct {
	client paymentv1.PaymentServiceClient
}

// NewPaymentClient creates a new PaymentClient adapter.
func NewPaymentClient(client paymentv1.PaymentServiceClient) *PaymentClient {
	return &PaymentClient{client: client}
}

// ProcessPayment calls the payment service to process a payment for a billing record.
func (c *PaymentClient) ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) error {
	_, err := c.client.ProcessPayment(ctx, &paymentv1.ProcessPaymentRequest{
		BillingId:      billingID,
		Amount:         amount,
		PaymentMethod:  paymentMethod,
		IdempotencyKey: idempotencyKey,
	})
	return err
}
```

- [ ] **Step 2: Verify the new package compiles**

Run:
```bash
export PATH=$PATH:/usr/local/go/bin
go build ./internal/reservation/client/...
```

Expected: `ok` (no output, exit code 0)

- [ ] **Step 3: Commit**

```bash
git add internal/reservation/client/payment_client.go
git commit -m "feat(reservation): add gRPC payment client adapter"
```

---

### Task 3: Wire Real Billing & Payment Clients in Reservation Service Main

**Files:**
- Modify: `cmd/reservation/main.go`

**Context:**
`cmd/reservation/main.go` currently defines inline `stubBillingClient` and `stubPaymentClient` structs that do nothing. We must:
1. Import the new `client` package and the proto packages.
2. Read `GRPC_BILLING_TARGET` and `GRPC_PAYMENT_TARGET` from environment (same pattern as `cmd/gateway/main.go:63-66`).
3. Dial both services using `pkg/grpcclient`.
4. Pass the real adapters to `usecase.NewUsecase`.
5. Register connection close handlers with the shutdown manager.
6. Delete the `stubBillingClient` and `stubPaymentClient` types.

- [ ] **Step 1: Replace the entire contents of `cmd/reservation/main.go`**

```go
// Package main is the entry point for the ParkirPintar Reservation Service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"parkir-pintar/internal/natssetup"
	reservationhandler "parkir-pintar/internal/reservation/handler"
	reservationrepo "parkir-pintar/internal/reservation/repository"
	"parkir-pintar/internal/reservation/usecase"
	"parkir-pintar/internal/reservation/worker"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	"parkir-pintar/pkg/grpcclient"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
	billingv1 "parkir-pintar/proto/billing/v1"
	paymentv1 "parkir-pintar/proto/payment/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"

	"parkir-pintar/internal/reservation/client"
)

func main() {
	cfg, err := config.Load("config/.env")
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.NewLogger(cfg.Logger)
	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled: cfg.Tracing.Enabled, ServiceName: "parkir-pintar-reservation",
		SampleRate: cfg.Tracing.SampleRate, Exporter: cfg.Tracing.Exporter,
	})
	if err != nil {
		log.Warn("tracer init failed", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		log.Error("postgres connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	redisClient, err := redis.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Error("redis connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	natsClient, err := nats.NewClient(cfg.NATS.URL)
	if err != nil {
		log.Error("nats connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	if err := natssetup.SetupStreams(natsClient); err != nil {
		log.Error("nats stream setup failed", slog.Any("error", err))
		os.Exit(1)
	}

	tracedPG := database.NewTracedPostgresClient(pgClient, tracer)
	tracedRedis := redis.NewTracedRedisClient(redisClient, tracer)

	// Dial downstream billing & payment services.
	billingTarget := getEnv("GRPC_BILLING_TARGET", "localhost:9093")
	paymentTarget := getEnv("GRPC_PAYMENT_TARGET", "localhost:9094")

	clientCfg := grpcclient.ClientConfig{
		DialTimeout:      cfg.GRPC.Client.DialTimeout,
		KeepAliveTime:    cfg.GRPC.Client.KeepAliveTime,
		KeepAliveTimeout: cfg.GRPC.Client.KeepAliveTimeout,
		Tracer:           tracer,
		Logger:           log,
	}

	clientCfg.Target = billingTarget
	billingConn, err := grpcclient.Dial(context.Background(), clientCfg)
	if err != nil {
		log.Error("failed to connect to billing service", slog.Any("error", err))
		os.Exit(1)
	}

	clientCfg.Target = paymentTarget
	paymentConn, err := grpcclient.Dial(context.Background(), clientCfg)
	if err != nil {
		log.Error("failed to connect to payment service", slog.Any("error", err))
		os.Exit(1)
	}

	billingGRPC := billingv1.NewBillingServiceClient(billingConn)
	paymentGRPC := paymentv1.NewPaymentServiceClient(paymentConn)

	// Wire domain layers.
	repo := reservationrepo.NewRepository(tracedPG.GetDB())
	uc := usecase.NewUsecase(
		repo, tracedRedis, natsClient,
		client.NewBillingClient(billingGRPC),
		client.NewPaymentClient(paymentGRPC),
	)
	handler := reservationhandler.NewHandler(uc)

	// Start expiry worker.
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go worker.RunExpiryWorker(workerCtx, 30*time.Second, repo, uc)

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { workerCancel(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { natsClient.Close(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { billingConn.Close(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { paymentConn.Close(); return nil })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })

	grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, 30*time.Second)
	grpcSrv.RegisterService(&reservationv1.ReservationService_ServiceDesc, handler)
	if err := grpcSrv.Start(); err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := shutdownMgr.Shutdown(ctx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
```

- [ ] **Step 2: Verify the reservation service binary builds**

Run:
```bash
export PATH=$PATH:/usr/local/go/bin
go build ./cmd/reservation
```

Expected: `ok` (no output, exit code 0)

- [ ] **Step 3: Commit**

```bash
git add cmd/reservation/main.go
git commit -m "feat(reservation): wire real gRPC billing and payment clients"
```

---

### Task 4: Update Docker Compose for Billing/Payment Service Discovery

**Files:**
- Modify: `docker-compose.yml`

**Context:**
The `reservation` service needs to know where the `billing` and `payment` services are. Inside the Docker network they run on port `9090` (not the host-mapped ports). We must add environment variables and `depends_on` conditions.

- [ ] **Step 1: Edit the `reservation` service block in `docker-compose.yml`**

Replace the existing `reservation:` service with:

```yaml
  reservation:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: parkir-reservation
    command: ["/app/reservation"]
    ports:
      - "9091:9090"
    environment:
      APP_NAME: parkir-pintar-reservation
      APP_ENV: local
      GRPC_SERVER_PORT: "9090"
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USERNAME: ${DB_USERNAME}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_DATABASE: ${DB_DATABASE}
      DB_SSL_MODE: disable
      REDIS_HOST: redis
      REDIS_PORT: "6379"
      REDIS_PASSWORD: ${REDIS_PASSWORD}
      NATS_URL: nats://nats:4222
      TRACING_ENABLED: "false"
      GRPC_BILLING_TARGET: billing:9090
      GRPC_PAYMENT_TARGET: payment:9090
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
      billing:
        condition: service_healthy
      payment:
        condition: service_healthy
    networks:
      - backend
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "ps | grep -v grep | grep /app/reservation || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 15s
```

- [ ] **Step 2: Validate Docker Compose syntax**

Run:
```bash
docker compose config > /dev/null
```

Expected: No output, exit code 0.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "infra(docker): add billing/payment targets to reservation service"
```

---

### Task 5: Add Vehicle-Type Compatibility Validation in Reservation Usecase

**Files:**
- Modify: `internal/reservation/usecase/usecase.go`

**Context:**
In `CreateReservation`, after `GetSpotForUpdate` locks the spot, we only check `spot.Status != "available"`. We must also reject the reservation if `spot.VehicleType != req.VehicleType`. This prevents a car driver from reserving a motorcycle spot (or vice versa) in user-selected mode.

- [ ] **Step 1: Edit `internal/reservation/usecase/usecase.go`**

Find the block inside `CreateReservation`:

```go
	// Step 4: Double-check spot availability under lock
	spot, err := uc.repo.GetSpotForUpdate(ctx, spotID)
	if err != nil || spot.Status != "available" {
		return nil, apperror.New("CONFLICT", "spot no longer available", 409)
	}
```

Replace it with:

```go
	// Step 4: Double-check spot availability and vehicle-type compatibility under lock
	spot, err := uc.repo.GetSpotForUpdate(ctx, spotID)
	if err != nil || spot.Status != "available" {
		return nil, apperror.New("CONFLICT", "spot no longer available", 409)
	}
	if spot.VehicleType != req.VehicleType {
		return nil, apperror.BadRequest("spot vehicle type does not match requested vehicle type")
	}
```

- [ ] **Step 2: Commit**

```bash
git add internal/reservation/usecase/usecase.go
git commit -m "feat(reservation): validate vehicle-type compatibility for user-selected spots"
```

---

### Task 6: Add Unit Test for Vehicle-Type Mismatch

**Files:**
- Modify: `internal/reservation/usecase/usecase_test.go`

**Context:**
Add a testify-based unit test that asserts `CreateReservation` returns a bad-request error when a driver requests a `car` but selects a `motorcycle` spot.

- [ ] **Step 1: Append the new test to `internal/reservation/usecase/usecase_test.go`**

Add the following test function at the end of the file (after all existing tests):

```go
// TestCreateReservation_ShouldReject_WhenVehicleTypeMismatches verifies that
// a user-selected reservation is rejected if the spot's vehicle type does not
// match the requested vehicle type.
func TestCreateReservation_ShouldReject_WhenVehicleTypeMismatches(t *testing.T) {
	ctx := context.Background()

	repo := new(MockRepository)
	redisClient := new(MockRedisClient)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)
	uc := NewUsecase(repo, redisClient, nil, billing, payment)

	// A motorcycle spot.
	motorcycleSpot := &model.ParkingSpot{
		ID:          "spot-moto-001",
		VehicleType: "motorcycle",
		Status:      "available",
	}

	repo.On("GetSpotForUpdate", ctx, "spot-moto-001").Return(motorcycleSpot, nil)
	redisClient.On("SetNX", ctx, "lock:spot:spot-moto-001", "locked", 30*time.Second).Return(true, nil)
	redisClient.On("Delete", ctx, "lock:spot:spot-moto-001").Return(nil)

	_, err := uc.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       "driver-001",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentUserSelected,
		SpotID:         "spot-moto-001",
		IdempotencyKey: "idem-mismatch-001",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "vehicle type does not match")

	repo.AssertExpectations(t)
	redisClient.AssertExpectations(t)
}
```

- [ ] **Step 2: Run the new test to verify it passes**

Run:
```bash
export PATH=$PATH:/usr/local/go/bin
go test -v ./internal/reservation/usecase/... -run TestCreateReservation_ShouldReject_WhenVehicleTypeMismatches
```

Expected:
```
=== RUN   TestCreateReservation_ShouldReject_WhenVehicleTypeMismatches
--- PASS: TestCreateReservation_ShouldReject_WhenVehicleTypeMismatches (0.00s)
PASS
ok      parkir-pintar/internal/reservation/usecase
```

- [ ] **Step 3: Commit**

```bash
git add internal/reservation/usecase/usecase_test.go
git commit -m "test(reservation): add vehicle-type mismatch rejection test"
```

---

### Task 7: Seed a Test Driver in Docker Compose E2E Setup

**Files:**
- Modify: `tests/e2e_docker/setup_test.go`

**Context:**
All Layer 2 tests send `"test-driver-001"` as `driver_id`, but the Postgres schema requires a UUID foreign key. The fix is to insert a test driver row after the Docker Compose stack is healthy, then expose the UUID to all tests via `denv`.

- [ ] **Step 1: Edit `tests/e2e_docker/setup_test.go`**

Add `"github.com/google/uuid"` to the imports.

Add `driverID string` to `dockerEnvStruct`:

```go
type dockerEnvStruct struct {
	baseURL    string
	httpClient *http.Client
	jwtToken   string
	driverID   string // seeded test driver UUID
}
```

In `TestMain`, after `log.Println("Gateway is healthy.")`, insert:

```go
	// Seed a test driver into Postgres so reservations have a valid FK.
	driverID := uuid.New().String()
	if err := seedTestDriver(driverID); err != nil {
		tearDown()
		log.Fatalf("seed test driver failed: %v", err)
	}
	log.Printf("Seeded test driver: %s", driverID)
```

Change the JWT generation line from:
```go
	token := testhelpers.GenerateTestJWT("test-driver-001", "driver", "test-jwt-secret")
```
to:
```go
	token := testhelpers.GenerateTestJWT(driverID, "driver", "test-jwt-secret")
```

Add `driverID: driverID` to the `denv` struct literal.

Add the `seedTestDriver` and `getEnv` helper functions at the bottom of the file (before the closing brace of the package):

```go
// seedTestDriver inserts a single driver row into the Postgres container.
func seedTestDriver(driverID string) error {
	phone := fmt.Sprintf("+628%010d", os.Getpid())
	cmd := exec.Command("docker", "exec", "assessment-postgres", "psql",
		"-U", getEnv("DB_USERNAME", "parkir_user"),
		"-d", getEnv("DB_DATABASE", "parkir_pintar"),
		"-c", fmt.Sprintf(
			`INSERT INTO drivers (id, name, phone, email, vehicle_type, vehicle_plate) VALUES ('%s', 'E2E Test Driver', '%s', 'e2e@test.local', 'car', 'B 1234 E2E') ON CONFLICT (id) DO NOTHING`,
			driverID, phone,
		),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("seed driver: %w (output: %s)", err, string(out))
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
```

- [ ] **Step 2: Verify setup_test.go compiles**

Run:
```bash
export PATH=$PATH:/usr/local/go/bin
go build ./tests/e2e_docker/...
```

Expected: `ok` (no output, exit code 0)

- [ ] **Step 3: Commit**

```bash
git add tests/e2e_docker/setup_test.go
git commit -m "test(e2e_docker): seed test driver before running Layer 2 tests"
```

---

### Task 8: Update All E2E Docker Tests to Use the Seeded Driver UUID

**Files:**
- Modify: `tests/e2e_docker/happy_path_test.go`
- Modify: `tests/e2e_docker/cancellation_test.go`
- Modify: `tests/e2e_docker/double_book_test.go`
- Modify: `tests/e2e_docker/payment_test.go`
- Modify: `tests/e2e_docker/middleware_test.go`

**Context:**
Every test that sends `"test-driver-001"` as `driver_id` in a JSON body must instead send `denv.driverID`.

- [ ] **Step 1: Replace `driver_id` values in `happy_path_test.go`**

Find:
```go
		"driver_id":       "test-driver-001",
```
Replace with:
```go
		"driver_id":       denv.driverID,
```

- [ ] **Step 2: Replace `driver_id` values in `cancellation_test.go`**

Same replacement as Step 1.

- [ ] **Step 3: Replace `driver_id` values in `double_book_test.go`**

Same replacement as Step 1. There are two occurrences (one in each goroutine).

- [ ] **Step 4: Replace `driver_id` values in `payment_test.go`**

Same replacement as Step 1.

- [ ] **Step 5: Replace `driver_id` values in `middleware_test.go`**

Same replacement as Step 1. Note: this file also sends `"test-driver-001"` in the JWT and in request bodies.

- [ ] **Step 6: Verify all e2e_docker tests compile**

Run:
```bash
export PATH=$PATH:/usr/local/go/bin
go build ./tests/e2e_docker/...
```

Expected: `ok` (no output, exit code 0)

- [ ] **Step 7: Commit**

```bash
git add tests/e2e_docker/
git commit -m "test(e2e_docker): use seeded UUID driver_id in all Layer 2 tests"
```

---

### Task 9: Run Full Test Suite and Verify Layer 2 E2E Tests Pass

**Files:**
- All of the above (verification step)

- [ ] **Step 1: Run unit and integration tests**

```bash
export PATH=$PATH:/usr/local/go/bin
go test ./... -count=1
```

Expected: All packages pass (including `internal/reservation/usecase` with the new test).

- [ ] **Step 2: Run Layer 1 E2E tests**

```bash
export PATH=$PATH:/usr/local/go/bin
go test -v -timeout 300s ./tests/e2e/...
```

Expected: All tests pass.

- [ ] **Step 3: Run Layer 2 E2E tests**

```bash
export PATH=$PATH:/usr/local/go/bin
go test -v -timeout 600s ./tests/e2e_docker/...
```

Expected: All tests pass, including:
- `TestDockerHappyPath_ShouldCompleteFullLifecycle`
- `TestDockerCancellation_ShouldReturnZeroFee_WhenCancelledImmediately`
- `TestDockerDoubleBook_ShouldRejectSecond_WhenSameSpot`
- `TestDockerPayment_ShouldReturnPaymentStatus`
- `TestDockerMiddleware_ShouldEnforceAuth`

- [ ] **Step 4: Commit** (if any final fixes were needed)

```bash
git add -A
git commit -m "fix: resolve E2E test failures — billing/payment wiring, vehicle validation, driver seeding"
```

---

## Self-Review Checklist

### 1. Spec Coverage

| Issue | Task(s) |
|-------|---------|
| Billing & Payment stubbed in Docker Compose | Tasks 1–4 |
| Vehicle type mismatch allowed | Tasks 5–6 |
| Invalid driver_id (string instead of UUID) in e2e_docker tests | Tasks 7–8 |

### 2. Placeholder Scan
- No "TBD", "TODO", "implement later", or "fill in details" found.
- Every step contains exact file paths, exact code, and exact commands.
- No vague directives like "add appropriate error handling" — the exact validation line is shown.

### 3. Type Consistency
- `BillingClient` interface method signatures match between `usecase.go` and `billing_client.go`.
- `PaymentClient` interface method signatures match between `usecase.go` and `payment_client.go`.
- `protoToBillingRecord` maps every field from `billingv1.BillingResponse` to `billingmodel.BillingRecord` using matching getter names.

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-05-fix-e2e-test-failures.md`.**

Two execution options:

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

2. **Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
