// Package main is the entry point for the ParkirPintar API Gateway service.
// It follows the bootstrap algorithm: load config → init logger → init tracer →
// init gRPC client connections → setup Gin with middleware → register health +
// REST routes → start server → run shutdown manager.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	gatewayhandler "parkir-pintar/internal/gateway/handler"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	"parkir-pintar/pkg/grpcclient"
	"parkir-pintar/pkg/health"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/middleware"
	natspkg "parkir-pintar/pkg/nats"
	redispkg "parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"

	paymentv1 "parkir-pintar/proto/payment/v1"
	presencev1 "parkir-pintar/proto/presence/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"
	searchv1 "parkir-pintar/proto/search/v1"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load("config/.env")
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	// 2. Initialize logger
	log := logger.NewLogger(cfg.Logger)

	// 3. Initialize OTEL tracer
	tracerCfg := &tracing.Config{
		Enabled:      cfg.Tracing.Enabled,
		ServiceName:  cfg.Tracing.ServiceName,
		SampleRate:   cfg.Tracing.SampleRate,
		ExcludePaths: cfg.Tracing.ExcludePaths,
		Exporter:     cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
		NewRelic: tracing.NewRelicExporterConfig{
			LicenseKey: cfg.Tracing.NewRelic.LicenseKey,
			Enabled:    cfg.Tracing.NewRelic.Enabled,
		},
	}
	tracer, err := tracing.NewTracer(tracerCfg)
	if err != nil {
		log.Warn("tracer init failed, falling back to noop", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	// 4. Initialize gRPC client connections to downstream services
	ctx := context.Background()

	reservationTarget := getEnv("GRPC_RESERVATION_TARGET", "localhost:9091")
	searchTarget := getEnv("GRPC_SEARCH_TARGET", "localhost:9092")
	paymentTarget := getEnv("GRPC_PAYMENT_TARGET", "localhost:9094")
	presenceTarget := getEnv("GRPC_PRESENCE_TARGET", "localhost:9095")

	clientCfg := grpcclient.ClientConfig{
		DialTimeout:      cfg.GRPC.Client.DialTimeout,
		KeepAliveTime:    cfg.GRPC.Client.KeepAliveTime,
		KeepAliveTimeout: cfg.GRPC.Client.KeepAliveTimeout,
		Tracer:           tracer,
		Logger:           log,
	}

	clientCfg.Target = reservationTarget
	reservationConn, err := grpcclient.Dial(ctx, clientCfg)
	if err != nil {
		log.Error("failed to connect to reservation service", slog.Any("error", err))
		os.Exit(1)
	}

	clientCfg.Target = searchTarget
	searchConn, err := grpcclient.Dial(ctx, clientCfg)
	if err != nil {
		log.Error("failed to connect to search service", slog.Any("error", err))
		os.Exit(1)
	}

	clientCfg.Target = paymentTarget
	paymentConn, err := grpcclient.Dial(ctx, clientCfg)
	if err != nil {
		log.Error("failed to connect to payment service", slog.Any("error", err))
		os.Exit(1)
	}

	clientCfg.Target = presenceTarget
	presenceConn, err := grpcclient.Dial(ctx, clientCfg)
	if err != nil {
		log.Error("failed to connect to presence service", slog.Any("error", err))
		os.Exit(1)
	}

	// 5. Create gRPC service clients
	reservationClient := reservationv1.NewReservationServiceClient(reservationConn)
	searchClient := searchv1.NewSearchServiceClient(searchConn)
	paymentClient := paymentv1.NewPaymentServiceClient(paymentConn)
	presenceClient := presencev1.NewPresenceServiceClient(presenceConn)

	// 6. Setup shutdown manager
	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return reservationConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return searchConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return paymentConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return presenceConn.Close() })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })

	// 7. Setup Gin engine with middleware chain
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	mw := middleware.NewMiddleware(cfg, log, tracer)

	engine.Use(mw.RecoveryHandler())
	engine.Use(mw.CorsHandler(cfg.Server.AllowedOrigins))
	engine.Use(mw.RateLimiter(middleware.DefaultRateLimitConfig()))
	engine.Use(mw.TracingHandler())

	// 8. Register health endpoints (before auth middleware)
	healthSvc := health.NewService(log)

	// Init direct connections for health checks
	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err == nil {
		healthSvc.AddChecker("postgres", health.NewPostgresChecker(pgClient))
		shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	}

	redisClient, err := redispkg.NewRedisClient(cfg.Redis)
	if err == nil {
		healthSvc.AddChecker("redis", health.NewRedisChecker(redisClient))
		shutdownMgr.Register(func(_ context.Context) error { _ = redisClient.GetClient().Close(); return nil })
	}

	natsClient, err := natspkg.NewClient(cfg.NATS.URL)
	if err == nil {
		healthSvc.AddChecker("nats", health.NewNATSChecker(natsClient))
		shutdownMgr.Register(func(_ context.Context) error { natsClient.Close(); return nil })
	}

	health.RegisterRoutes(engine, cfg.App.Name, cfg.App.Version, healthSvc)

	// 9. Register gateway REST routes (with JWT auth)
	jwtSecret := cfg.JWT.Secret
	gwHandler := gatewayhandler.NewHandler(reservationClient, searchClient, paymentClient, presenceClient)
	gwHandler.RegisterRoutes(engine, mw, jwtSecret)

	// 10. Start server
	srv := server.NewGracefulServer(engine, log, cfg.Server.Port, time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	if err := srv.Start(); err != nil {
		log.Error("server error", slog.Any("error", err))
	}

	// 11. Run shutdown manager
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := shutdownMgr.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
	}
}

// getEnv reads an environment variable with a fallback default.
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
