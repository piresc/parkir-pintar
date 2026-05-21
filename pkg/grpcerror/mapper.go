package grpcerror

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/pkg/apperror"
)

// MapError maps domain errors to gRPC status codes.
// Each service handler should check its own sentinel errors first,
// then fall through to this for AppError and generic mapping.
func MapError(err error) error {
	if err == nil {
		return nil
	}

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
		case 503:
			return status.Error(codes.Unavailable, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}

	return status.Error(codes.Internal, err.Error())
}
