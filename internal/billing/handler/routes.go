package handler

import (
	"google.golang.org/grpc"

	billingv1 "parkir-pintar/proto/billing/v1"
)

// RegisterService registers the billing gRPC handler with the given server.
func (h *Handler) RegisterService(s *grpc.Server) {
	billingv1.RegisterBillingServiceServer(s, h)
}
