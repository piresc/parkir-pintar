package bootstrap

import (
	"context"
	"log/slog"
	"parkir-pintar/pkg/logger"
	"os/signal"
	"syscall"
	"time"

	billinghandler "parkir-pintar/internal/billing/handler"
	billingrepo "parkir-pintar/internal/billing/repository"
	billinguc "parkir-pintar/internal/billing/usecase"
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
	grpcSrv     *grpcserver.GRPCServer
	shutdownMgr *server.ShutdownManager
}

func New() (*App, error) {
	cfg, err := config.LoadConfig("billing")
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

	repo := billingrepo.NewRepository(tracedPG.GetDB())
	uc := billinguc.NewUsecase(repo)
	handler := billinghandler.NewHandler(uc)

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
	shutdownMgr.Register(func(ctx context.Context) error { return tel.Metrics.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tel.Tracer.Shutdown(ctx) })
	if tel.Providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return tel.Providers.Shutdown(ctx) })
	}

	return &App{
		cfg:         cfg,
		log:         log,
		tel:         tel,
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
			a.log.Error("gRPC server error", logger.Err(err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.GRPC.Server.ShutdownTimeout)
	defer cancel()
	if err := a.shutdownMgr.Shutdown(shutdownCtx); err != nil {
		a.log.Error("shutdown error", logger.Err(err))
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
