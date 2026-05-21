package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/search/constants"
	"parkir-pintar/internal/search/repository"
	"parkir-pintar/pkg/grpcerror"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, constants.ErrSpotNotFound),
		errors.Is(err, constants.ErrFloorNotFound),
		errors.Is(err, repository.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, constants.ErrCacheUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, constants.ErrInvalidVehicleType):
		return status.Error(codes.InvalidArgument, err.Error())
	}

	return grpcerror.MapError(err)
}
