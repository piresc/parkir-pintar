package handler

import (
	"google.golang.org/grpc"

	searchv1 "parkir-pintar/proto/search/v1"
)

// RegisterService registers the search gRPC handler with the given server.
func (h *Handler) RegisterService(s *grpc.Server) {
	searchv1.RegisterSearchServiceServer(s, h)
}
