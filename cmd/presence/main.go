// Package main is the entry point for the ParkirPintar Presence Service.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	presencehandler "parkir-pintar/internal/presence/handler"
	presencerepo "parkir-pintar/internal/presence/repository"
	presenceuc "parkir-pintar/internal/presence/usecase"
	"parkir-pintar/pkg/config"
	grpcmiddleware "parkir-pintar/pkg/grpcmiddleware"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
	presencev1 "parkir-pintar/proto/presence/v1"

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
		Enabled: cfg.Tracing.Enabled, ServiceName: "parkir-pintar-presence",
		SampleRate: cfg.Tracing.SampleRate, Exporter: cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, nil)

	// Initialize stub sensor gateway (replace with real sensor integration in production).
	sensorGateway := presencerepo.NewStubSensorGateway()
	uc := presenceuc.NewUsecase(sensorGateway)
	handler := presencehandler.NewHandler(uc)

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })

	grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, cfg.GRPC.Server.RequestTimeout,
		grpc.ChainUnaryInterceptor(
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
