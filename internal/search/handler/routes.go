package handler

import (
	grpchandler "parkir-pintar/internal/search/handler/grpc"
	natshandler "parkir-pintar/internal/search/handler/nats"
	"parkir-pintar/internal/search/sync"
	"parkir-pintar/internal/search/usecase"
	pkgnats "parkir-pintar/pkg/nats"
)

// GRPCHandler is a type alias for backward compatibility with bootstrap.
type GRPCHandler = grpchandler.Handler

// NATSHandler is a type alias for backward compatibility with bootstrap.
type NATSHandler = natshandler.Handler

// NewHandler creates a new gRPC handler for the search service.
func NewHandler(uc usecase.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}

// NewNATSHandler creates a new NATS handler for the search service.
func NewNATSHandler(spotSync *sync.SpotSync, redis natshandler.RedisCache, client *pkgnats.Client, floorCount int) *NATSHandler {
	return natshandler.NewHandler(spotSync, redis, client, floorCount)
}
