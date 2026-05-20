package handler

import (
	"parkir-pintar/internal/search"
	grpchandler "parkir-pintar/internal/search/handler/grpc"
	natshandler "parkir-pintar/internal/search/handler/nats"
	"parkir-pintar/internal/search/sync"
	pkgnats "parkir-pintar/pkg/nats"
)

// GRPCHandler is the gRPC handler for this service.
type GRPCHandler = grpchandler.Handler

// NATSHandler is the NATS handler for this service.
type NATSHandler = natshandler.Handler

// NewHandler creates a new gRPC handler for the search service.
func NewHandler(uc search.Usecase) *GRPCHandler {
	return grpchandler.NewHandler(uc)
}

// NewNATSHandler creates a new NATS handler for the search service.
func NewNATSHandler(spotSync *sync.SpotSync, redis natshandler.RedisCache, client *pkgnats.Client, floorCount int) *NATSHandler {
	return natshandler.NewHandler(spotSync, redis, client, floorCount)
}
