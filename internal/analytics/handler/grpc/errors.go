package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/analytics/constants"
	"parkir-pintar/pkg/grpcerror"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, constants.ErrNoData):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, constants.ErrInvalidHorizon):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, constants.ErrRecordFailed):
		return status.Error(codes.Internal, err.Error())
	}

	return grpcerror.MapError(err)
}
