// Package main is the entry point for the ParkirPintar Notification Service (stub).
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/natssetup"
	notificationhandler "parkir-pintar/internal/notification/handler"
	notificationsub "parkir-pintar/internal/notification/subscriber"
	notificationuc "parkir-pintar/internal/notification/usecase"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/grpcserver"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/nats"
	"parkir-pintar/pkg/server"
	"parkir-pintar/pkg/tracing"
	notificationv1 "parkir-pintar/proto/notification/v1"
)

func main() {
	cfg, err := config.Load("config/.env")
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.NewLogger(cfg.Logger)
	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled: cfg.Tracing.Enabled, ServiceName: "parkir-pintar-notification",
		SampleRate: cfg.Tracing.SampleRate, Exporter: cfg.Tracing.Exporter,
	})
	if err != nil {
		log.Warn("tracer init failed", slog.Any("error", err))
		tracer = tracing.NewNoOpTracer()
	}

	natsClient, err := nats.NewClient(cfg.NATS.URL)
	if err != nil {
		log.Error("nats connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	if err := natssetup.SetupStreams(natsClient); err != nil {
		log.Error("nats stream setup failed", slog.Any("error", err))
		os.Exit(1)
	}

	uc := notificationuc.NewUsecase()
	handler := notificationhandler.NewHandler(uc)

	// Wire NATS subscriber for all domain events
	sub := notificationsub.NewNATSSubscriber()
	for _, subject := range sub.Subjects() {
		subj := subject
		streamName := streamForSubject(subj)
		consumerName := "notification-" + subj
		if err := natsClient.CreateConsumer(nats.ConsumerConfig{
			StreamName:    streamName,
			ConsumerName:  consumerName,
			FilterSubject: subj,
			DeliverPolicy: jetstream.DeliverLastPolicy,
		}); err != nil {
			log.Warn("failed to create consumer", slog.String("subject", subj), slog.Any("error", err))
			continue
		}
		go func() {
			if err := natsClient.ConsumeMessages(streamName, consumerName, func(msg jetstream.Msg) error {
				sub.HandleEvent(context.Background(), msg.Subject(), msg.Data())
				return nil
			}); err != nil {
				log.Error("NATS consume error", slog.String("subject", subj), slog.Any("error", err))
			}
		}()
	}

	shutdownMgr := server.NewShutdownManager(log)
	shutdownMgr.Register(func(_ context.Context) error { natsClient.Close(); return nil })
	shutdownMgr.Register(func(ctx context.Context) error { return tracer.Shutdown(ctx) })

	grpcSrv := grpcserver.New(log, cfg.GRPC.Server.Port, 30*time.Second)
	grpcSrv.RegisterService(&notificationv1.NotificationService_ServiceDesc, handler)
	if err := grpcSrv.Start(); err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := shutdownMgr.Shutdown(ctx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
	}
}

// streamForSubject maps a NATS subject pattern to its stream name.
func streamForSubject(subject string) string {
	switch {
	case len(subject) > 12 && subject[:12] == "reservation.":
		return "RESERVATIONS"
	case len(subject) > 8 && subject[:8] == "billing.":
		return "BILLING"
	case len(subject) > 8 && subject[:8] == "payment.":
		return "PAYMENTS"
	default:
		return "RESERVATIONS"
	}
}
