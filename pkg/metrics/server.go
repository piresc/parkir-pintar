package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// StartMetricsServer starts a lightweight HTTP server that serves only the
// /metrics endpoint. This is useful for gRPC services that don't otherwise
// have an HTTP server. The returned *http.Server can be used to shut down
// the metrics server gracefully.
func (m *Metrics) StartMetricsServer(port int, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", m.Handler())

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("metrics server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server failed", "error", err)
		}
	}()

	return srv
}

// StopMetricsServer gracefully shuts down the metrics server.
func StopMetricsServer(srv *http.Server, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("metrics server shutdown error", "error", err)
	} else {
		logger.Info("metrics server stopped")
	}
}
