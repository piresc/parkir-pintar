package handler

import (
	grpchandler "parkir-pintar/internal/billing/handler/grpc"
	"parkir-pintar/internal/billing/usecase"
)

// GRPCHandler is a type alias for backward compatibility with bootstrap.
type GRPCHandler = grpchandler.Handler

// NewHandler creates a new gRPC handler for the billing service.
func NewHandler(uc usecase.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}
