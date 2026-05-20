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

type GracefulServer struct {
	engine          *gin.Engine
	logger          *slog.Logger
	port            int
	shutdownTimeout time.Duration
}

func NewGracefulServer(engine *gin.Engine, logger *slog.Logger, port int, shutdownTimeout time.Duration) *GracefulServer {
	return &GracefulServer{
		engine:          engine,
		logger:          logger,
		port:            port,
		shutdownTimeout: shutdownTimeout,
	}
}

func (s *GracefulServer) Start() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting HTTP server", slog.Int("port", s.port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed to start: %w", err)
	case sig := <-quit:
		s.logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	}

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

type ShutdownManager struct {
	logger    *slog.Logger
	functions []func(context.Context) error
}

func NewShutdownManager(logger *slog.Logger) *ShutdownManager {
	return &ShutdownManager{
		logger:    logger,
		functions: make([]func(context.Context) error, 0),
	}
}

func (sm *ShutdownManager) Register(fn func(context.Context) error) {
	sm.functions = append(sm.functions, fn)
}

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
		}
	}

	sm.logger.Info("ordered shutdown complete")
	return firstErr
}
