// Package main is the entry point for the ParkirPintar Presence Service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
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

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// staticLookup is a minimal ReservationLookup that returns a fixed spot
// for any reservation. In production, this would call the reservation service.
type staticLookup struct{}

func (s *staticLookup) GetSpotForReservation(_ context.Context, reservationID string) (*presenceuc.SpotInfo, error) {
	// In a real implementation, this would query the reservation service
	// to get the assigned spot for the given reservation.
	return &presenceuc.SpotInfo{
		SpotID:   "spot:" + reservationID,
		SpotCode: "A1-01",
	}, nil
}

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

	// Redis client for geo operations.
	redisAddr := getEnv("REDIS_ADDR", fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port))
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Threshold from env or default 50m.
	threshold := 50.0
	if v := os.Getenv("PRESENCE_THRESHOLD_METERS"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = parsed
		}
	}

	interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, log, tracer, nil)

	repo := presencerepo.NewRepository(rdb)
	lookup := &staticLookup{}
	uc := presenceuc.NewUsecase(repo, lookup, threshold)
	handler := presencehandler.NewHandler(uc)

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return rdb.Close() })
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
	if err := grpcSrv.Start(); err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.Server.ShutdownTimeout)
	defer cancel()
	if err := shutdownMgr.Shutdown(ctx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
	}
}
