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
