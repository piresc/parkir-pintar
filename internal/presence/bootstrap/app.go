package bootstrap

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	presencehandler "parkir-pintar/internal/presence/handler"
	presencerepo "parkir-pintar/internal/presence/repository"
	presenceuc "parkir-pintar/internal/presence/usecase"
	"parkir-pintar/pkg/config"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/server"
	presencev1 "parkir-pintar/proto/presence/v1"

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
	cfg, err := config.LoadConfig("presence")
	if err != nil {
		return nil, err
	}

	tel, err := initTelemetry(cfg)
	if err != nil {
		return nil, err
	}
	log := tel.Logger

	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tel.Tracer, nil)

	// Initialize stub sensor gateway (replace with real sensor integration in production).
	sensorGateway := presencerepo.NewStubSensorGateway()
	uc := presenceuc.NewUsecase(sensorGateway)
	handler := presencehandler.NewHandler(uc)

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(ctx context.Context) error { return tel.Metrics.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tel.Tracer.Shutdown(ctx) })
	if tel.Providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return tel.Providers.Shutdown(ctx) })
	}

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
	grpcSrv.RegisterService(&presencev1.PresenceService_ServiceDesc, handler)

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
