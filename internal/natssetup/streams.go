// Package natssetup provides shared NATS JetStream stream and consumer
// configuration for the ParkirPintar microservices. It ensures all required
// streams exist before services start publishing or consuming events.
package natssetup

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	pkgnats "parkir-pintar/pkg/nats"
)

// NATSClient defines the subset of pkg/nats.Client methods needed for stream setup.
type NATSClient interface {
	CreateOrUpdateStream(cfg pkgnats.StreamConfig) error
	CreateConsumer(cfg pkgnats.ConsumerConfig) error
}

// streamDef groups a stream name with its subjects for concise declaration.
type streamDef struct {
	Name     string
	Subjects []string
}

// streams defines the four JetStream streams used across ParkirPintar services.
var streams = []streamDef{
	{
		Name: "RESERVATIONS",
		Subjects: []string{
			"reservation.confirmed",
			"reservation.checked_in",
			"reservation.checked_out",
			"reservation.expired",
			"reservation.cancelled",
			"spot.updated",
		},
	},
	{
		Name: "BILLING",
		Subjects: []string{
			"billing.calculated",
			"billing.invoiced",
		},
	},
	{
		Name: "PAYMENTS",
		Subjects: []string{
			"payment.success",
			"payment.failed",
		},
	},
	{
		Name: "PRESENCE",
		Subjects: []string{
			"presence.arrival",
			"presence.wrong_spot",
		},
	},
}

// SetupStreams creates or updates all JetStream streams required by ParkirPintar.
// It uses file-based storage, limits retention, and a 72-hour max age for events.
// Errors are returned immediately if any stream fails to create.
func SetupStreams(client NATSClient) error {
	for _, s := range streams {
		cfg := pkgnats.StreamConfig{
			Name:      s.Name,
			Subjects:  s.Subjects,
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			Replicas:  1,
			MaxAge:    72 * time.Hour,
			MaxBytes:  -1,
			MaxMsgs:   -1,
			Discard:   jetstream.DiscardOld,
		}
		if err := client.CreateOrUpdateStream(cfg); err != nil {
			return fmt.Errorf("setup stream %s: %w", s.Name, err)
		}
		slog.Info("NATS stream configured", slog.String("stream", s.Name), slog.Any("subjects", s.Subjects))
	}
	return nil
}
