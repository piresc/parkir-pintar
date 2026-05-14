// Package main is the entry point for the ParkirPintar Search Service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	searchhandler "parkir-pintar/internal/search/handler"
	searchrepo "parkir-pintar/internal/search/repository"
	searchuc "parkir-pintar/internal/search/usecase"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/metrics"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
	searchv1 "parkir-pintar/proto/search/v1"

	"google.golang.org/grpc"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

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
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)
	metricsInst, err := metrics.NewMetrics("parkir-pintar-search", otlpEndpoint)
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
	tracedRedis := redis.NewTracedRedisClient(redisClient, tracer)
	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, redisClient)

	repo := searchrepo.NewRepository(tracedPG.GetDB())
	uc := searchuc.NewUsecase(repo, tracedRedis)
	handler := searchhandler.NewHandler(uc)

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })

	shutdownMgr.Register(func(ctx context.Context) error { return metricsInst.Shutdown(ctx) })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })

	grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, 30*time.Second,
		grpc.ChainUnaryInterceptor(
			metricsInst.GRPCUnaryInterceptor(),
			interceptors.RecoveryUnaryInterceptor(),
			interceptors.AuthUnaryInterceptor(nil),
			interceptors.LoggingUnaryInterceptor(),
			interceptors.TracingUnaryInterceptor(),
			interceptors.RateLimitUnaryInterceptor(grpcmiddleware.RateLimitConfig{
				RequestsPerSecond: 100,
				BurstSize:         200,
				CleanupInterval:   5 * time.Minute,
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
