package handler

import (
	"parkir-pintar/internal/billing"
	grpchandler "parkir-pintar/internal/billing/handler/grpc"
)

// GRPCHandler is a type alias for backward compatibility with bootstrap.
type GRPCHandler = grpchandler.Handler

// NewHandler creates a new gRPC handler for the billing service.
func NewHandler(uc billing.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}
