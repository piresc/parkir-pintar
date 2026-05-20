package bootstrap

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	grpcgw "parkir-pintar/internal/reservation/gateway/grpc"
	reservationhandler "parkir-pintar/internal/reservation/handler"
	reservationrepo "parkir-pintar/internal/reservation/repository"
	"parkir-pintar/internal/reservation/usecase"
	taskqueue "parkir-pintar/pkg/asynq"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/redislock"
	"parkir-pintar/pkg/server"
	presencev1 "parkir-pintar/proto/presence/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"

	"google.golang.org/grpc"
)

type App struct {
	cfg         *config.Config
	log         *slog.Logger
	tel         *Telemetry
	messaging   *Messaging
	grpcSrv     *grpcserver.GRPCServer
	natsHandler *reservationhandler.NATSHandler
	shutdownMgr *server.ShutdownManager
}

func New() (*App, error) {
	cfg, err := config.Load("config/.env")
	if err != nil {
		return nil, err
	}

	tel, err := initTelemetry(cfg)
	if err != nil {
		return nil, err
	}
	log := tel.Logger

	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		return nil, err
	}
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return nil, err
	}

	tracedPG := database.NewTracedPostgresClient(pgClient, tel.Tracer)

	clients, err := initClients(cfg, tel.Tracer, log)
	if err != nil {
		return nil, err
	}

	redislockLocker, err := redislock.NewLocker(redisClient, redislock.Config{
		TTL:           12 * time.Minute,
		RetryAttempts: 0,
	})
	if err != nil {
		return nil, err
	}

	messaging, err := initMessaging(cfg)
	if err != nil {
		return nil, err
	}

	var presenceClient usecase.PresenceClient
	if clients.PresenceConn != nil {
		presenceClient = &presenceClientAdapter{
			inner: grpcgw.NewPresenceClient(presencev1.NewPresenceServiceClient(clients.PresenceConn)),
		}
	}

	repo := reservationrepo.NewRepository(tracedPG.GetDB())
	uc := usecase.NewUsecase(
		repo, usecase.NewLockerAdapter(redislockLocker),
		grpcgw.NewBillingClient(clients.BillingGRPC),
		grpcgw.NewPaymentClient(clients.PaymentGRPC),
		presenceClient,
		messaging.AsynqClient,
		messaging.EventPublisher,
		cfg.Reservation.ExpiryTimeoutMinutes,
		cfg.Reservation.PaymentTimeoutMinutes,
	)
	handler := reservationhandler.NewHandler(uc)

	expiryHandler := taskqueue.NewReservationExpiryHandler(&usecaseExpirerAdapter{uc: uc})
	paymentHandler := taskqueue.NewPaymentHoldTimeoutHandler(&usecaseFailerAdapter{uc: uc})
	messaging.AsynqServer.RegisterHandlers(expiryHandler, paymentHandler)

	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tel.Tracer, redisClient)
	grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, cfg.GRPC.Server.RequestTimeout,
		grpc.ChainUnaryInterceptor(
			tel.Metrics.GRPCUnaryInterceptor(),
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

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { messaging.AsynqServer.Shutdown(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { return messaging.AsynqClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return clients.BillingConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return clients.PaymentConn.Close() })
	shutdownMgr.Register(func(ctx context.Context) error { return tel.Metrics.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tel.Tracer.Shutdown(ctx) })
	if tel.Providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return tel.Providers.Shutdown(ctx) })
	}

	var natsHandler *reservationhandler.NATSHandler
	if cfg.NATS.Enabled && messaging.NATSClient != nil {
		natsHandler = reservationhandler.NewNATSHandler(uc, messaging.NATSClient)
		shutdownMgr.Register(func(_ context.Context) error { natsHandler.Stop(); return nil })
		shutdownMgr.Register(func(ctx context.Context) error { messaging.NATSClient.Close(ctx); return nil })
	}

	return &App{
		cfg:         cfg,
		log:         log,
		tel:         tel,
		messaging:   messaging,
		grpcSrv:     grpcSrv,
		natsHandler: natsHandler,
		shutdownMgr: shutdownMgr,
	}, nil
}

func (a *App) Run() error {
	go func() {
		if err := a.messaging.AsynqServer.Start(); err != nil {
			a.log.Error("asynq server error", slog.Any("error", err))
		}
	}()

	if a.natsHandler != nil {
		if err := a.natsHandler.Start(); err != nil {
			return err
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- a.grpcSrv.Start()
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			a.log.Error("gRPC server error", slog.Any("error", err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.GRPC.Server.ShutdownTimeout)
	defer cancel()
	if err := a.shutdownMgr.Shutdown(shutdownCtx); err != nil {
		a.log.Error("shutdown error", slog.Any("error", err))
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
