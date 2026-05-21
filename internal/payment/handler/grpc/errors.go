package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/payment/constants"
	"parkir-pintar/internal/payment/repository"
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
	case errors.Is(err, constants.ErrStatusMismatch),
		errors.Is(err, constants.ErrCannotRefund):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, constants.ErrGatewayFailed),
		errors.Is(err, constants.ErrRefundFailed):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, constants.ErrCancelled):
		return status.Error(codes.Unavailable, err.Error())
	}

	return grpcerror.MapError(err)
}
