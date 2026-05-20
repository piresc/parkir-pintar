package handler

import (
	grpchandler "parkir-pintar/internal/presence/handler/grpc"
	"parkir-pintar/internal/presence/usecase"
)

// GRPCHandler is a type alias for backward compatibility with bootstrap.
type GRPCHandler = grpchandler.Handler

// NewHandler creates a new gRPC handler for the presence service.
func NewHandler(uc usecase.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}
