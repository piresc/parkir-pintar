package handler

import (
	"parkir-pintar/internal/reservation"
	grpchandler "parkir-pintar/internal/reservation/handler/grpc"
	natshandler "parkir-pintar/internal/reservation/handler/nats"
	pkgnats "parkir-pintar/pkg/nats"
)

// GRPCHandler is the gRPC handler for this service.
type GRPCHandler = grpchandler.Handler

// NATSHandler is the NATS handler for this service.
type NATSHandler = natshandler.Handler

// NewHandler creates a new gRPC handler for the reservation service.
func NewHandler(uc reservation.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}

// NewNATSHandler creates a new NATS handler for the reservation service.
func NewNATSHandler(uc natshandler.ReservationConfirmer, client *pkgnats.Client) *NATSHandler {
	return natshandler.NewHandler(uc, client)
}
