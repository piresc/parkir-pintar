package handler

import (
	"google.golang.org/grpc"

	reservationv1 "parkir-pintar/proto/reservation/v1"
)

// RegisterService registers the reservation gRPC handler with the given server.
func (h *Handler) RegisterService(s *grpc.Server) {
	reservationv1.RegisterReservationServiceServer(s, h)
}
