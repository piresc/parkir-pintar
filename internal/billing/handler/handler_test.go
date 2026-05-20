package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/repository"
	billingv1 "parkir-pintar/proto/billing/v1"

	"time"
)

type mockUsecase struct {
	mock.Mock
}

func (m *mockUsecase) StartBilling(ctx context.Context, req *model.StartBillingRequest) (*model.BillingRecord, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.BillingRecord), args.Error(1)
}

func (m *mockUsecase) CalculateFee(ctx context.Context, req *model.CalculateFeeRequest) (*model.BillingRecord, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.BillingRecord), args.Error(1)
}

func (m *mockUsecase) GenerateInvoice(ctx context.Context, req *model.GenerateInvoiceRequest) (*model.BillingRecord, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.BillingRecord), args.Error(1)
}

func (m *mockUsecase) ApplyOvernightFee(ctx context.Context, req *model.ApplyOvernightFeeRequest) (*model.BillingRecord, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.BillingRecord), args.Error(1)
}

func TestStartBilling(t *testing.T) {
	tests := []struct {
		name       string
		req        *billingv1.StartBillingRequest
		mockResult *model.BillingRecord
		mockErr    error
		wantCode   codes.Code
		wantNil    bool
	}{
		{
			name: "happy path",
			req: &billingv1.StartBillingRequest{
				ReservationId:  "res-123",
				BookingFee:     5000,
				IdempotencyKey: "idem-123",
			},
			mockResult: &model.BillingRecord{
				ID:             "bill-123",
				ReservationID:  "res-123",
				BookingFee:     5000,
				TotalAmount:    5000,
				IdempotencyKey: "idem-123",
				Status:         model.BillingStatusPending,
			},
			mockErr:  nil,
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &billingv1.StartBillingRequest{
				ReservationId:  "",
				BookingFee:     5000,
				IdempotencyKey: "idem-123",
			},
			wantCode: codes.InvalidArgument,
			wantNil:  true,
		},
		{
			name: "missing idempotency_key",
			req: &billingv1.StartBillingRequest{
				ReservationId:  "res-123",
				BookingFee:     5000,
				IdempotencyKey: "",
			},
			wantCode: codes.InvalidArgument,
			wantNil:  true,
		},
		{
			name: "usecase returns not found",
			req: &billingv1.StartBillingRequest{
				ReservationId:  "res-123",
				BookingFee:     5000,
				IdempotencyKey: "idem-123",
			},
			mockResult: nil,
			mockErr:    repository.ErrNotFound,
			wantCode:   codes.NotFound,
			wantNil:    true,
		},
		{
			name: "usecase returns internal error",
			req: &billingv1.StartBillingRequest{
				ReservationId:  "res-123",
				BookingFee:     5000,
				IdempotencyKey: "idem-123",
			},
			mockResult: nil,
			mockErr:    errors.New("db connection failed"),
			wantCode:   codes.Internal,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" && tt.req.GetIdempotencyKey() != "" {
				uc.On("StartBilling", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.StartBilling(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
				assert.Equal(t, tt.mockResult.ReservationID, resp.GetReservationId())
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

func TestCalculateFee(t *testing.T) {
	now := time.Now()
	checkIn := timestamppb.New(now.Add(-2 * time.Hour))
	checkOut := timestamppb.New(now)

	tests := []struct {
		name       string
		req        *billingv1.CalculateFeeRequest
		mockResult *model.BillingRecord
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &billingv1.CalculateFeeRequest{
				ReservationId: "res-123",
				CheckInAt:     checkIn,
				CheckOutAt:    checkOut,
			},
			mockResult: &model.BillingRecord{
				ID:              "bill-123",
				ReservationID:   "res-123",
				ParkingFee:      10000,
				TotalAmount:     15000,
				DurationMinutes: 120,
				BilledHours:     2,
				Status:          model.BillingStatusCalculated,
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &billingv1.CalculateFeeRequest{
				ReservationId: "",
				CheckInAt:     checkIn,
				CheckOutAt:    checkOut,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing check_in_at",
			req: &billingv1.CalculateFeeRequest{
				ReservationId: "res-123",
				CheckInAt:     nil,
				CheckOutAt:    checkOut,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing check_out_at",
			req: &billingv1.CalculateFeeRequest{
				ReservationId: "res-123",
				CheckInAt:     checkIn,
				CheckOutAt:    nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &billingv1.CalculateFeeRequest{
				ReservationId: "res-123",
				CheckInAt:     checkIn,
				CheckOutAt:    checkOut,
			},
			mockResult: nil,
			mockErr:    repository.ErrNotFound,
			wantCode:   codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" && tt.req.GetCheckInAt() != nil && tt.req.GetCheckOutAt() != nil {
				uc.On("CalculateFee", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.CalculateFee(t.Context(), tt.req)

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

func TestGenerateInvoice(t *testing.T) {
	tests := []struct {
		name       string
		req        *billingv1.GenerateInvoiceRequest
		mockResult *model.BillingRecord
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &billingv1.GenerateInvoiceRequest{
				ReservationId:  "res-123",
				IdempotencyKey: "inv-idem-123",
			},
			mockResult: &model.BillingRecord{
				ID:            "bill-123",
				ReservationID: "res-123",
				Status:        model.BillingStatusInvoiced,
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &billingv1.GenerateInvoiceRequest{
				ReservationId:  "",
				IdempotencyKey: "inv-idem-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing idempotency_key",
			req: &billingv1.GenerateInvoiceRequest{
				ReservationId:  "res-123",
				IdempotencyKey: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &billingv1.GenerateInvoiceRequest{
				ReservationId:  "res-123",
				IdempotencyKey: "inv-idem-123",
			},
			mockResult: nil,
			mockErr:    errors.New("internal error"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetReservationId() != "" && tt.req.GetIdempotencyKey() != "" {
				uc.On("GenerateInvoice", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.GenerateInvoice(t.Context(), tt.req)

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

func TestApplyOvernightFee(t *testing.T) {
	tests := []struct {
		name       string
		req        *billingv1.ApplyOvernightFeeRequest
		mockResult *model.BillingRecord
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &billingv1.ApplyOvernightFeeRequest{
				ReservationId: "res-123",
			},
			mockResult: &model.BillingRecord{
				ID:            "bill-123",
				ReservationID: "res-123",
				OvernightFee:  15000,
				IsOvernight:   true,
			},
			wantCode: codes.OK,
		},
		{
			name: "missing reservation_id",
			req: &billingv1.ApplyOvernightFeeRequest{
				ReservationId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &billingv1.ApplyOvernightFeeRequest{
				ReservationId: "res-123",
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

			if tt.req.GetReservationId() != "" {
				uc.On("ApplyOvernightFee", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.ApplyOvernightFee(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
				assert.True(t, resp.GetIsOvernight())
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
