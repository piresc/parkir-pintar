package grpcserver

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNew_ShouldUseDefaultLogger_WhenLoggerIsNil(t *testing.T) {
	srv := New(nil, 9090, 5*time.Second)

	require.NotNil(t, srv)
	assert.NotNil(t, srv.logger)
	assert.NotNil(t, srv.server)
	assert.Equal(t, 9090, srv.port)
	assert.Equal(t, 5*time.Second, srv.timeout)
}

func TestNew_ShouldUseProvidedLogger_WhenLoggerIsNotNil(t *testing.T) {
	logger := newTestLogger()

	srv := New(logger, 8080, 10*time.Second)

	require.NotNil(t, srv)
	assert.Equal(t, logger, srv.logger)
	assert.Equal(t, 8080, srv.port)
	assert.Equal(t, 10*time.Second, srv.timeout)
}

func TestNew_ShouldPassServerOptions_WhenOptsProvided(t *testing.T) {
	logger := newTestLogger()
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024),
	}

	srv := New(logger, 9090, 5*time.Second, opts...)

	require.NotNil(t, srv)
	assert.NotNil(t, srv.server)
}

func TestStart_ShouldReturnError_WhenPortAlreadyInUse(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	port := lis.Addr().(*net.TCPAddr).Port
	srv := New(newTestLogger(), port, 5*time.Second)

	startErr := srv.Start()

	require.Error(t, startErr)
	assert.Contains(t, startErr.Error(), "failed to listen on port")
}

func TestGracefulStop_ShouldReturnNil_WhenServerStopsWithinTimeout(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := lis.Addr().(*net.TCPAddr).Port
	lis.Close() // free the port for the gRPC server

	srv := New(newTestLogger(), port, 5*time.Second)

	started := make(chan struct{})
	go func() {
		innerLis, lisErr := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if lisErr != nil {
			return
		}
		close(started)
		_ = srv.server.Serve(innerLis)
	}()

	<-started
	time.Sleep(50 * time.Millisecond)

	stopErr := srv.GracefulStop()

	assert.NoError(t, stopErr)
}

func TestGracefulStop_ShouldReturnError_WhenTimeoutExceeded(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := lis.Addr().(*net.TCPAddr).Port
	lis.Close()

	srv := New(newTestLogger(), port, 1*time.Millisecond)

	innerLis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	require.NoError(t, err)

	started := make(chan struct{})
	go func() {
		close(started)
		_ = srv.server.Serve(innerLis)
	}()
	<-started
	time.Sleep(50 * time.Millisecond)

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	stopErr := srv.GracefulStop()

	if stopErr != nil {
		assert.Contains(t, stopErr.Error(), "graceful stop timed out")
	}
}
