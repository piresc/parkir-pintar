package bootstrap

import (
	"context"
	"fmt"

	natsgateway "parkir-pintar/internal/reservation/gateway/nats"
	taskqueue "parkir-pintar/pkg/asynq"
	"parkir-pintar/pkg/config"
	pkgnats "parkir-pintar/pkg/nats"
)

type Messaging struct {
	NATSClient     *pkgnats.Client
	EventPublisher natsgateway.EventPublisher
	AsynqClient    *taskqueue.Client
	AsynqServer    *taskqueue.Server
}

func initAsynq(cfg *config.Config) (*taskqueue.Client, *taskqueue.Server) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	client := taskqueue.NewClient(redisAddr, cfg.Redis.Password)
	server := taskqueue.NewServer(redisAddr, cfg.Redis.Password, cfg.Asynq.Concurrency)
	return client, server
}

func initMessaging(cfg *config.Config) (*Messaging, error) {
	asynqClient, asynqServer := initAsynq(cfg)

	m := &Messaging{
		AsynqClient: asynqClient,
		AsynqServer: asynqServer,
	}

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
	if err := pkgnats.CreateConsumersForService(ctx, natsClient, "reservation"); err != nil {
		return nil, fmt.Errorf("nats create consumers: %w", err)
	}

	publisher := pkgnats.NewPublisher(natsClient)
	m.NATSClient = natsClient
	m.EventPublisher = natsgateway.NewPublisher(publisher)

	return m, nil
}
