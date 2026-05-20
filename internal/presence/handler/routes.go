package handler

import (
	"parkir-pintar/internal/presence"
	grpchandler "parkir-pintar/internal/presence/handler/grpc"
)

// GRPCHandler is the gRPC handler for this service.
type GRPCHandler = grpchandler.Handler

// NewHandler creates a new gRPC handler for the presence service.
func NewHandler(uc presence.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}
