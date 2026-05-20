package bootstrap

import (
	"context"
	"log/slog"
	"parkir-pintar/pkg/logger"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/grpcclient"
	"parkir-pintar/pkg/tracing"

	billingv1 "parkir-pintar/proto/billing/v1"
	paymentv1 "parkir-pintar/proto/payment/v1"
	presencev1 "parkir-pintar/proto/presence/v1"

	"google.golang.org/grpc"
)

type Clients struct {
	BillingConn  *grpc.ClientConn
	PaymentConn  *grpc.ClientConn
	PresenceConn *grpc.ClientConn // may be nil if presence is unavailable

	BillingGRPC  billingv1.BillingServiceClient
	PaymentGRPC  paymentv1.PaymentServiceClient
	PresenceGRPC presencev1.PresenceServiceClient // may be nil
}

func initClients(cfg *config.Config, tracer tracing.Tracer, log *slog.Logger) (*Clients, error) {
	billingTarget := getEnv("GRPC_BILLING_TARGET", "localhost:9093")
	paymentTarget := getEnv("GRPC_PAYMENT_TARGET", "localhost:9094")

	clientCfg := grpcclient.ClientConfig{
		DialTimeout:      cfg.GRPC.Client.DialTimeout,
		KeepAliveTime:    cfg.GRPC.Client.KeepAliveTime,
		KeepAliveTimeout: cfg.GRPC.Client.KeepAliveTimeout,
		Tracer:           tracer,
		Logger:           log,
	}

	clientCfg.Target = billingTarget
	billingConn, err := grpcclient.Dial(context.Background(), clientCfg)
	if err != nil {
		return nil, err
	}

	clientCfg.Target = paymentTarget
	paymentConn, err := grpcclient.Dial(context.Background(), clientCfg)
	if err != nil {
		_ = billingConn.Close()
		return nil, err
	}

	clients := &Clients{
		BillingConn: billingConn,
		PaymentConn: paymentConn,
		BillingGRPC: billingv1.NewBillingServiceClient(billingConn),
		PaymentGRPC: paymentv1.NewPaymentServiceClient(paymentConn),
	}

	// Dial presence service (optional — graceful degradation if unavailable).
	presenceTarget := getEnv("GRPC_PRESENCE_TARGET", "localhost:9095")
	clientCfg.Target = presenceTarget
	presenceConn, presenceErr := grpcclient.Dial(context.Background(), clientCfg)
	if presenceErr != nil {
		log.Warn("presence service unavailable, presence verification disabled", logger.Err(presenceErr))
	} else {
		clients.PresenceConn = presenceConn
		clients.PresenceGRPC = presencev1.NewPresenceServiceClient(presenceConn)
	}

	return clients, nil
}
