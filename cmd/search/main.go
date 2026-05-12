// Package main is the entry point for the ParkirPintar Search Service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/natssetup"
	searchhandler "parkir-pintar/internal/search/handler"
	searchrepo "parkir-pintar/internal/search/repository"
	searchsub "parkir-pintar/internal/search/subscriber"
	searchsync "parkir-pintar/internal/search/sync"
	searchuc "parkir-pintar/internal/search/usecase"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
	searchv1 "parkir-pintar/proto/search/v1"

	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load("config/.env")
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.NewLogger(cfg.Logger)
	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled: cfg.Tracing.Enabled, ServiceName: "parkir-pintar-search",
		SampleRate: cfg.Tracing.SampleRate, Exporter: cfg.Tracing.Exporter,
	})
	if err != nil {
		log.Warn("tracer init failed", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		log.Error("postgres connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	redisClient, err := redis.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Error("redis connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	natsClient, err := nats.NewClient(cfg.NATS.URL)
	if err != nil {
		log.Error("nats connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	if err := natssetup.SetupStreams(natsClient); err != nil {
		log.Error("nats stream setup failed", slog.Any("error", err))
		os.Exit(1)
	}

	tracedPG := database.NewTracedPostgresClient(pgClient, tracer)
	tracedRedis := redis.NewTracedRedisClient(redisClient, tracer)
	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, redisClient)

	repo := searchrepo.NewRepository(tracedPG.GetDB())
	readModelRepo := searchrepo.NewReadModelRepository(tracedPG.GetDB())
	spotSyncer := searchsync.NewSpotSync(readModelRepo)
	uc := searchuc.NewUsecase(repo, tracedRedis)
	handler := searchhandler.NewHandler(uc)

	// Wire NATS subscriber for cache invalidation
	sub := searchsub.NewNATSSubscriber(tracedRedis)
	if err := natsClient.CreateConsumer(nats.ConsumerConfig{
		StreamName:    "RESERVATIONS",
		ConsumerName:  "search-cache-invalidation",
		FilterSubject: "reservation.*",
		DeliverPolicy: jetstream.DeliverLastPolicy,
	}); err != nil {
		log.Error("failed to create NATS consumer", slog.Any("error", err))
	}
	go func() {
		if err := natsClient.ConsumeMessages("RESERVATIONS", "search-cache-invalidation", func(msg jetstream.Msg) error {
			sub.HandleReservationEvent(context.Background(), msg.Subject(), msg.Data())
			return nil
		}); err != nil {
			log.Error("NATS consume error", slog.Any("error", err))
		}
	}()

	// Wire NATS subscriber for spot read model sync
	if err := natsClient.CreateConsumer(nats.ConsumerConfig{
		StreamName:    "RESERVATIONS",
		ConsumerName:  "search-spot-sync",
		FilterSubject: "spot.updated",
		DeliverPolicy: jetstream.DeliverLastPolicy,
	}); err != nil {
		log.Error("failed to create spot sync consumer", slog.Any("error", err))
	}
	go func() {
		if err := natsClient.ConsumeMessages("RESERVATIONS", "search-spot-sync", func(msg jetstream.Msg) error {
			spotSyncer.HandleNATSEvent(context.Background(), msg.Subject(), msg.Data())
			return nil
		}); err != nil {
			log.Error("spot sync consume error", slog.Any("error", err))
		}
	}()

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { natsClient.Close(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })

	grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, 30*time.Second,
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryUnaryInterceptor(),
			interceptors.AuthUnaryInterceptor(nil),
			interceptors.LoggingUnaryInterceptor(),
			interceptors.TracingUnaryInterceptor(),
			interceptors.RateLimitUnaryInterceptor(grpcmiddleware.RateLimitConfig{
				RequestsPerSecond: 100,
				BurstSize:        200,
				CleanupInterval:  5 * time.Minute,
			}),
		),
	)
	grpcSrv.RegisterService(&searchv1.SearchService_ServiceDesc, handler)
	if err := grpcSrv.Start(); err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := shutdownMgr.Shutdown(ctx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
	}
}
