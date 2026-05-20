package handler

import (
	grpchandler "parkir-pintar/internal/analytics/handler/grpc"
	natshandler "parkir-pintar/internal/analytics/handler/nats"
	"parkir-pintar/internal/analytics/usecase"
	pkgnats "parkir-pintar/pkg/nats"
)

// GRPCHandler is a type alias for backward compatibility with bootstrap.
type GRPCHandler = grpchandler.Handler

// NATSHandler is a type alias for backward compatibility with bootstrap.
type NATSHandler = natshandler.Handler

// NewHandler creates a new gRPC handler for the analytics service.
func NewHandler(uc usecase.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}

// NewNATSHandler creates a new NATS handler for the analytics service.
func NewNATSHandler(uc usecase.Usecase, client *pkgnats.Client) *NATSHandler {
	return natshandler.NewHandler(uc, client)
}
