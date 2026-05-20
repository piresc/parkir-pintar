package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	searchgrpc "parkir-pintar/internal/search/handler/grpc"
	searchnats "parkir-pintar/internal/search/handler/nats"
	searchrepo "parkir-pintar/internal/search/repository"
	searchsync "parkir-pintar/internal/search/sync"
	searchuc "parkir-pintar/internal/search/usecase"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/metrics"
	pkgnats "parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redis"
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
	cfg, err := config.LoadConfig("search")
	if err != nil {
		return err
	}

	// --- Telemetry ---
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)

	providers, telErr := telemetry.Init(context.Background(), telemetry.Config{
		ServiceName:     "parkir-pintar-search",
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
		ServiceName:  "parkir-pintar-search",
		SampleRate:   cfg.Tracing.SampleRate,
		Exporter:     cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", logger.Err(err))
		tracer = tracing.NewNoOpTracer()
	}

	metricsInst, err := metrics.NewMetrics("parkir-pintar-search", otlpEndpoint)
	if err != nil {
		return err
	}

	// --- Infrastructure ---
	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		return err
	}
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return err
	}

	tracedPG := database.NewTracedPostgresClient(pgClient, tracer)
	tracedRedis := redis.NewTracedRedisClient(redisClient, tracer)

	// --- Layers ---
	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, redisClient)

	repo := searchrepo.NewRepository(tracedPG.GetDB())
	uc := searchuc.NewUsecase(repo, tracedRedis)
	handler := searchgrpc.NewHandler(uc)

	// --- Shutdown ---
	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })

	// --- NATS JetStream consumer (spot updates from reservation service) ---
	if cfg.NATS.Enabled {
		natsClient, natsErr := pkgnats.NewClient(cfg.NATS.URL)
		if natsErr != nil {
			return fmt.Errorf("nats connect: %w", natsErr)
		}

		natsCtx := context.Background()
		if err := pkgnats.CreateStreams(natsCtx, natsClient); err != nil {
			return fmt.Errorf("nats create streams: %w", err)
		}
		if err := pkgnats.CreateConsumersForService(natsCtx, natsClient, "search"); err != nil {
			return fmt.Errorf("nats create consumers: %w", err)
		}

		readModelRepo := searchrepo.NewReadModelRepository(tracedPG.GetDB())
		spotSync := searchsync.NewSpotSync(readModelRepo)
		natsHandler := searchnats.NewHandler(spotSync, tracedRedis, natsClient, 0)
		cc, err := natsHandler.InitConsumers()
		if err != nil {
			return fmt.Errorf("nats consumer init: %w", err)
		}

		shutdownMgr.Register(func(_ context.Context) error { cc.Stop(); return nil })
		shutdownMgr.Register(func(ctx context.Context) error { natsClient.Close(ctx); return nil })
	}

	shutdownMgr.Register(func(ctx context.Context) error { return metricsInst.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })
	if providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return providers.Shutdown(ctx) })
	}

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
