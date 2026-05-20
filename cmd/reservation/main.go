package main

import (
	"log/slog"
	"parkir-pintar/pkg/logger"
	"os"

	"parkir-pintar/internal/reservation/bootstrap"
)

func main() {
	app, err := bootstrap.New()
	if err != nil {
		slog.Error("failed to start", logger.Err(err))
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		slog.Error("application error", logger.Err(err))
		os.Exit(1)
	}
}
