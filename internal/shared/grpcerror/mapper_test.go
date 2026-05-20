package grpcerror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMapToGRPCError_ShouldReturnNil_WhenErrorIsNil(t *testing.T) {
	result := MapToGRPCError(nil)
	assert.Nil(t, result)
}

func TestMapToGRPCError_ShouldReturnNotFound_WhenDomainNotFoundErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"searchrepo.ErrNotFound", searchrepo.ErrNotFound},
		{"searcherrors.ErrSpotNotFound", searcherrors.ErrSpotNotFound},
		{"billingrepo.ErrNotFound", billingrepo.ErrNotFound},
		{"billingerrors.ErrNotFound", billingerrors.ErrNotFound},
		{"paymentrepo.ErrNotFound", paymentrepo.ErrNotFound},
		{"paymenterrors.ErrNotFound", paymenterrors.ErrNotFound},
		{"reservationerrors.ErrNotFound", reservationerrors.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToGRPCError(tt.err)
			require.NotNil(t, result)

			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code())
			assert.Equal(t, tt.err.Error(), st.Message())
		})
	}
}

func TestMapToGRPCError_ShouldReturnAlreadyExists_WhenConflictErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"reservationerrors.ErrConflict", reservationerrors.ErrConflict},
		{"reservationerrors.ErrAlreadyActive", reservationerrors.ErrAlreadyActive},
		{"reservationerrors.ErrSpotLocked", reservationerrors.ErrSpotLocked},
		{"reservationerrors.ErrConcurrentChange", reservationerrors.ErrConcurrentChange},
		{"billingerrors.ErrConflict", billingerrors.ErrConflict},
		{"paymenterrors.ErrConflict", paymenterrors.ErrConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToGRPCError(tt.err)
			require.NotNil(t, result)

			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, codes.AlreadyExists, st.Code())
			assert.Equal(t, tt.err.Error(), st.Message())
		})
	}
}

func TestMapToGRPCError_ShouldReturnFailedPrecondition_WhenStateErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"reservationerrors.ErrInvalidTransition", reservationerrors.ErrInvalidTransition},
		{"reservationerrors.ErrSpotUnavailable", reservationerrors.ErrSpotUnavailable},
		{"reservationerrors.ErrNotPending", reservationerrors.ErrNotPending},
		{"reservationerrors.ErrNotCheckedOut", reservationerrors.ErrNotCheckedOut},
		{"billingerrors.ErrCannotCalculate", billingerrors.ErrCannotCalculate},
		{"billingerrors.ErrCannotInvoice", billingerrors.ErrCannotInvoice},
		{"billingerrors.ErrInvalidStatus", billingerrors.ErrInvalidStatus},
		{"paymenterrors.ErrCannotRefund", paymenterrors.ErrCannotRefund},
		{"paymenterrors.ErrStatusMismatch", paymenterrors.ErrStatusMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToGRPCError(tt.err)
			require.NotNil(t, result)

			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, codes.FailedPrecondition, st.Code())
			assert.Equal(t, tt.err.Error(), st.Message())
		})
	}
}

func TestMapToGRPCError_ShouldReturnPermissionDenied_WhenForbiddenError(t *testing.T) {
	result := MapToGRPCError(reservationerrors.ErrForbidden)
	require.NotNil(t, result)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	assert.Equal(t, reservationerrors.ErrForbidden.Error(), st.Message())
}

func TestMapToGRPCError_ShouldReturnAborted_WhenPaymentBillingFailureErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"reservationerrors.ErrPaymentFailed", reservationerrors.ErrPaymentFailed},
		{"reservationerrors.ErrBillingFailed", reservationerrors.ErrBillingFailed},
		{"paymenterrors.ErrGatewayFailed", paymenterrors.ErrGatewayFailed},
		{"paymenterrors.ErrRefundFailed", paymenterrors.ErrRefundFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToGRPCError(tt.err)
			require.NotNil(t, result)

			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, codes.Aborted, st.Code())
			assert.Equal(t, tt.err.Error(), st.Message())
		})
	}
}

func TestMapToGRPCError_ShouldReturnUnavailable_WhenServiceUnavailableErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"presenceerrors.ErrSensorUnavailable", presenceerrors.ErrSensorUnavailable},
		{"searcherrors.ErrCacheUnavailable", searcherrors.ErrCacheUnavailable},
		{"billingerrors.ErrConcurrentModification", billingerrors.ErrConcurrentModification},
		{"paymenterrors.ErrCancelled", paymenterrors.ErrCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToGRPCError(tt.err)
			require.NotNil(t, result)

			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, codes.Unavailable, st.Code())
			assert.Equal(t, tt.err.Error(), st.Message())
		})
	}
}

func TestMapToGRPCError_ShouldMapAppError_WhenHTTPStatus400(t *testing.T) {
	err := apperror.New("BAD_REQUEST", "invalid input", 400)
	result := MapToGRPCError(err)
	require.NotNil(t, result)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "invalid input", st.Message())
}

func TestMapToGRPCError_ShouldMapAppError_WhenHTTPStatus402(t *testing.T) {
	err := apperror.New("PAYMENT_REQUIRED", "payment required", 402)
	result := MapToGRPCError(err)
	require.NotNil(t, result)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Aborted, st.Code())
	assert.Equal(t, "payment required", st.Message())
}

func TestMapToGRPCError_ShouldMapAppError_WhenHTTPStatus403(t *testing.T) {
	err := apperror.New("FORBIDDEN", "access denied", 403)
	result := MapToGRPCError(err)
	require.NotNil(t, result)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	assert.Equal(t, "access denied", st.Message())
}

func TestMapToGRPCError_ShouldMapAppError_WhenHTTPStatus404(t *testing.T) {
	err := apperror.New("NOT_FOUND", "resource missing", 404)
	result := MapToGRPCError(err)
	require.NotNil(t, result)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "resource missing", st.Message())
}

func TestMapToGRPCError_ShouldMapAppError_WhenHTTPStatus409(t *testing.T) {
	err := apperror.New("CONFLICT", "already exists", 409)
	result := MapToGRPCError(err)
	require.NotNil(t, result)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, "already exists", st.Message())
}

func TestMapToGRPCError_ShouldMapAppError_WhenHTTPStatusUnknown(t *testing.T) {
	err := apperror.New("INTERNAL", "something broke", 500)
	result := MapToGRPCError(err)
	require.NotNil(t, result)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "something broke", st.Message())
}

func TestMapToGRPCError_ShouldReturnInternal_WhenUnknownError(t *testing.T) {
	err := errors.New("unexpected error")
	result := MapToGRPCError(err)
	require.NotNil(t, result)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "unexpected error", st.Message())
}

func TestMapToGRPCError_ShouldHandleWrappedErrors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "wrapped NotFound",
			err:          errors.Join(reservationerrors.ErrNotFound, errors.New("extra context")),
			expectedCode: codes.NotFound,
		},
		{
			name:         "wrapped Conflict",
			err:          errors.Join(reservationerrors.ErrConflict, errors.New("extra context")),
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "wrapped FailedPrecondition",
			err:          errors.Join(reservationerrors.ErrInvalidTransition, errors.New("extra context")),
			expectedCode: codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToGRPCError(tt.err)
			require.NotNil(t, result)

			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, tt.expectedCode, st.Code())
		})
	}
}
