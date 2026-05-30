package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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

	analyticsv1 "parkir-pintar/proto/analytics/v1"
	billingv1 "parkir-pintar/proto/billing/v1"
	paymentv1 "parkir-pintar/proto/payment/v1"
	presencev1 "parkir-pintar/proto/presence/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"
	searchv1 "parkir-pintar/proto/search/v1"

	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", logger.Err(err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig("gateway")
	if err != nil {
		return err
	}

	// --- Telemetry ---
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)

	providers, telErr := telemetry.Init(context.Background(), telemetry.Config{
		ServiceName:     "parkir-pintar-gateway",
		OTLPEndpoint:    otlpEndpoint,
		TraceSampleRate: cfg.Tracing.SampleRate,
	})
	if telErr != nil {
		slog.Warn("telemetry init failed, continuing with noop", logger.Err(telErr))
	}

	var log *slog.Logger
	if providers != nil && providers.LoggerProvider != nil {
		log = logger.NewLoggerWithProvider(cfg.Logger, providers.LoggerProvider)
	} else {
		log = logger.NewLogger(cfg.Logger)
	}

	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled:      cfg.Tracing.Enabled,
		ServiceName:  cfg.Tracing.ServiceName,
		SampleRate:   cfg.Tracing.SampleRate,
		ExcludePaths: cfg.Tracing.ExcludePaths,
		Exporter:     cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
	})
	if err != nil {
		log.Warn("tracer init failed, falling back to noop", logger.Err(err))
		tracer = tracing.NewNoOpTracer()
	}

	metricsInst, err := metrics.NewMetrics("parkir-pintar-gateway", otlpEndpoint)
	if err != nil {
		log.Warn("metrics init failed, continuing without metrics", logger.Err(err))
	}

	// --- gRPC Clients ---
	reservationTarget := getEnv("GRPC_RESERVATION_TARGET", "localhost:9091")
	searchTarget := getEnv("GRPC_SEARCH_TARGET", "localhost:9092")
	billingTarget := getEnv("GRPC_BILLING_TARGET", "localhost:9093")
	paymentTarget := getEnv("GRPC_PAYMENT_TARGET", "localhost:9094")
	analyticsTarget := getEnv("GRPC_ANALYTICS_TARGET", "localhost:9095")
	presenceTarget := getEnv("GRPC_PRESENCE_TARGET", "localhost:9096")

	clientCfg := grpcclient.ClientConfig{
		DialTimeout:      cfg.GRPC.Client.DialTimeout,
		KeepAliveTime:    cfg.GRPC.Client.KeepAliveTime,
		KeepAliveTimeout: cfg.GRPC.Client.KeepAliveTimeout,
		Tracer:           tracer,
		Logger:           log,
	}

	ctx := context.Background()

	clientCfg.Target = reservationTarget
	reservationConn, err := grpcclient.Dial(ctx, clientCfg)
	if err != nil {
		return err
	}

	clientCfg.Target = searchTarget
	searchConn, err := grpcclient.Dial(ctx, clientCfg)
	if err != nil {
		_ = reservationConn.Close()
		return err
	}

	clientCfg.Target = paymentTarget
	paymentConn, err := grpcclient.Dial(ctx, clientCfg)
	if err != nil {
		_ = reservationConn.Close()
		_ = searchConn.Close()
		return err
	}

	clientCfg.Target = billingTarget
	billingConn, err := grpcclient.Dial(ctx, clientCfg)
	if err != nil {
		_ = reservationConn.Close()
		_ = searchConn.Close()
		_ = paymentConn.Close()
		return err
	}

	reservationGRPC := reservationv1.NewReservationServiceClient(reservationConn)
	searchGRPC := searchv1.NewSearchServiceClient(searchConn)
	paymentGRPC := paymentv1.NewPaymentServiceClient(paymentConn)
	billingGRPC := billingv1.NewBillingServiceClient(billingConn)

	// Analytics is optional — graceful degradation if unavailable.
	var analyticsConn *grpc.ClientConn
	var analyticsGRPC analyticsv1.AnalyticsServiceClient
	clientCfg.Target = analyticsTarget
	analyticsConn, analyticsErr := grpcclient.Dial(ctx, clientCfg)
	if analyticsErr != nil {
		log.Warn("analytics service unavailable, continuing without analytics", logger.Err(analyticsErr))
	} else {
		analyticsGRPC = analyticsv1.NewAnalyticsServiceClient(analyticsConn)
	}

	// Presence is optional — graceful degradation if unavailable.
	var presenceConn *grpc.ClientConn
	var presenceGRPC presencev1.PresenceServiceClient
	clientCfg.Target = presenceTarget
	presenceConn, presenceErr := grpcclient.Dial(ctx, clientCfg)
	if presenceErr != nil {
		log.Warn("presence service unavailable, continuing without presence", logger.Err(presenceErr))
	} else {
		presenceGRPC = presencev1.NewPresenceServiceClient(presenceConn)
	}

	// --- Shutdown Manager ---
	shutdownMgr := server.NewShutdownManager(log)

	shutdownMgr.Register(func(_ context.Context) error { return reservationConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return searchConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return paymentConn.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return billingConn.Close() })
	if analyticsConn != nil {
		shutdownMgr.Register(func(_ context.Context) error { return analyticsConn.Close() })
	}
	if presenceConn != nil {
		shutdownMgr.Register(func(_ context.Context) error { return presenceConn.Close() })
	}
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })
	if metricsInst != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return metricsInst.Shutdown(ctx) })
	}
	if providers != nil {
		shutdownMgr.Register(func(ctx context.Context) error { return providers.Shutdown(ctx) })
	}

	// --- HTTP Server ---
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	mw := middleware.NewMiddleware(cfg, log, tracer)

	engine.Use(mw.RecoveryHandler())
	engine.Use(mw.SecurityHeaders())
	if metricsInst != nil {
		engine.Use(metricsInst.HTTPMiddleware())
	}
	engine.Use(mw.CorsHandler(cfg.Server.AllowedOrigins))
	engine.Use(mw.RateLimiter(middleware.DefaultRateLimitConfig()))
	engine.Use(mw.TracingHandler())

	// Register pprof debug endpoints (only if ENABLE_PPROF=true).
	gatewayhandler.RegisterPprof(engine, mw, cfg.JWT.Secret)

	// Swagger UI: open docs/api/swagger-ui/index.html directly in browser (no server needed).

	// Register health endpoints (before auth middleware).
	healthSvc := health.NewService(log)

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

	// Register gateway REST routes (with JWT auth).
	jwtSecret := cfg.JWT.Secret
	gwHandler := gatewayhandler.NewHandler(reservationGRPC, searchGRPC, paymentGRPC, presenceGRPC)
	gwHandler.RegisterRoutes(engine, mw, jwtSecret)

	// Register analytics REST routes (gRPC-backed, with JWT auth).
	analyticsHandler := gatewayhandler.NewAnalyticsHandler(analyticsGRPC)
	analyticsHandler.RegisterRoutes(engine, mw, jwtSecret)

	// Register billing REST routes (gRPC-backed, with JWT auth).
	billingHandler := gatewayhandler.NewBillingHandler(billingGRPC, reservationGRPC)
	billingHandler.RegisterRoutes(engine, mw, jwtSecret)

	srv := server.NewGracefulServer(engine, log, cfg.Server.Port, time.Duration(cfg.Server.ShutdownTimeout)*time.Second)

	// --- Run ---
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	select {
	case <-sigCtx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			log.Error("server error", logger.Err(err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.Server.ShutdownTimeout)
	defer cancel()
	if err := shutdownMgr.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", logger.Err(err))
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
