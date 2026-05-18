// Package grpcerror provides gRPC error mapping utilities.
package grpcerror

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	billingrepo "parkir-pintar/internal/billing/repository"
	paymentrepo "parkir-pintar/internal/payment/repository"
	reservationmodel "parkir-pintar/internal/reservation/model"
	searchrepo "parkir-pintar/internal/search/repository"
	"parkir-pintar/pkg/apperror"
)

// MapToGRPCError maps domain errors to gRPC status codes. It is a superset
// of all sentinel errors used across the search, billing, reservation, and
// payment handler packages.
func MapToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// Check repository/model sentinel errors (NotFound variants).
	if errors.Is(err, searchrepo.ErrNotFound) ||
		errors.Is(err, billingrepo.ErrNotFound) ||
		errors.Is(err, paymentrepo.ErrNotFound) ||
		errors.Is(err, reservationmodel.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	// Check reservation model sentinel errors.
	if errors.Is(err, reservationmodel.ErrConflict) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if errors.Is(err, reservationmodel.ErrInvalidTransition) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}
	if errors.Is(err, reservationmodel.ErrSpotUnavailable) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	// Check structured AppError with HTTP status mapping.
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		switch appErr.HTTPStatus {
		case 400:
			return status.Error(codes.InvalidArgument, appErr.Message)
		case 403:
			return status.Error(codes.PermissionDenied, appErr.Message)
		case 404:
			return status.Error(codes.NotFound, appErr.Message)
		case 409:
			return status.Error(codes.AlreadyExists, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}

	return status.Error(codes.Internal, err.Error())
}
