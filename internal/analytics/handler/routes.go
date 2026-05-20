package handler

import (
	"parkir-pintar/internal/analytics"
	grpchandler "parkir-pintar/internal/analytics/handler/grpc"
	natshandler "parkir-pintar/internal/analytics/handler/nats"
	pkgnats "parkir-pintar/pkg/nats"
)

// GRPCHandler is a type alias for backward compatibility with bootstrap.
type GRPCHandler = grpchandler.Handler

// NATSHandler is a type alias for backward compatibility with bootstrap.
type NATSHandler = natshandler.Handler

// NewHandler creates a new gRPC handler for the analytics service.
func NewHandler(uc analytics.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}

// NewNATSHandler creates a new NATS handler for the analytics service.
func NewNATSHandler(uc analytics.Usecase, client *pkgnats.Client) *NATSHandler {
	return natshandler.NewHandler(uc, client)
}
