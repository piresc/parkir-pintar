// Package server provides graceful HTTP server startup/shutdown and
// ordered component lifecycle management.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// GracefulServer wraps a Gin engine with signal-based graceful shutdown.
type GracefulServer struct {
	engine          *gin.Engine
	logger          *slog.Logger
	port            int
	shutdownTimeout time.Duration
}

// NewGracefulServer creates a new GracefulServer.
func NewGracefulServer(engine *gin.Engine, logger *slog.Logger, port int, shutdownTimeout time.Duration) *GracefulServer {
	return &GracefulServer{
		engine:          engine,
		logger:          logger,
		port:            port,
		shutdownTimeout: shutdownTimeout,
	}
}

// Start starts the HTTP server and blocks until SIGINT or SIGTERM is received.
// On signal, it initiates a graceful shutdown with a 30-second timeout.
// Uses slog.Error + os.Exit(1) for fatal errors (NOT srv.ErrorLog.Fatal()).
func (s *GracefulServer) Start() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Channel to capture server errors
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting HTTP server", slog.Int("port", s.port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		// Use slog.Error + os.Exit(1) instead of srv.ErrorLog.Fatal()
		// to avoid nil pointer crash (fix from boilerplate-golang)
		slog.Error("server failed to start", slog.String("error", err.Error()))
		os.Exit(1)
	case sig := <-quit:
		s.logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	}

	// Graceful shutdown with 30s timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	s.logger.Info("shutting down HTTP server")
	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("server shutdown error", slog.String("error", err.Error()))
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("HTTP server stopped gracefully")
	return nil
}

// ShutdownManager manages ordered shutdown of application components.
// Registered functions are called in registration order.
// Individual failures do not block other cleanups from executing.
type ShutdownManager struct {
	logger    *slog.Logger
	functions []func(context.Context) error
}

// NewShutdownManager creates a new ShutdownManager.
func NewShutdownManager(logger *slog.Logger) *ShutdownManager {
	return &ShutdownManager{
		logger:    logger,
		functions: make([]func(context.Context) error, 0),
	}
}

// Register adds a cleanup function to be called during shutdown.
// Functions are called in the order they are registered.
func (sm *ShutdownManager) Register(fn func(context.Context) error) {
	sm.functions = append(sm.functions, fn)
}

// Shutdown executes all registered cleanup functions in order.
// Individual failures are logged but do not prevent other cleanups from running.
func (sm *ShutdownManager) Shutdown(ctx context.Context) error {
	sm.logger.Info("starting ordered shutdown", slog.Int("components", len(sm.functions)))

	var firstErr error
	for i, fn := range sm.functions {
		if err := ctx.Err(); err != nil {
			sm.logger.Error("shutdown context cancelled", slog.String("error", err.Error()))
			if firstErr == nil {
				firstErr = err
			}
			break
		}

		if err := fn(ctx); err != nil {
			sm.logger.Error("shutdown function failed",
				slog.Int("index", i),
				slog.String("error", err.Error()))
			if firstErr == nil {
				firstErr = err
			}
			// Continue with remaining cleanups
		}
	}

	sm.logger.Info("ordered shutdown complete")
	return firstErr
}
