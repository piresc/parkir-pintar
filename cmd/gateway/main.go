// Package main is the entry point for the ParkirPintar API Gateway service.
// It follows the bootstrap algorithm: load config → init logger → init telemetry →
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
	"parkir-pintar/pkg/metrics"
	"parkir-pintar/pkg/middleware"
	redispkg "parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/telemetry"
	"parkir-pintar/pkg/tracing"

	paymentv1 "parkir-pintar/proto/payment/v1"
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

	// 2. Initialize unified telemetry (traces, metrics, logs via OTLP)
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)
	telCfg := telemetry.Config{
		ServiceName:     "parkir-pintar-gateway",
		OTLPEndpoint:    otlpEndpoint,
		TraceSampleRate: cfg.Tracing.SampleRate,
	}
	providers, telErr := telemetry.Init(context.Background(), telCfg)
	if telErr != nil {
		slog.Warn("telemetry init failed, continuing with noop", slog.Any("error", telErr))
	}

	// 3. Initialize logger (with OTLP log export if available)
	var log *slog.Logger
	if providers != nil && providers.LoggerProvider != nil {
		log = logger.NewLoggerWithProvider(cfg.Logger, providers.LoggerProvider)
	} else {
		log = logger.NewLogger(cfg.Logger)
	}

	// 4. Initialize OTEL tracer (uses the existing tracing package interface)
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

	// 5. Initialize metrics (OTLP push)
	met, err := metrics.NewMetrics("parkir-pintar-gateway", otlpEndpoint)
	if err != nil {
		log.Warn("metrics init failed, continuing without metrics", slog.Any("error", err))
	}

	// 6. Initialize gRPC client connections to downstream services
	ctx := context.Background()

	reservationTarget := getEnv("GRPC_RESERVATION_TARGET", "localhost:9091")
	searchTarget := getEnv("GRPC_SEARCH_TARGET", "localhost:9092")
	paymentTarget := getEnv("GRPC_PAYMENT_TARGET", "localhost:9094")

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

	// 7. Create gRPC service clients
	reservationClient := reservationv1.NewReservationServiceClient(reservationConn)
	searchClient := searchv1.NewSearchServiceClient(searchConn)
	paymentClient := paymentv1.NewPaymentServiceClient(paymentConn)

	// 8. Setup shutdown manager
	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { return reservationConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return searchConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return paymentConn.Close() })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })
	if met != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return met.Shutdown(ctx) })
	}
	if providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return providers.Shutdown(ctx) })
	}

	// 9. Setup Gin engine with middleware chain
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	mw := middleware.NewMiddleware(cfg, log, tracer)

	engine.Use(mw.RecoveryHandler())
	engine.Use(mw.SecurityHeaders())
	if met != nil {
		engine.Use(met.HTTPMiddleware())
	}
	engine.Use(mw.CorsHandler(cfg.Server.AllowedOrigins))
	engine.Use(mw.RateLimiter(middleware.DefaultRateLimitConfig()))
	engine.Use(mw.TracingHandler())

	// Register pprof debug endpoints (only if ENABLE_PPROF=true)
	gatewayhandler.RegisterPprof(engine)

	// 10. Serve Swagger UI
	engine.StaticFile("/swagger/doc.yaml", "./docs/swagger.yaml")
	engine.StaticFile("/swagger", "./docs/swagger-ui/index.html")
	engine.StaticFile("/swagger/", "./docs/swagger-ui/index.html")

	// 11. Register health endpoints (before auth middleware)
	healthSvc := health.NewService(log)

	// Init direct connections for health checks
	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err == nil {
		healthSvc.AddChecker("postgres", health.NewPostgresChecker(pgClient))
		shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	}

	redisClient, err := redispkg.NewClient(cfg.Redis)
	if err == nil {
		healthSvc.AddChecker("redis", health.NewRedisChecker(redisClient))
		shutdownMgr.Register(func(_ context.Context) error { _ = redisClient.GetClient().Close(); return nil })
	}

	health.RegisterRoutes(engine, cfg.App.Name, cfg.App.Version, healthSvc)

	// 12. Register gateway REST routes (with JWT auth)
	jwtSecret := cfg.JWT.Secret
	gwHandler := gatewayhandler.NewHandler(reservationClient, searchClient, paymentClient)
	gwHandler.RegisterRoutes(engine, mw, jwtSecret)

	// 13. Start server
	srv := server.NewGracefulServer(engine, log, cfg.Server.Port, time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	if err := srv.Start(); err != nil {
		log.Error("server error", slog.Any("error", err))
	}

	// 14. Run shutdown manager
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
