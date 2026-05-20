package handler

import (
	"google.golang.org/grpc"

	paymentv1 "parkir-pintar/proto/payment/v1"
)

// RegisterService registers the payment gRPC handler with the given server.
func (h *Handler) RegisterService(s *grpc.Server) {
	paymentv1.RegisterPaymentServiceServer(s, h)
}
