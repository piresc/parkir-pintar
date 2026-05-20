package handler

import (
	grpchandler "parkir-pintar/internal/payment/handler/grpc"
	"parkir-pintar/internal/payment/usecase"
)

// GRPCHandler is a type alias for backward compatibility with bootstrap.
type GRPCHandler = grpchandler.Handler

// NewHandler creates a new gRPC handler for the payment service.
func NewHandler(uc usecase.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}
