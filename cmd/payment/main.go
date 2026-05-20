package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	paymentgateway "parkir-pintar/internal/payment/gateway"
	paymenthandler "parkir-pintar/internal/payment/handler/grpc"
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
	"parkir-pintar/pkg/telemetry"
	"parkir-pintar/pkg/tracing"

	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", logger.Err(err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig("payment")
	if err != nil {
		return err
	}

	// --- Telemetry ---
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)

	providers, telErr := telemetry.Init(context.Background(), telemetry.Config{
		ServiceName:     "parkir-pintar-payment",
		OTLPEndpoint:    otlpEndpoint,
		TraceSampleRate: cfg.Tracing.SampleRate,
	})
	if telErr != nil {
		slog.Warn("telemetry init failed, continuing with noop", logger.Err(telErr))
	}

	var log *slog.Logger
	if providers != nil && providers.LoggerProvider != nil {
		log = logger.NewLoggerWithProvider(cfg.Logger, providers.LoggerProvider)
	} else {
		log = logger.NewLogger(cfg.Logger)
	}

	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled:      cfg.Tracing.Enabled,
		ServiceName:  "parkir-pintar-payment",
		SampleRate:   cfg.Tracing.SampleRate,
		Exporter:     cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", logger.Err(err))
		tracer = tracing.NewNoOpTracer()
	}

	metricsInst, err := metrics.NewMetrics("parkir-pintar-payment", otlpEndpoint)
	if err != nil {
		return err
	}

	// --- Infrastructure ---
	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		return err
	}

	tracedPG := database.NewTracedPostgresClient(pgClient, tracer)

	// --- Messaging (NATS) ---
	var natsClient *pkgnats.Client
	var publisher paymentuc.EventPublisher

	if cfg.NATS.Enabled {
		natsClient, err = pkgnats.NewClient(cfg.NATS.URL)
		if err != nil {
			return fmt.Errorf("nats connect: %w", err)
		}

		natsCtx := context.Background()
		if err := pkgnats.CreateStreams(natsCtx, natsClient); err != nil {
			return fmt.Errorf("nats create streams: %w", err)
		}

		pub := pkgnats.NewPublisher(natsClient)
		publisher = paymentgateway.NewPaymentEventPublisher(pub)
	}

	// --- Layers ---
	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, nil)

	repo := paymentrepo.NewRepository(tracedPG.GetDB())
	gw := paymentgateway.NewStubGateway(false)
	uc := paymentuc.NewUsecase(repo, gw, publisher)
	handler := paymenthandler.NewHandler(uc)

	// --- gRPC Server ---
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
	handler.RegisterService(grpcSrv.Server())

	// --- Shutdown ---
	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	if natsClient != nil {
		shutdownMgr.Register(func(ctx context.Context) error { natsClient.Close(ctx); return nil })
	}
	shutdownMgr.Register(func(ctx context.Context) error { return metricsInst.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })
	if providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return providers.Shutdown(ctx) })
	}

	// --- Run ---
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
			log.Error("gRPC server error", logger.Err(err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.Server.ShutdownTimeout)
	defer cancel()
	if err := shutdownMgr.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", logger.Err(err))
	}

	select {
	case err := <-serverErr:
		if err != nil {
			return err
		}
	default:
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
