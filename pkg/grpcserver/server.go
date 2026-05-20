package grpcserver

import (
	"context"
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

type GRPCServer struct {
	server  *grpc.Server
	logger  *slog.Logger
	port    int
	timeout time.Duration
}

// OTel server stats handler is always prepended to extract incoming trace
func New(logger *slog.Logger, port int, shutdownTimeout time.Duration, opts ...grpc.ServerOption) *GRPCServer {
	if logger == nil {
		logger = slog.Default()
	}
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

func (s *GRPCServer) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
}

// Server returns the underlying *grpc.Server for direct service registration.
func (s *GRPCServer) Server() *grpc.Server {
	return s.server
}

func (s *GRPCServer) Start() error {
	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting gRPC server", slog.Int("port", s.port))
		if err := s.server.Serve(lis); err != nil {
			errCh <- err
		}
	}()

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
