// Package main is the entry point for the ParkirPintar Reservation Service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"parkir-pintar/internal/natssetup"
	billingmodel "parkir-pintar/internal/billing/model"
	reservationhandler "parkir-pintar/internal/reservation/handler"
	reservationrepo "parkir-pintar/internal/reservation/repository"
	"parkir-pintar/internal/reservation/usecase"
	"parkir-pintar/internal/reservation/worker"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
	reservationv1 "parkir-pintar/proto/reservation/v1"
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

	// Wire domain layers
	// BillingClient and PaymentClient are stub implementations for now;
	// in production these would be gRPC client adapters.
	repo := reservationrepo.NewRepository(tracedPG.GetDB())
	uc := usecase.NewUsecase(repo, tracedRedis, natsClient, &stubBillingClient{}, &stubPaymentClient{})
	handler := reservationhandler.NewHandler(uc)

	// Start expiry worker
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go worker.RunExpiryWorker(workerCtx, 30*time.Second, repo, uc)

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { workerCancel(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { natsClient.Close(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })
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

// stubBillingClient is a minimal billing client for standalone operation.
type stubBillingClient struct{}

func (s *stubBillingClient) StartBilling(_ context.Context, _ string, _ int64, _ string) error {
	return nil
}
func (s *stubBillingClient) CalculateFee(_ context.Context, _ string, _, _ time.Time) (*billingmodel.BillingRecord, error) {
	return &billingmodel.BillingRecord{}, nil
}
func (s *stubBillingClient) GenerateInvoice(_ context.Context, _ string, _ string) (*billingmodel.BillingRecord, error) {
	return &billingmodel.BillingRecord{}, nil
}
func (s *stubBillingClient) ApplyPenalty(_ context.Context, _ string, _ string, _ int64, _ string) error {
	return nil
}

// stubPaymentClient is a minimal payment client for standalone operation.
type stubPaymentClient struct{}

func (s *stubPaymentClient) ProcessPayment(_ context.Context, _ string, _ int64, _ string, _ string) error {
	return nil
}
