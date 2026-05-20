package handler

import (
	"google.golang.org/grpc"

	presencev1 "parkir-pintar/proto/presence/v1"
)

// RegisterService registers the presence gRPC handler with the given server.
func (h *Handler) RegisterService(s *grpc.Server) {
	presencev1.RegisterPresenceServiceServer(s, h)
}
