package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"parkir-pintar/internal/analytics/constants"
	grpchandler "parkir-pintar/internal/analytics/handler/grpc"
	natshandler "parkir-pintar/internal/analytics/handler/nats"
	analyticsrepo "parkir-pintar/internal/analytics/repository"
	analyticsuc "parkir-pintar/internal/analytics/usecase"
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

	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", logger.Err(err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig("analytics")
	if err != nil {
		return err
	}

	// --- Telemetry ---
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)

	providers, telErr := telemetry.Init(context.Background(), telemetry.Config{
		ServiceName:     "parkir-pintar-analytics",
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
		ServiceName:  "parkir-pintar-analytics",
		SampleRate:   cfg.Tracing.SampleRate,
		Exporter:     cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", logger.Err(err))
		tracer = tracing.NewNoOpTracer()
	}

	metricsInst, err := metrics.NewMetrics("parkir-pintar-analytics", otlpEndpoint)
	if err != nil {
		return err
	}

	// --- Infrastructure ---
	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		return err
	}

	tracedPG := database.NewTracedPostgresClient(pgClient, tracer)

	// --- Layers ---
	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, nil)

	repo := analyticsrepo.NewRepository(tracedPG.GetDB())
	uc := analyticsuc.NewUsecase(repo)
	handler := grpchandler.NewHandler(uc)

	// --- Shutdown ---
	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })

	// --- NATS JetStream consumer (reservation analytics events) ---
	if cfg.NATS.Enabled {
		natsClient, natsErr := pkgnats.NewClient(cfg.NATS.URL)
		if natsErr != nil {
			return fmt.Errorf("nats connect: %w", natsErr)
		}

		natsCtx := context.Background()
		streamConfigs := []pkgnats.StreamConfig{
			{Name: constants.StreamReservationAnalytics, Subjects: []string{constants.SubjectPatternAnalytics}, Retention: jetstream.LimitsPolicy, Storage: jetstream.FileStorage, MaxAge: 7 * 24 * time.Hour},
			{Name: constants.StreamReservationSearch, Subjects: []string{constants.SubjectPatternSearch}, Retention: jetstream.LimitsPolicy, Storage: jetstream.FileStorage, MaxAge: 7 * 24 * time.Hour},
		}
		if err := pkgnats.CreateStreams(natsCtx, natsClient, streamConfigs); err != nil {
			return fmt.Errorf("nats create streams: %w", err)
		}

		// Analytics event consumer
		consumerCfg := pkgnats.ConsumerConfig{
			Stream:        constants.StreamReservationAnalytics,
			Name:          constants.ConsumerAnalytics,
			FilterSubject: constants.SubjectPatternAnalytics,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		}
		if _, err := natsClient.CreateConsumer(natsCtx, consumerCfg.Stream, consumerCfg.ToJetStreamConfig()); err != nil {
			return fmt.Errorf("nats create consumer: %w", err)
		}

		// Spot snapshot consumer
		spotConsumerCfg := pkgnats.ConsumerConfig{
			Stream:        constants.StreamReservationSearch,
			Name:          constants.ConsumerAnalyticsSpot,
			FilterSubject: constants.SubjectPatternSearch,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		}
		if _, err := natsClient.CreateConsumer(natsCtx, spotConsumerCfg.Stream, spotConsumerCfg.ToJetStreamConfig()); err != nil {
			return fmt.Errorf("nats create spot consumer: %w", err)
		}

		natsH := natshandler.NewHandler(uc, natsClient)
		cc, err := natsH.InitConsumers()
		if err != nil {
			return fmt.Errorf("nats consumer init: %w", err)
		}

		spotCC, err := natsH.InitSpotConsumer()
		if err != nil {
			return fmt.Errorf("nats spot consumer init: %w", err)
		}

		shutdownMgr.Register(func(_ context.Context) error { cc.Stop(); return nil })
		shutdownMgr.Register(func(_ context.Context) error { spotCC.Stop(); return nil })
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
