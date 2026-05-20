package bootstrap

import (
	"context"
	"log/slog"
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
	"parkir-pintar/pkg/server"

	"google.golang.org/grpc"
)

type App struct {
	cfg         *config.Config
	log         *slog.Logger
	tel         *Telemetry
	messaging   *Messaging
	grpcSrv     *grpcserver.GRPCServer
	shutdownMgr *server.ShutdownManager
}

func New() (*App, error) {
	cfg, err := config.LoadConfig("payment")
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

	tracedPG := database.NewTracedPostgresClient(pgClient, tel.Tracer)

	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tel.Tracer, nil)

	messaging, err := initMessaging(cfg)
	if err != nil {
		return nil, err
	}

	repo := paymentrepo.NewRepository(tracedPG.GetDB())
	gw := paymentgateway.NewStubGateway(false)
	uc := paymentuc.NewUsecase(repo, gw, messaging.Publisher)
	handler := paymenthandler.NewHandler(uc)

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
	handler.RegisterService(grpcSrv.Server())

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	if messaging.NATSClient != nil {
		shutdownMgr.Register(func(ctx context.Context) error { messaging.NATSClient.Close(ctx); return nil })
	}
	shutdownMgr.Register(func(ctx context.Context) error { return tel.Metrics.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tel.Tracer.Shutdown(ctx) })
	if tel.Providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return tel.Providers.Shutdown(ctx) })
	}

	return &App{
		cfg:         cfg,
		log:         log,
		tel:         tel,
		messaging:   messaging,
		grpcSrv:     grpcSrv,
		shutdownMgr: shutdownMgr,
	}, nil
}

func (a *App) Run() error {
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
