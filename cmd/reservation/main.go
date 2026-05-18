// Package main is the entry point for the ParkirPintar Reservation Service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	reservationgateway "parkir-pintar/internal/reservation/gateway"
	reservationhandler "parkir-pintar/internal/reservation/handler"
	"parkir-pintar/internal/reservation/model"
	reservationrepo "parkir-pintar/internal/reservation/repository"
	"parkir-pintar/internal/reservation/usecase"
	"parkir-pintar/internal/reservation/worker"
	taskqueue "parkir-pintar/pkg/asynq"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	"parkir-pintar/pkg/grpcclient"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/metrics"
	pkgnats "parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/redislock"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/telemetry"
	"parkir-pintar/pkg/tracing"
	billingv1 "parkir-pintar/proto/billing/v1"
	paymentv1 "parkir-pintar/proto/payment/v1"
	presencev1 "parkir-pintar/proto/presence/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"

	"google.golang.org/grpc"

	"parkir-pintar/internal/reservation/client"
)

func main() {
	cfg, err := config.Load("config/.env")
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize unified telemetry (traces, metrics, logs via OTLP)
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)
	providers, telErr := telemetry.Init(context.Background(), telemetry.Config{
		ServiceName:     "parkir-pintar-reservation",
		OTLPEndpoint:    otlpEndpoint,
		TraceSampleRate: cfg.Tracing.SampleRate,
	})
	if telErr != nil {
		slog.Warn("telemetry init failed, continuing with noop", slog.Any("error", telErr))
	}

	// Initialize logger (with OTLP log export if available)
	var log *slog.Logger
	if providers != nil && providers.LoggerProvider != nil {
		log = logger.NewLoggerWithProvider(cfg.Logger, providers.LoggerProvider)
	} else {
		log = logger.NewLogger(cfg.Logger)
	}

	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled: cfg.Tracing.Enabled, ServiceName: "parkir-pintar-reservation",
		SampleRate: cfg.Tracing.SampleRate, Exporter: cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	metricsInst, err := metrics.NewMetrics("parkir-pintar-reservation", otlpEndpoint)
	if err != nil {
		log.Error("metrics init failed", slog.Any("error", err))
		os.Exit(1)
	}

	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		log.Error("postgres connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Error("redis connect failed", slog.Any("error", err))
		os.Exit(1)
	}

	tracedPG := database.NewTracedPostgresClient(pgClient, tracer)
	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, redisClient)

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

	// Dial presence service (optional — graceful degradation if unavailable).
	var presenceClient usecase.PresenceClient
	presenceTarget := getEnv("GRPC_PRESENCE_TARGET", "localhost:9095")
	clientCfg.Target = presenceTarget
	presenceConn, presenceErr := grpcclient.Dial(context.Background(), clientCfg)
	if presenceErr != nil {
		log.Warn("presence service unavailable, presence verification disabled", slog.Any("error", presenceErr))
	} else {
		presenceClient = &presenceClientAdapter{inner: client.NewPresenceClient(presencev1.NewPresenceServiceClient(presenceConn))}
	}

	// Wire domain layers.
	repo := reservationrepo.NewRepository(tracedPG.GetDB())
	redislockLocker, err := redislock.NewLocker(redisClient, redislock.Config{
		TTL:           12 * time.Minute,
		RetryAttempts: 0,
	})
	if err != nil {
		log.Error("failed to create locker", slog.Any("error", err))
		os.Exit(1)
	}
	// Start Asynq task queue (delayed tasks: reservation expiry, payment hold timeout).
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	asynqClient := taskqueue.NewClient(redisAddr, cfg.Redis.Password)
	asynqServer := taskqueue.NewServer(redisAddr, cfg.Redis.Password, cfg.Asynq.Concurrency)

	// NATS JetStream (event-driven messaging)
	var natsClient *pkgnats.Client
	var eventPublisher reservationgateway.EventPublisher
	if cfg.NATS.Enabled {
		var natsErr error
		natsClient, natsErr = pkgnats.NewClient(cfg.NATS.URL)
		if natsErr != nil {
			log.Error("nats connect failed", slog.Any("error", natsErr))
			os.Exit(1)
		}

		natsCtx := context.Background()
		if err := pkgnats.CreateStreams(natsCtx, natsClient); err != nil {
			log.Error("nats create streams failed", slog.Any("error", err))
			os.Exit(1)
		}
		if err := pkgnats.CreateConsumersForService(natsCtx, natsClient, "reservation"); err != nil {
			log.Error("nats create consumers failed", slog.Any("error", err))
			os.Exit(1)
		}

		publisher := pkgnats.NewPublisher(natsClient)
		eventPublisher = reservationgateway.NewNATSEventPublisher(publisher)
	}

	uc := usecase.NewUsecase(
		repo, usecase.NewLockerAdapter(redislockLocker),
		client.NewBillingClient(billingGRPC),
		client.NewPaymentClient(paymentGRPC),
		presenceClient,
		asynqClient,
		eventPublisher,
		cfg.Reservation.ExpiryTimeoutMinutes,
	)
	handler := reservationhandler.NewHandler(uc)

	// Start legacy polling workers (fallback — catches anything Asynq misses).
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go worker.RunExpiryWorker(workerCtx, cfg.Reservation.WorkerPollInterval, repo, uc)
	go worker.RunPaymentTimeoutWorker(workerCtx, cfg.Reservation.WorkerPollInterval, repo, uc, cfg.Reservation.PaymentTimeoutMinutes)

	expiryHandler := taskqueue.NewReservationExpiryHandler(&usecaseExpirerAdapter{uc: uc})
	paymentHandler := taskqueue.NewPaymentHoldTimeoutHandler(&usecaseFailerAdapter{uc: uc})
	asynqServer.RegisterHandlers(expiryHandler, paymentHandler)

	go func() {
		if err := asynqServer.Start(); err != nil {
			log.Error("asynq server error", slog.Any("error", err))
		}
	}()

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { workerCancel(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { asynqServer.Shutdown(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { return asynqClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return billingConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return paymentConn.Close() })
	shutdownMgr.Register(func(ctx context.Context) error { return metricsInst.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })
	if providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return providers.Shutdown(ctx) })
	}

	// Start NATS consumer for payment results.
	if cfg.NATS.Enabled && natsClient != nil {
		natsHandler := reservationhandler.NewNATSHandler(uc, natsClient)
		if err := natsHandler.Start(); err != nil {
			log.Error("nats consumer start failed", slog.Any("error", err))
			os.Exit(1)
		}
		shutdownMgr.Register(func(_ context.Context) error { natsHandler.Stop(); return nil })
		shutdownMgr.Register(func(ctx context.Context) error { natsClient.Close(ctx); return nil })
	}

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
	grpcSrv.RegisterService(&reservationv1.ReservationService_ServiceDesc, handler)

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
			os.Exit(1)
		}
	default:
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// usecaseExpirerAdapter adapts the reservation usecase to the taskqueue.ReservationExpirer interface.
type usecaseExpirerAdapter struct {
	uc usecase.Usecase
}

func (a *usecaseExpirerAdapter) ExpireReservation(ctx context.Context, reservationID string) error {
	return a.uc.ExpireReservation(ctx, &model.ExpireReservationRequest{
		ReservationID: reservationID,
	})
}

// usecaseFailerAdapter adapts the reservation usecase to the taskqueue.ReservationFailer interface.
type usecaseFailerAdapter struct {
	uc usecase.Usecase
}

func (a *usecaseFailerAdapter) FailReservation(ctx context.Context, reservationID string, _ string) error {
	return a.uc.FailReservation(ctx, &model.FailReservationRequest{
		ReservationID: reservationID,
	})
}

// presenceClientAdapter adapts the client.PresenceClient to the usecase.PresenceClient interface.
type presenceClientAdapter struct {
	inner client.PresenceClient
}

func (a *presenceClientAdapter) VerifyPresence(ctx context.Context, driverID string, reservationID string, floorNumber int, spotNumber int) (*usecase.PresenceResult, error) {
	result, err := a.inner.VerifyPresence(ctx, driverID, reservationID, floorNumber, spotNumber)
	if err != nil {
		return nil, err
	}
	return &usecase.PresenceResult{
		Verified: result.Verified,
		Message:  result.Message,
	}, nil
}
