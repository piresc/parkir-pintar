// Package grpcserver provides a reusable gRPC server with signal-based
// graceful shutdown, mirroring the pattern from pkg/server.GracefulServer.
package grpcserver

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// GRPCServer wraps a grpc.Server with signal-based graceful shutdown.
type GRPCServer struct {
	server  *grpc.Server
	logger  *slog.Logger
	port    int
	timeout time.Duration
}

// New creates a GRPCServer. If logger is nil, slog.Default() is used.
// opts are passed directly to grpc.NewServer().
// OTel server stats handler is always prepended to extract incoming trace
// context (W3C traceparent) so spans join the caller's trace.
func New(logger *slog.Logger, port int, shutdownTimeout time.Duration, opts ...grpc.ServerOption) *GRPCServer {
	if logger == nil {
		logger = slog.Default()
	}
	// Prepend OTel handler so it runs before user-supplied options.
	allOpts := append([]grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}, opts...)
	return &GRPCServer{
		server:  grpc.NewServer(allOpts...),
		logger:  logger,
		port:    port,
		timeout: shutdownTimeout,
	}
}

// RegisterService registers a gRPC service implementation before Start is called.
func (s *GRPCServer) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
}

// Start listens on the configured port and blocks until SIGINT or SIGTERM.
// Returns an error if the port bind fails or graceful stop times out.
func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	// Channel to capture serve errors
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting gRPC server", slog.Int("port", s.port))
		if err := s.server.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("gRPC server failed: %w", err)
	case sig := <-quit:
		s.logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	}

	return s.GracefulStop()
}

// GracefulStop initiates a graceful shutdown with the configured timeout.
// If the timeout is exceeded, the server is force-stopped and an error is returned.
func (s *GRPCServer) GracefulStop() error {
	s.logger.Info("shutting down gRPC server")

	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	timer := time.NewTimer(s.timeout)
	defer timer.Stop()

	select {
	case <-done:
		s.logger.Info("gRPC server stopped gracefully")
		return nil
	case <-timer.C:
		s.server.Stop()
		return fmt.Errorf("graceful stop timed out")
	}
}
