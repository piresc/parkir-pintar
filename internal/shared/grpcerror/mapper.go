// Package grpcerror provides gRPC error mapping utilities.
package grpcerror

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	billingerrors "parkir-pintar/internal/billing/constants"
	billingrepo "parkir-pintar/internal/billing/repository"
	paymenterrors "parkir-pintar/internal/payment/constants"
	paymentrepo "parkir-pintar/internal/payment/repository"
	presenceerrors "parkir-pintar/internal/presence/constants"
	reservationerrors "parkir-pintar/internal/reservation/constants"
	searcherrors "parkir-pintar/internal/search/constants"
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

	// Check domain-specific NotFound errors.
	if errors.Is(err, searchrepo.ErrNotFound) ||
		errors.Is(err, searcherrors.ErrSpotNotFound) ||
		errors.Is(err, billingrepo.ErrNotFound) ||
		errors.Is(err, billingerrors.ErrNotFound) ||
		errors.Is(err, paymentrepo.ErrNotFound) ||
		errors.Is(err, paymenterrors.ErrNotFound) ||
		errors.Is(err, reservationerrors.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	// Check conflict/already-exists errors.
	if errors.Is(err, reservationerrors.ErrConflict) ||
		errors.Is(err, reservationerrors.ErrAlreadyActive) ||
		errors.Is(err, reservationerrors.ErrSpotLocked) ||
		errors.Is(err, reservationerrors.ErrConcurrentChange) ||
		errors.Is(err, billingerrors.ErrConflict) ||
		errors.Is(err, paymenterrors.ErrConflict) {
		return status.Error(codes.AlreadyExists, err.Error())
	}

	// Check precondition/state errors.
	if errors.Is(err, reservationerrors.ErrInvalidTransition) ||
		errors.Is(err, reservationerrors.ErrSpotUnavailable) ||
		errors.Is(err, reservationerrors.ErrNotPending) ||
		errors.Is(err, reservationerrors.ErrNotCheckedOut) ||
		errors.Is(err, billingerrors.ErrCannotCalculate) ||
		errors.Is(err, billingerrors.ErrCannotInvoice) ||
		errors.Is(err, billingerrors.ErrInvalidStatus) ||
		errors.Is(err, paymenterrors.ErrCannotRefund) ||
		errors.Is(err, paymenterrors.ErrStatusMismatch) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	// Check permission errors.
	if errors.Is(err, reservationerrors.ErrForbidden) {
		return status.Error(codes.PermissionDenied, err.Error())
	}

	// Check payment/billing failure errors.
	if errors.Is(err, reservationerrors.ErrPaymentFailed) ||
		errors.Is(err, reservationerrors.ErrBillingFailed) ||
		errors.Is(err, paymenterrors.ErrGatewayFailed) ||
		errors.Is(err, paymenterrors.ErrRefundFailed) {
		return status.Error(codes.Aborted, err.Error())
	}

	// Check unavailability errors.
	if errors.Is(err, presenceerrors.ErrSensorUnavailable) ||
		errors.Is(err, searcherrors.ErrCacheUnavailable) ||
		errors.Is(err, billingerrors.ErrConcurrentModification) ||
		errors.Is(err, paymenterrors.ErrCancelled) {
		return status.Error(codes.Unavailable, err.Error())
	}

	// Check structured AppError with HTTP status mapping.
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		switch appErr.HTTPStatus {
		case 400:
			return status.Error(codes.InvalidArgument, appErr.Message)
		case 402:
			return status.Error(codes.Aborted, appErr.Message)
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
