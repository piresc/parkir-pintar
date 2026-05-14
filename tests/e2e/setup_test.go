// Package e2e_test provides Layer 1 E2E integration tests for ParkirPintar.
//
// This file contains TestMain which bootstraps real PostgreSQL and Redis
// containers via testcontainers-go, applies migrations,
// wires all repositories and usecases, and tears everything down on exit.
//
// Best practices applied (from Go coding standards):
// - Use context.Context as first parameter for consistency
// - Handle errors explicitly with proper wrapping
// - Use keyed fields in struct literals to prevent breakages during refactors
// - Never ignore errors
package e2e_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	billingrepo "parkir-pintar/internal/billing/repository"
	billinguc "parkir-pintar/internal/billing/usecase"
	"parkir-pintar/internal/payment/gateway"
	paymentrepo "parkir-pintar/internal/payment/repository"
	paymentuc "parkir-pintar/internal/payment/usecase"
	reservationrepo "parkir-pintar/internal/reservation/repository"
	reservationuc "parkir-pintar/internal/reservation/usecase"
	searchrepo "parkir-pintar/internal/search/repository"
	searchuc "parkir-pintar/internal/search/usecase"
)

// testEnvStruct holds shared infrastructure and service instances for all
// Layer 1 E2E tests. It is populated once in TestMain and accessed via the
// package-level `env` variable.
type testEnvStruct struct {
	db              *sqlx.DB
	redisClient     *redis.Client
	reservationUC   reservationuc.Usecase
	billingUC       billinguc.Usecase
	paymentUC       paymentuc.Usecase
	searchUC        searchuc.Usecase
	reservationRepo reservationrepo.Repository
	billingRepo     billingrepo.Repository
	paymentRepo     paymentrepo.Repository
	searchRepo      searchrepo.Repository
	paymentGW       *gateway.StubGateway
}

// env is the package-level test environment accessible by all test functions.
var env *testEnvStruct

// TestMain bootstraps real infrastructure containers, applies migrations,
// wires all domain layers, runs the test suite, and tears down containers.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// -----------------------------------------------------------------------
	// 1. Start containers
	// -----------------------------------------------------------------------

	// PostgreSQL 14
	pgContainer, err := tcpostgres.Run(ctx, "postgres:14",
		tcpostgres.WithDatabase("parkir_pintar_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}

	// Redis 7
	redisContainer, err := tcredis.Run(ctx, "redis:7")
	if err != nil {
		log.Fatalf("start redis container: %v", err)
	}

	// -----------------------------------------------------------------------
	// 2. Obtain connection strings
	// -----------------------------------------------------------------------

	pgConnStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("postgres connection string: %v", err)
	}

	redisEndpoint, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("redis connection string: %v", err)
	}

	// -----------------------------------------------------------------------
	// 3. Create clients
	// -----------------------------------------------------------------------

	db, err := sqlx.Connect("pgx", pgConnStr)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}

	redisOpts, err := redis.ParseURL(redisEndpoint)
	if err != nil {
		log.Fatalf("parse redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("ping redis: %v", err)
	}

	// -----------------------------------------------------------------------
	// 4. Apply migrations
	// -----------------------------------------------------------------------

	migrations := []string{
		"../../db/migrations/000001_init.up.sql",
		"../../db/migrations/000002_parkir_pintar.up.sql",
		"../../db/migrations/000003_payment_flow.up.sql",
		"../../db/migrations/000004_schema_per_service.up.sql",
	}
	for _, path := range migrations {
		sql, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := db.ExecContext(ctx, string(sql)); err != nil {
			log.Fatalf("apply migration %s: %v", path, err)
		}
	}

	// Set default search_path at database level so all pool connections inherit it
	var dbName string
	_ = db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName)
	_, _ = db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s SET search_path TO reservation, billing, payment, search, public", dbName))
	db.Close()
	var reconnectErr error
	for i := 0; i < 5; i++ {
		db, reconnectErr = sqlx.Connect("pgx", pgConnStr)
		if reconnectErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if reconnectErr != nil {
		log.Fatalf("reconnect to postgres after search_path change: %v", reconnectErr)
	}

	// -----------------------------------------------------------------------
	// 5. Wire repositories, adapters, and usecases
	// -----------------------------------------------------------------------

	// Repositories
	resRepo := reservationrepo.NewRepository(db)
	billRepo := billingrepo.NewRepository(db)
	payRepo := paymentrepo.NewRepository(db)
	srchRepo := searchrepo.NewRepository(db)

	// Stub / adapters
	stubGW := gateway.NewStubGateway(false)

	redisAdapter := &reservationLockerAdapter{client: redisClient}
	srchRedisAdapter := &searchRedisAdapter{client: redisClient}

	// Usecases
	billUC := billinguc.NewUsecase(billRepo)
	payUC := paymentuc.NewUsecase(payRepo, stubGW)

	billAdapter := &billingAdapter{uc: billUC}
	payAdapter := &paymentAdapter{uc: payUC}

	resUC := reservationuc.NewUsecase(resRepo, redisAdapter, billAdapter, payAdapter)
	srchUC := searchuc.NewUsecase(srchRepo, srchRedisAdapter)

	// -----------------------------------------------------------------------
	// 6. Populate test environment
	// -----------------------------------------------------------------------

	env = &testEnvStruct{
		db:              db,
		redisClient:     redisClient,
		reservationUC:   resUC,
		billingUC:       billUC,
		paymentUC:       payUC,
		searchUC:        srchUC,
		reservationRepo: resRepo,
		billingRepo:     billRepo,
		paymentRepo:     payRepo,
		searchRepo:      srchRepo,
		paymentGW:       stubGW,
	}

	// -----------------------------------------------------------------------
	// 7. Run tests then tear down
	// -----------------------------------------------------------------------

	code := m.Run()

	// Cleanup
	db.Close()
	_ = redisClient.Close()

	_ = pgContainer.Terminate(ctx)
	_ = redisContainer.Terminate(ctx)

	os.Exit(code)
}
