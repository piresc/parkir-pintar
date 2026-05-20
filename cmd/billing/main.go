package main

import (
	"log/slog"
	"os"

	"parkir-pintar/internal/billing/bootstrap"
)

func main() {
	app, err := bootstrap.New()
	if err != nil {
		slog.Error("failed to start", slog.Any("error", err))
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		slog.Error("application error", slog.Any("error", err))
		os.Exit(1)
	}
}
