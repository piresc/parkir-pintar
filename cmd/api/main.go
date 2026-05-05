// Package main is the application entry point for the parkir-pintar API.
// It follows the bootstrap algorithm: load config → init logger → init tracer →
// init infrastructure → create traced wrappers → setup shutdown manager →
// setup Gin with middleware → register health + domain routes → start server →
// run shutdown manager.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"parkir-pintar/internal/example"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/database"
	"parkir-pintar/pkg/health"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/redis"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
)

func main() {
	// 1. Load configuration (fail fast on missing required vars)
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

	// 4. Initialize infrastructure clients
	pgClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		log.Error("failed to connect to postgres", slog.Any("error", err))
		os.Exit(1)
	}

	redisClient, err := redis.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Error("failed to connect to redis", slog.Any("error", err))
		os.Exit(1)
	}

	natsClient, err := nats.NewClient(cfg.NATS.URL)
	if err != nil {
		log.Error("failed to connect to nats", slog.Any("error", err))
		os.Exit(1)
	}

	// 5. Create traced wrappers
	tracedPG := database.NewTracedPostgresClient(pgClient, tracer)
	_ = redis.NewTracedRedisClient(redisClient, tracer)
	_ = nats.NewTracedClient(natsClient, tracer)

	// 6. Setup shutdown manager
	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { natsClient.Close(); return nil })
	shutdownMgr.Register(func(_ context.Context) error { return pgClient.Close() })
	shutdownMgr.Register(func(_ context.Context) error { return redisClient.Close() })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })

	// 7. Setup Gin engine with unified middleware chain
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	mw := middleware.NewMiddleware(cfg, log, tracer)

	engine.Use(mw.RecoveryHandler())
	engine.Use(mw.CorsHandler(cfg.Server.AllowedOrigins))
	engine.Use(mw.RateLimiter(middleware.DefaultRateLimitConfig()))
	engine.Use(mw.NormalizeMsisdn())
	engine.Use(mw.GenerateTransactionID())
	engine.Use(mw.SetContextValues())
	engine.Use(mw.TracingHandler())
	engine.Use(mw.LogResponse())

	// 8. Register health endpoints (before auth middleware)
	healthSvc := health.NewService(log)
	healthSvc.AddChecker("postgres", health.NewPostgresChecker(pgClient))
	healthSvc.AddChecker("redis", health.NewRedisChecker(redisClient))
	healthSvc.AddChecker("nats", health.NewNATSChecker(natsClient))
	health.RegisterRoutes(engine, cfg.App.Name, cfg.App.Version, healthSvc)

	// 9. Register domain routes
	example.RegisterRoutes(engine, mw, tracedPG)

	// 10. Start server (blocks until shutdown signal)
	srv := server.NewGracefulServer(engine, log, cfg.Server.Port)
	if err := srv.Start(); err != nil {
		log.Error("server error", slog.Any("error", err))
	}

	// 11. Run shutdown manager
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := shutdownMgr.Shutdown(ctx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
	}
}
