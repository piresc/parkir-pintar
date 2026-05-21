package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/pkg/grpcerror"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, constants.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, constants.ErrConflict),
		errors.Is(err, constants.ErrAlreadyActive),
		errors.Is(err, constants.ErrSpotLocked),
		errors.Is(err, constants.ErrConcurrentChange):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, constants.ErrInvalidTransition),
		errors.Is(err, constants.ErrSpotUnavailable),
		errors.Is(err, constants.ErrNotPending),
		errors.Is(err, constants.ErrNotCheckedOut):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, constants.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, constants.ErrPaymentFailed),
		errors.Is(err, constants.ErrBillingFailed):
		return status.Error(codes.Aborted, err.Error())
	}

	return grpcerror.MapError(err)
}
