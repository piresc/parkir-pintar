// Package main is the entry point for the ParkirPintar Presence Service.
package main

import (
	"log/slog"
	"os"

	"parkir-pintar/internal/presence/bootstrap"
)

func main() {
	app, err := bootstrap.New()
	if err != nil {
		slog.Error("failed to initialize app", slog.Any("error", err))
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		slog.Error("app exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}
