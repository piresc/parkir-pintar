package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/reservation/constants"
	reservationerrors "parkir-pintar/internal/reservation/errors"
	"parkir-pintar/internal/reservation/model"
	reservationv1 "parkir-pintar/proto/reservation/v1"
)

// mockUsecase is a testify mock for the reservation usecase.Usecase interface.
type mockUsecase struct {
	mock.Mock
}

func (m *mockUsecase) CreateReservation(ctx context.Context, req *model.CreateReservationRequest) (*model.Reservation, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *mockUsecase) GetReservation(ctx context.Context, id string, callerID string) (*model.Reservation, error) {
	args := m.Called(ctx, id, callerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *mockUsecase) CancelReservation(ctx context.Context, req *model.CancelReservationRequest) (*model.Reservation, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *mockUsecase) CheckIn(ctx context.Context, req *model.CheckInRequest) (*model.CheckInResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.CheckInResponse), args.Error(1)
}

func (m *mockUsecase) CheckOut(ctx context.Context, req *model.CheckOutRequest) (*model.CheckOutResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.CheckOutResponse), args.Error(1)
}

func (m *mockUsecase) ConfirmReservation(ctx context.Context, req *model.ConfirmReservationRequest) (*model.Reservation, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *mockUsecase) CompleteCheckout(ctx context.Context, req *model.CompleteCheckoutRequest) (*model.CheckOutResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.CheckOutResponse), args.Error(1)
}

func (m *mockUsecase) ExpireReservation(ctx context.Context, req *model.ExpireReservationRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockUsecase) FailReservation(ctx context.Context, req *model.FailReservationRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockUsecase) ListByDriver(ctx context.Context, driverID string, status string) ([]*model.Reservation, error) {
	args := m.Called(ctx, driverID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Reservation), args.Error(1)
}

func TestCreateReservation(t *testing.T) {
	tests := []struct {
		name       string
		req        *reservationv1.CreateReservationRequest
		mockResult *model.Reservation
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.CreateReservationRequest{
				DriverId:       "driver-123",
				VehicleType:    "car",
				AssignmentMode: "system_assigned",
				IdempotencyKey: "idem-res-123",
			},
			mockResult: &model.Reservation{
				ID:             "res-123",
				DriverID:       "driver-123",
				SpotID:         "spot-1",
				VehicleType:    "car",
				AssignmentMode: "system_assigned",
				Status:         string(constants.StatusWaitingPayment),
				IdempotencyKey: "idem-res-123",
			},
			wantCode: codes.OK,
		},
		{
			name: "missing driver_id",
			req: &reservationv1.CreateReservationRequest{
				DriverId:       "",
				VehicleType:    "car",
				IdempotencyKey: "idem-res-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing vehicle_type",
			req: &reservationv1.CreateReservationRequest{
				DriverId:       "driver-123",
				VehicleType:    "",
				IdempotencyKey: "idem-res-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing idempotency_key",
			req: &reservationv1.CreateReservationRequest{
				DriverId:       "driver-123",
				VehicleType:    "car",
				IdempotencyKey: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns not found",
			req: &reservationv1.CreateReservationRequest{
				DriverId:       "driver-123",
				VehicleType:    "car",
				IdempotencyKey: "idem-res-123",
			},
			mockResult: nil,
			mockErr:    reservationerrors.ErrNotFound,
			wantCode:   codes.NotFound,
		},
		{
			name: "usecase returns conflict",
			req: &reservationv1.CreateReservationRequest{
				DriverId:       "driver-123",
				VehicleType:    "car",
				IdempotencyKey: "idem-res-123",
			},
			mockResult: nil,
			mockErr:    reservationerrors.ErrConflict,
			wantCode:   codes.AlreadyExists,
		},
		{
			name: "usecase returns spot unavailable",
			req: &reservationv1.CreateReservationRequest{
				DriverId:       "driver-123",
				VehicleType:    "car",
				IdempotencyKey: "idem-res-123",
			},
			mockResult: nil,
			mockErr:    reservationerrors.ErrSpotUnavailable,
			wantCode:   codes.FailedPrecondition,
		},
		{
			name: "usecase returns internal error",
			req: &reservationv1.CreateReservationRequest{
				DriverId:       "driver-123",
				VehicleType:    "car",
				IdempotencyKey: "idem-res-123",
			},
			mockResult: nil,
			mockErr:    errors.New("unexpected error"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetDriverId() != "" && tt.req.GetVehicleType() != "" && tt.req.GetIdempotencyKey() != "" {
				uc.On("CreateReservation", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.CreateReservation(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
				assert.Equal(t, tt.mockResult.DriverID, resp.GetDriverId())
				assert.Equal(t, tt.mockResult.Status, resp.GetStatus())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestGetReservation(t *testing.T) {
	tests := []struct {
		name       string
		req        *reservationv1.GetReservationRequest
		mockResult *model.Reservation
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.GetReservationRequest{
				ReservationId: "res-123",
			},
			mockResult: &model.Reservation{
				ID:       "res-123",
				DriverID: "driver-123",
				Status:   string(constants.StatusConfirmed),
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &reservationv1.GetReservationRequest{
				ReservationId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns not found",
			req: &reservationv1.GetReservationRequest{
				ReservationId: "res-999",
			},
			mockResult: nil,
			mockErr:    reservationerrors.ErrNotFound,
			wantCode:   codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" {
				uc.On("GetReservation", mock.Anything, tt.req.GetReservationId(), "").Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.GetReservation(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestCancelReservation(t *testing.T) {
	tests := []struct {
		name       string
		req        *reservationv1.CancelReservationRequest
		mockResult *model.Reservation
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.CancelReservationRequest{
				ReservationId: "res-123",
			},
			mockResult: &model.Reservation{
				ID:     "res-123",
				Status: string(constants.StatusCancelled),
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &reservationv1.CancelReservationRequest{
				ReservationId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns invalid transition",
			req: &reservationv1.CancelReservationRequest{
				ReservationId: "res-123",
			},
			mockResult: nil,
			mockErr:    reservationerrors.ErrInvalidTransition,
			wantCode:   codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" {
				uc.On("CancelReservation", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.CancelReservation(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
				assert.Equal(t, tt.mockResult.Status, resp.GetStatus())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestCheckIn(t *testing.T) {
	tests := []struct {
		name       string
		req        *reservationv1.CheckInRequest
		mockResult *model.CheckInResponse
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.CheckInRequest{
				ReservationId: "res-123",
			},
			mockResult: &model.CheckInResponse{
				Reservation: &model.Reservation{
					ID:     "res-123",
					Status: string(constants.StatusCheckedIn),
				},
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &reservationv1.CheckInRequest{
				ReservationId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns invalid transition",
			req: &reservationv1.CheckInRequest{
				ReservationId: "res-123",
			},
			mockResult: nil,
			mockErr:    reservationerrors.ErrInvalidTransition,
			wantCode:   codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" {
				uc.On("CheckIn", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.CheckIn(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.Reservation.ID, resp.GetId())
				assert.Equal(t, tt.mockResult.Reservation.Status, resp.GetStatus())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestCheckOut(t *testing.T) {
	tests := []struct {
		name       string
		req        *reservationv1.CheckOutRequest
		mockResult *model.CheckOutResponse
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.CheckOutRequest{
				ReservationId: "res-123",
			},
			mockResult: &model.CheckOutResponse{
				Reservation: &model.Reservation{
					ID:     "res-123",
					Status: string(constants.StatusCheckedOut),
				},
				TotalAmount:  25000,
				BillingID:    "bill-123",
				BookingFee:   5000,
				ParkingFee:   20000,
				OvernightFee: 0,
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &reservationv1.CheckOutRequest{
				ReservationId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &reservationv1.CheckOutRequest{
				ReservationId: "res-123",
			},
			mockResult: nil,
			mockErr:    reservationerrors.ErrInvalidTransition,
			wantCode:   codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" {
				uc.On("CheckOut", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.CheckOut(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.TotalAmount, resp.GetTotalAmount())
				assert.Equal(t, tt.mockResult.BillingID, resp.GetBillingId())
				assert.Equal(t, tt.mockResult.BookingFee, resp.GetBookingFee())
				assert.Equal(t, tt.mockResult.ParkingFee, resp.GetParkingFee())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestConfirmReservation(t *testing.T) {
	tests := []struct {
		name       string
		req        *reservationv1.ConfirmReservationRequest
		mockResult *model.Reservation
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.ConfirmReservationRequest{
				ReservationId: "res-123",
			},
			mockResult: &model.Reservation{
				ID:     "res-123",
				Status: string(constants.StatusConfirmed),
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &reservationv1.ConfirmReservationRequest{
				ReservationId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &reservationv1.ConfirmReservationRequest{
				ReservationId: "res-123",
			},
			mockResult: nil,
			mockErr:    errors.New("payment failed"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" {
				uc.On("ConfirmReservation", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.ConfirmReservation(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
				assert.Equal(t, tt.mockResult.Status, resp.GetStatus())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestCompleteCheckout(t *testing.T) {
	tests := []struct {
		name       string
		req        *reservationv1.CompleteCheckoutRequest
		mockResult *model.CheckOutResponse
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.CompleteCheckoutRequest{
				ReservationId: "res-123",
			},
			mockResult: &model.CheckOutResponse{
				Reservation: &model.Reservation{
					ID:     "res-123",
					Status: string(constants.StatusCheckedOut),
				},
				TotalAmount: 25000,
				BillingID:   "bill-123",
				PaymentID:   "pay-123",
				BookingFee:  5000,
				ParkingFee:  20000,
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &reservationv1.CompleteCheckoutRequest{
				ReservationId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &reservationv1.CompleteCheckoutRequest{
				ReservationId: "res-123",
			},
			mockResult: nil,
			mockErr:    reservationerrors.ErrNotFound,
			wantCode:   codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" {
				uc.On("CompleteCheckout", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.CompleteCheckout(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.TotalAmount, resp.GetTotalAmount())
				assert.Equal(t, tt.mockResult.PaymentID, resp.GetPaymentId())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestExpireReservation(t *testing.T) {
	tests := []struct {
		name     string
		req      *reservationv1.ExpireReservationRequest
		mockErr  error
		wantCode codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.ExpireReservationRequest{
				ReservationId: "res-123",
			},
			mockErr:  nil,
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &reservationv1.ExpireReservationRequest{
				ReservationId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &reservationv1.ExpireReservationRequest{
				ReservationId: "res-123",
			},
			mockErr:  reservationerrors.ErrNotFound,
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" {
				uc.On("ExpireReservation", mock.Anything, mock.Anything).Return(tt.mockErr)
			}

			resp, err := h.ExpireReservation(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestListByDriver(t *testing.T) {
	tests := []struct {
		name       string
		req        *reservationv1.ListByDriverRequest
		mockResult []*model.Reservation
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &reservationv1.ListByDriverRequest{
				DriverId: "driver-123",
				Status:   "confirmed",
			},
			mockResult: []*model.Reservation{
				{ID: "res-1", DriverID: "driver-123", Status: string(constants.StatusConfirmed)},
				{ID: "res-2", DriverID: "driver-123", Status: string(constants.StatusConfirmed)},
			},
			wantCode: codes.OK,
		},
		{
			name: "missing driver_id",
			req: &reservationv1.ListByDriverRequest{
				DriverId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &reservationv1.ListByDriverRequest{
				DriverId: "driver-123",
			},
			mockResult: nil,
			mockErr:    errors.New("db error"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetDriverId() != "" {
				uc.On("ListByDriver", mock.Anything, tt.req.GetDriverId(), tt.req.GetStatus()).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.ListByDriver(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.GetReservations(), len(tt.mockResult))
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}
