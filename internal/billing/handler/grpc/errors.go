package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/billing/constants"
	"parkir-pintar/internal/billing/repository"
	"parkir-pintar/pkg/grpcerror"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, constants.ErrNotFound),
		errors.Is(err, repository.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, constants.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, constants.ErrCannotCalculate),
		errors.Is(err, constants.ErrCannotInvoice),
		errors.Is(err, constants.ErrInvalidStatus):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, constants.ErrConcurrentModification):
		return status.Error(codes.Unavailable, err.Error())
	}

	return grpcerror.MapError(err)
}
