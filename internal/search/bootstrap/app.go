package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	searchhandler "parkir-pintar/internal/search/handler"
	searchrepo "parkir-pintar/internal/search/repository"
	searchsync "parkir-pintar/internal/search/sync"
	searchuc "parkir-pintar/internal/search/usecase"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	pkgnats "parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	searchv1 "parkir-pintar/proto/search/v1"

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
	cfg, err := config.LoadConfig("search")
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
	tracedRedis := redis.NewTracedRedisClient(redisClient, tel.Tracer)
	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tel.Tracer, redisClient)

	repo := searchrepo.NewRepository(tracedPG.GetDB())
	uc := searchuc.NewUsecase(repo, tracedRedis)
	handler := searchhandler.NewHandler(uc)

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })

	// NATS JetStream consumer (spot updates from reservation service)
	if cfg.NATS.Enabled {
		natsClient, natsErr := pkgnats.NewClient(cfg.NATS.URL)
		if natsErr != nil {
			return nil, fmt.Errorf("nats connect: %w", natsErr)
		}

		natsCtx := context.Background()
		if err := pkgnats.CreateStreams(natsCtx, natsClient); err != nil {
			return nil, fmt.Errorf("nats create streams: %w", err)
		}
		if err := pkgnats.CreateConsumersForService(natsCtx, natsClient, "search"); err != nil {
			return nil, fmt.Errorf("nats create consumers: %w", err)
		}

		readModelRepo := searchrepo.NewReadModelRepository(tracedPG.GetDB())
		spotSync := searchsync.NewSpotSync(readModelRepo)
		natsHandler := searchhandler.NewNATSHandler(spotSync, tracedRedis, natsClient, 0)
		cc, err := natsHandler.InitConsumers()
		if err != nil {
			return nil, fmt.Errorf("nats consumer init: %w", err)
		}

		shutdownMgr.Register(func(_ context.Context) error { cc.Stop(); return nil })
		shutdownMgr.Register(func(ctx context.Context) error { natsClient.Close(ctx); return nil })
	}

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
	grpcSrv.RegisterService(&searchv1.SearchService_ServiceDesc, handler)

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
