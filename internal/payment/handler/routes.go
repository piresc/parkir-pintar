package handler

import (
	"parkir-pintar/internal/payment"
	grpchandler "parkir-pintar/internal/payment/handler/grpc"
)

// GRPCHandler is a type alias for backward compatibility with bootstrap.
type GRPCHandler = grpchandler.Handler

// NewHandler creates a new gRPC handler for the payment service.
func NewHandler(uc payment.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}
