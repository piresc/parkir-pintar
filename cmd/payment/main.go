// Package main is the entry point for the ParkirPintar Payment Service.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	paymentgateway "parkir-pintar/internal/payment/gateway"
	paymenthandler "parkir-pintar/internal/payment/handler"
	paymentrepo "parkir-pintar/internal/payment/repository"
	paymentuc "parkir-pintar/internal/payment/usecase"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/metrics"
	pkgnats "parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
	paymentv1 "parkir-pintar/proto/payment/v1"

	"google.golang.org/grpc"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func main() {
	cfg, err := config.LoadConfig("payment")
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.NewLogger(cfg.Logger)
	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled: cfg.Tracing.Enabled, ServiceName: "parkir-pintar-payment",
		SampleRate: cfg.Tracing.SampleRate, Exporter: cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)
	metricsInst, err := metrics.NewMetrics("parkir-pintar-payment", otlpEndpoint)
	if err != nil {
		log.Error("metrics init failed", slog.Any("error", err))
		os.Exit(1)
	}

	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		log.Error("postgres connect failed", slog.Any("error", err))
		os.Exit(1)
	}

	tracedPG := database.NewTracedPostgresClient(pgClient, tracer)

	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, nil)

	repo := paymentrepo.NewRepository(tracedPG.GetDB())
	gw := paymentgateway.NewStubGateway(false)

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })

	// NATS setup (publish-only for payment results)
	var paymentPublisher paymentuc.EventPublisher
	if cfg.NATS.Enabled {
		natsClient, natsErr := pkgnats.NewClient(cfg.NATS.URL)
		if natsErr != nil {
			log.Error("nats connect failed", slog.Any("error", natsErr))
			os.Exit(1)
		}

		natsCtx := context.Background()
		if err := pkgnats.CreateStreams(natsCtx, natsClient); err != nil {
			log.Error("nats create streams failed", slog.Any("error", err))
			os.Exit(1)
		}

		publisher := pkgnats.NewPublisher(natsClient)
		paymentPublisher = paymentgateway.NewPaymentEventPublisher(publisher)

		shutdownMgr.Register(func(ctx context.Context) error { natsClient.Close(ctx); return nil })
	}

	uc := paymentuc.NewUsecase(repo, gw, paymentPublisher)
	handler := paymenthandler.NewHandler(uc)

	shutdownMgr.Register(func(ctx context.Context) error { return metricsInst.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })

	grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, cfg.GRPC.Server.RequestTimeout,
		grpc.ChainUnaryInterceptor(
			metricsInst.GRPCUnaryInterceptor(),
			interceptors.RecoveryUnaryInterceptor(),
			interceptors.AuthUnaryInterceptor(nil),
			interceptors.LoggingUnaryInterceptor(),
			interceptors.TracingUnaryInterceptor(),
			interceptors.RateLimitUnaryInterceptor(grpcmiddleware.RateLimitConfig{
				RequestsPerSecond: cfg.GRPC.RateLimit.RequestsPerSecond,
				BurstSize:         cfg.GRPC.RateLimit.BurstSize,
				CleanupInterval:   5 * time.Minute,
			}),
		),
	)
	grpcSrv.RegisterService(&paymentv1.PaymentService_ServiceDesc, handler)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- grpcSrv.Start()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			log.Error("gRPC server error", slog.Any("error", err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.Server.ShutdownTimeout)
	defer cancel()
	if err := shutdownMgr.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
	}

	select {
	case err := <-serverErr:
		if err != nil {
			log.Error("server exited with error", slog.Any("error", err))
		}
	default:
	}
}
