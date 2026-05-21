package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	reservation "parkir-pintar/internal/reservation"
	"parkir-pintar/internal/reservation/constants"
	grpcgw "parkir-pintar/internal/reservation/gateway/grpc"
	natsgateway "parkir-pintar/internal/reservation/gateway/nats"
	reservationasynq "parkir-pintar/internal/reservation/handler/asynq"
	grpchandler "parkir-pintar/internal/reservation/handler/grpc"
	natshandler "parkir-pintar/internal/reservation/handler/nats"
	"parkir-pintar/internal/reservation/model"
	reservationrepo "parkir-pintar/internal/reservation/repository"
	"parkir-pintar/internal/reservation/usecase"
	taskqueue "parkir-pintar/pkg/asynq"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
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
	presencev1 "parkir-pintar/proto/presence/v1"

	billingv1 "parkir-pintar/proto/billing/v1"
	paymentv1 "parkir-pintar/proto/payment/v1"

	"parkir-pintar/pkg/grpcclient"

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
	cfg, err := config.LoadConfig("reservation")
	if err != nil {
		return err
	}

	// --- Telemetry ---
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)

	providers, telErr := telemetry.Init(context.Background(), telemetry.Config{
		ServiceName:     "parkir-pintar-reservation",
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
		ServiceName:  "parkir-pintar-reservation",
		SampleRate:   cfg.Tracing.SampleRate,
		Exporter:     cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", logger.Err(err))
		tracer = tracing.NewNoOpTracer()
	}

	metricsInst, err := metrics.NewMetrics("parkir-pintar-reservation", otlpEndpoint)
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

	// --- gRPC Clients ---
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
		return err
	}

	clientCfg.Target = paymentTarget
	paymentConn, err := grpcclient.Dial(context.Background(), clientCfg)
	if err != nil {
		_ = billingConn.Close()
		return err
	}

	billingGRPC := billingv1.NewBillingServiceClient(billingConn)
	paymentGRPC := paymentv1.NewPaymentServiceClient(paymentConn)

	// Presence is optional — graceful degradation if unavailable.
	presenceTarget := getEnv("GRPC_PRESENCE_TARGET", "localhost:9095")
	clientCfg.Target = presenceTarget
	var presenceConn *grpc.ClientConn
	presenceConn, presenceErr := grpcclient.Dial(context.Background(), clientCfg)
	if presenceErr != nil {
		log.Warn("presence service unavailable, presence verification disabled", logger.Err(presenceErr))
		presenceConn = nil
	}

	// --- Redis Lock ---
	redislockLocker, err := redislock.NewLocker(redisClient, redislock.Config{
		TTL:           12 * time.Minute,
		RetryAttempts: 0,
	})
	if err != nil {
		return err
	}

	// --- Messaging (NATS + Asynq) ---
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	asynqClient := taskqueue.NewClient(redisAddr, cfg.Redis.Password)
	asynqServer := taskqueue.NewServer(redisAddr, cfg.Redis.Password, cfg.Asynq.Concurrency)

	var natsClient *pkgnats.Client
	var eventPublisher natsgateway.EventPublisher

	if cfg.NATS.Enabled {
		natsClient, err = pkgnats.NewClient(cfg.NATS.URL)
		if err != nil {
			return fmt.Errorf("nats connect: %w", err)
		}

		natsCtx := context.Background()
		streamConfigs := []pkgnats.StreamConfig{
			{Name: "RESERVATION_SEARCH", Subjects: []string{"reservation.search.*"}, Retention: jetstream.InterestPolicy, Storage: jetstream.FileStorage, MaxAge: 24 * time.Hour},
			{Name: "RESERVATION_ANALYTICS", Subjects: []string{"reservation.analytics.*"}, Retention: jetstream.LimitsPolicy, Storage: jetstream.FileStorage, MaxAge: 7 * 24 * time.Hour},
			{Name: "PAYMENT_RESERVATION", Subjects: []string{"payment.reservation.*"}, Retention: jetstream.InterestPolicy, Storage: jetstream.FileStorage, MaxAge: 24 * time.Hour},
		}
		if err := pkgnats.CreateStreams(natsCtx, natsClient, streamConfigs); err != nil {
			return fmt.Errorf("nats create streams: %w", err)
		}
		consumerCfg := pkgnats.ConsumerConfig{
			Stream:        constants.StreamPaymentReservation,
			Name:          constants.ConsumerReservationPayment,
			FilterSubject: constants.SubjectPatternPayment,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		}
		if _, err := natsClient.CreateConsumer(natsCtx, consumerCfg.Stream, consumerCfg.ToJetStreamConfig()); err != nil {
			return fmt.Errorf("nats create consumer: %w", err)
		}

		publisher := pkgnats.NewPublisher(natsClient)
		eventPublisher = natsgateway.NewPublisher(publisher)
	}

	// --- Presence Client Adapter ---
	var presenceClient usecase.PresenceClient
	if presenceConn != nil {
		presenceClient = &presenceClientAdapter{
			inner: grpcgw.NewPresenceClient(presencev1.NewPresenceServiceClient(presenceConn)),
		}
	}

	// --- Layers ---
	repo := reservationrepo.NewRepository(tracedPG.GetDB())
	taskEnqueuer := reservationasynq.NewTaskEnqueuer(asynqClient)
	uc := usecase.NewUsecase(
		repo, usecase.NewLockerAdapter(redislockLocker),
		grpcgw.NewBillingClient(billingGRPC),
		grpcgw.NewPaymentClient(paymentGRPC),
		presenceClient,
		taskEnqueuer,
		eventPublisher,
		cfg.Reservation.ExpiryTimeoutMinutes,
		cfg.Reservation.PaymentTimeoutMinutes,
	)
	handler := grpchandler.NewHandler(uc)

	// Register Asynq task handlers.
	expiryHandler := reservationasynq.NewExpiryHandler(&usecaseExpirerAdapter{uc: uc})
	paymentHandler := reservationasynq.NewPaymentTimeoutHandler(&usecaseFailerAdapter{uc: uc})
	asynqServer.Register(reservationasynq.TypeReservationExpire, expiryHandler)
	asynqServer.Register(reservationasynq.TypePaymentHoldTimeout, paymentHandler)

	// --- gRPC Server ---
	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, redisClient)
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

	var natsHandler *natshandler.Handler
	if cfg.NATS.Enabled && natsClient != nil {
		natsHandler = natshandler.NewHandler(uc, natsClient)
		shutdownMgr.Register(func(_ context.Context) error { natsHandler.Stop(); return nil })
		shutdownMgr.Register(func(ctx context.Context) error { natsClient.Close(ctx); return nil })
	}

	// --- Run ---
	go func() {
		if err := asynqServer.Start(); err != nil {
			log.Error("asynq server error", logger.Err(err))
		}
	}()

	if natsHandler != nil {
		if err := natsHandler.Start(); err != nil {
			return err
		}
	}

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

// --- Adapters (moved from bootstrap/adapters.go) ---

type usecaseExpirerAdapter struct {
	uc reservation.Usecase
}

func (a *usecaseExpirerAdapter) ExpireReservation(ctx context.Context, reservationID string) error {
	return a.uc.ExpireReservation(ctx, &model.ExpireReservationRequest{
		ReservationID: reservationID,
	})
}

type usecaseFailerAdapter struct {
	uc reservation.Usecase
}

func (a *usecaseFailerAdapter) FailReservation(ctx context.Context, reservationID string, _ string) error {
	return a.uc.FailReservation(ctx, &model.FailReservationRequest{
		ReservationID: reservationID,
	})
}

type presenceClientAdapter struct {
	inner grpcgw.PresenceClient
}

func (a *presenceClientAdapter) VerifyPresence(ctx context.Context, driverID string, reservationID string, floorNumber int, spotNumber int) (*reservation.PresenceResult, error) {
	result, err := a.inner.VerifyPresence(ctx, driverID, reservationID, floorNumber, spotNumber)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
