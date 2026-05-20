package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentconstants "parkir-pintar/internal/payment/constants"
	"parkir-pintar/internal/payment/model"
	"parkir-pintar/internal/payment/repository"
	paymentv1 "parkir-pintar/proto/payment/v1"
)

type mockUsecase struct {
	mock.Mock
}

func (m *mockUsecase) ProcessPayment(ctx context.Context, req *model.ProcessPaymentRequest) (*model.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *mockUsecase) ProcessQRIS(ctx context.Context, req *model.ProcessQRISRequest) (*model.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *mockUsecase) RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *mockUsecase) GetPaymentStatus(ctx context.Context, req *model.GetPaymentStatusRequest) (*model.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func TestProcessPayment(t *testing.T) {
	tests := []struct {
		name       string
		req        *paymentv1.ProcessPaymentRequest
		mockResult *model.Payment
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &paymentv1.ProcessPaymentRequest{
				BillingId:      "bill-123",
				Amount:         50000,
				PaymentMethod:  "qris",
				IdempotencyKey: "idem-pay-123",
			},
			mockResult: &model.Payment{
				ID:             "pay-123",
				BillingID:      "bill-123",
				Amount:         50000,
				PaymentMethod:  "qris",
				PaymentGateway: "stub-gateway",
				IdempotencyKey: "idem-pay-123",
				Status:         string(paymentconstants.PaymentStatusSuccess),
			},
			wantCode: codes.OK,
		},
		{
			name: "missing billing_id",
			req: &paymentv1.ProcessPaymentRequest{
				BillingId:      "",
				Amount:         50000,
				PaymentMethod:  "qris",
				IdempotencyKey: "idem-pay-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid amount zero",
			req: &paymentv1.ProcessPaymentRequest{
				BillingId:      "bill-123",
				Amount:         0,
				PaymentMethod:  "qris",
				IdempotencyKey: "idem-pay-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid amount negative",
			req: &paymentv1.ProcessPaymentRequest{
				BillingId:      "bill-123",
				Amount:         -100,
				PaymentMethod:  "qris",
				IdempotencyKey: "idem-pay-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing payment_method",
			req: &paymentv1.ProcessPaymentRequest{
				BillingId:      "bill-123",
				Amount:         50000,
				PaymentMethod:  "",
				IdempotencyKey: "idem-pay-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing idempotency_key",
			req: &paymentv1.ProcessPaymentRequest{
				BillingId:      "bill-123",
				Amount:         50000,
				PaymentMethod:  "qris",
				IdempotencyKey: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns not found",
			req: &paymentv1.ProcessPaymentRequest{
				BillingId:      "bill-123",
				Amount:         50000,
				PaymentMethod:  "qris",
				IdempotencyKey: "idem-pay-123",
			},
			mockResult: nil,
			mockErr:    repository.ErrNotFound,
			wantCode:   codes.NotFound,
		},
		{
			name: "usecase returns internal error",
			req: &paymentv1.ProcessPaymentRequest{
				BillingId:      "bill-123",
				Amount:         50000,
				PaymentMethod:  "qris",
				IdempotencyKey: "idem-pay-123",
			},
			mockResult: nil,
			mockErr:    errors.New("gateway timeout"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetBillingId() != "" && tt.req.GetAmount() > 0 && tt.req.GetPaymentMethod() != "" && tt.req.GetIdempotencyKey() != "" {
				uc.On("ProcessPayment", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.ProcessPayment(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
				assert.Equal(t, tt.mockResult.BillingID, resp.GetBillingId())
				assert.Equal(t, tt.mockResult.Amount, resp.GetAmount())
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

func TestProcessQRIS(t *testing.T) {
	tests := []struct {
		name       string
		req        *paymentv1.ProcessQRISRequest
		mockResult *model.Payment
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &paymentv1.ProcessQRISRequest{
				BillingId:      "bill-123",
				Amount:         25000,
				IdempotencyKey: "qris-idem-123",
			},
			mockResult: &model.Payment{
				ID:             "pay-456",
				BillingID:      "bill-123",
				Amount:         25000,
				PaymentMethod:  "qris",
				PaymentGateway: "stub-gateway",
				IdempotencyKey: "qris-idem-123",
				Status:         string(paymentconstants.PaymentStatusSuccess),
			},
			wantCode: codes.OK,
		},
		{
			name: "missing billing_id",
			req: &paymentv1.ProcessQRISRequest{
				BillingId:      "",
				Amount:         25000,
				IdempotencyKey: "qris-idem-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid amount",
			req: &paymentv1.ProcessQRISRequest{
				BillingId:      "bill-123",
				Amount:         0,
				IdempotencyKey: "qris-idem-123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing idempotency_key",
			req: &paymentv1.ProcessQRISRequest{
				BillingId:      "bill-123",
				Amount:         25000,
				IdempotencyKey: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &paymentv1.ProcessQRISRequest{
				BillingId:      "bill-123",
				Amount:         25000,
				IdempotencyKey: "qris-idem-123",
			},
			mockResult: nil,
			mockErr:    errors.New("internal failure"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetBillingId() != "" && tt.req.GetAmount() > 0 && tt.req.GetIdempotencyKey() != "" {
				uc.On("ProcessQRIS", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.ProcessQRIS(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
				assert.Equal(t, tt.mockResult.Amount, resp.GetAmount())
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

func TestRefundPayment(t *testing.T) {
	tests := []struct {
		name       string
		req        *paymentv1.RefundPaymentRequest
		mockResult *model.Payment
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &paymentv1.RefundPaymentRequest{
				PaymentId: "pay-123",
			},
			mockResult: &model.Payment{
				ID:     "pay-123",
				Status: string(paymentconstants.PaymentStatusRefunded),
			},
			wantCode: codes.OK,
		},
		{
			name: "missing payment_id",
			req: &paymentv1.RefundPaymentRequest{
				PaymentId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns not found",
			req: &paymentv1.RefundPaymentRequest{
				PaymentId: "pay-999",
			},
			mockResult: nil,
			mockErr:    repository.ErrNotFound,
			wantCode:   codes.NotFound,
		},
		{
			name: "usecase returns internal error",
			req: &paymentv1.RefundPaymentRequest{
				PaymentId: "pay-123",
			},
			mockResult: nil,
			mockErr:    errors.New("refund gateway error"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetPaymentId() != "" {
				uc.On("RefundPayment", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.RefundPayment(t.Context(), tt.req)

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

func TestGetPaymentStatus(t *testing.T) {
	tests := []struct {
		name       string
		req        *paymentv1.GetPaymentStatusRequest
		mockResult *model.Payment
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &paymentv1.GetPaymentStatusRequest{
				PaymentId: "pay-123",
			},
			mockResult: &model.Payment{
				ID:            "pay-123",
				BillingID:     "bill-123",
				Amount:        50000,
				PaymentMethod: "qris",
				Status:        string(paymentconstants.PaymentStatusSuccess),
			},
			wantCode: codes.OK,
		},
		{
			name: "missing payment_id",
			req: &paymentv1.GetPaymentStatusRequest{
				PaymentId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns not found",
			req: &paymentv1.GetPaymentStatusRequest{
				PaymentId: "pay-999",
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

			if tt.req.GetPaymentId() != "" {
				uc.On("GetPaymentStatus", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.GetPaymentStatus(t.Context(), tt.req)

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
