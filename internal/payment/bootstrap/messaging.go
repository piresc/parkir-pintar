package bootstrap

import (
	"context"
	"fmt"

	paymentgateway "parkir-pintar/internal/payment/gateway"
	paymentuc "parkir-pintar/internal/payment/usecase"
	"parkir-pintar/pkg/config"
	pkgnats "parkir-pintar/pkg/nats"
)

type Messaging struct {
	NATSClient *pkgnats.Client
	Publisher  paymentuc.EventPublisher
}

func initMessaging(cfg *config.Config) (*Messaging, error) {
	m := &Messaging{}

	if !cfg.NATS.Enabled {
		return m, nil
	}

	natsClient, err := pkgnats.NewClient(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	ctx := context.Background()
	if err := pkgnats.CreateStreams(ctx, natsClient); err != nil {
		return nil, fmt.Errorf("nats create streams: %w", err)
	}

	publisher := pkgnats.NewPublisher(natsClient)
	m.NATSClient = natsClient
	m.Publisher = paymentgateway.NewPaymentEventPublisher(publisher)

	return m, nil
}
