// Package main is the entry point for the ParkirPintar Presence Service.
package main

import (
	"log/slog"
	"parkir-pintar/pkg/logger"
	"os"

	"parkir-pintar/internal/presence/bootstrap"
)

func main() {
	app, err := bootstrap.New()
	if err != nil {
		slog.Error("failed to initialize app", logger.Err(err))
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		slog.Error("app exited with error", logger.Err(err))
		os.Exit(1)
	}
}
