// - Don't mock the class under test
// - Don't use real dependencies when mocks are appropriate
package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	paymentconstants "parkir-pintar/internal/payment/constants"
	"parkir-pintar/internal/payment/model"
	"parkir-pintar/internal/payment/repository"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreatePayment(ctx context.Context, payment *model.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.Payment, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockRepository) UpdatePayment(ctx context.Context, payment *model.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockRepository) UpdatePaymentWithStatusCheck(ctx context.Context, payment *model.Payment, expectedStatus string) error {
	args := m.Called(ctx, payment, expectedStatus)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*model.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) Charge(ctx context.Context, amount int64, method string) (string, error) {
	args := m.Called(ctx, amount, method)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) Refund(ctx context.Context, transactionRef string) error {
	args := m.Called(ctx, transactionRef)
	return args.Error(0)
}

func (m *MockPaymentGateway) GetStatus(ctx context.Context, transactionRef string) (string, error) {
	args := m.Called(ctx, transactionRef)
	return args.String(0), args.Error(1)
}

func TestProcessPayment_ShouldReturnExisting_WhenDuplicateIdempotencyKey(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	existing := &model.Payment{
		ID:             "existing-pay-id",
		BillingID:      "billing-1",
		Amount:         5000,
		PaymentMethod:  "qris",
		IdempotencyKey: "pay-key-1",
		Status:         string(paymentconstants.PaymentStatusSuccess),
	}
	repo.On("GetByIdempotencyKey", mock.Anything, "pay-key-1").Return(existing, nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.ProcessPaymentRequest{
		BillingID:      "billing-1",
		Amount:         5000,
		PaymentMethod:  "qris",
		IdempotencyKey: "pay-key-1",
	}

	result, err := uc.ProcessPayment(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, "existing-pay-id", result.ID)
	assert.Equal(t, string(paymentconstants.PaymentStatusSuccess), result.Status)
	gw.AssertNotCalled(t, "Charge")
	repo.AssertNotCalled(t, "CreatePayment")
	repo.AssertExpectations(t)
}

func TestProcessPayment_ShouldReturnSuccess_WhenGatewaySucceeds(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	repo.On("GetByIdempotencyKey", mock.Anything, "pay-key-2").Return(nil, repository.ErrNotFound)
	repo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)
	gw.On("Charge", mock.Anything, int64(10000), "credit_card").Return("txn-abc-123", nil)
	repo.On("UpdatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.ProcessPaymentRequest{
		BillingID:      "billing-2",
		Amount:         10000,
		PaymentMethod:  "credit_card",
		IdempotencyKey: "pay-key-2",
	}

	result, err := uc.ProcessPayment(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, string(paymentconstants.PaymentStatusSuccess), result.Status)
	assert.Equal(t, "txn-abc-123", result.TransactionRef)
	assert.NotNil(t, result.PaidAt)
	assert.Equal(t, int64(10000), result.Amount)
	repo.AssertExpectations(t)
	gw.AssertExpectations(t)
}

func TestProcessPayment_ShouldReturnFailed_WhenGatewayFails(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	gatewayErr := errors.New("gateway timeout")
	repo.On("GetByIdempotencyKey", mock.Anything, "pay-key-3").Return(nil, repository.ErrNotFound)
	repo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)
	gw.On("Charge", mock.Anything, int64(5000), "qris").Return("", gatewayErr)
	repo.On("UpdatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.ProcessPaymentRequest{
		BillingID:      "billing-3",
		Amount:         5000,
		PaymentMethod:  "qris",
		IdempotencyKey: "pay-key-3",
	}

	result, err := uc.ProcessPayment(t.Context(), req)

	require.NoError(t, err) // no error returned; payment is marked failed
	assert.Equal(t, string(paymentconstants.PaymentStatusFailed), result.Status)
	assert.Empty(t, result.TransactionRef)
	assert.Nil(t, result.PaidAt)
	gw.AssertNumberOfCalls(t, "Charge", 3)
	repo.AssertExpectations(t)
}

func TestRefundPayment_ShouldReturnRefunded_WhenSuccessful(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	paidAt := time.Now().Add(-1 * time.Hour)
	existing := &model.Payment{
		ID:             "pay-4",
		BillingID:      "billing-4",
		Amount:         15000,
		PaymentMethod:  "qris",
		TransactionRef: "txn-refund-123",
		Status:         string(paymentconstants.PaymentStatusSuccess),
		PaidAt:         &paidAt,
	}
	repo.On("GetByID", mock.Anything, "pay-4").Return(existing, nil)
	gw.On("Refund", mock.Anything, "txn-refund-123").Return(nil)
	repo.On("UpdatePaymentWithStatusCheck", mock.Anything, mock.AnythingOfType("*model.Payment"), string(paymentconstants.PaymentStatusSuccess)).Return(nil)
	repo.On("UpdatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.RefundPaymentRequest{PaymentID: "pay-4"}

	result, err := uc.RefundPayment(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, string(paymentconstants.PaymentStatusRefunded), result.Status)
	gw.AssertCalled(t, "Refund", mock.Anything, "txn-refund-123")
	repo.AssertExpectations(t)
	gw.AssertExpectations(t)
}

func TestProcessPayment_ShouldRetryAndSucceed_WhenGatewayFailsThenSucceeds(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	gatewayErr := errors.New("transient failure")
	repo.On("GetByIdempotencyKey", mock.Anything, "pay-key-5").Return(nil, repository.ErrNotFound)
	repo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)
	gw.On("Charge", mock.Anything, int64(5000), "ewallet").Return("", gatewayErr).Once()
	gw.On("Charge", mock.Anything, int64(5000), "ewallet").Return("txn-retry-ok", nil).Once()
	repo.On("UpdatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.ProcessPaymentRequest{
		BillingID:      "billing-5",
		Amount:         5000,
		PaymentMethod:  "ewallet",
		IdempotencyKey: "pay-key-5",
	}

	result, err := uc.ProcessPayment(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, string(paymentconstants.PaymentStatusSuccess), result.Status)
	assert.Equal(t, "txn-retry-ok", result.TransactionRef)
	gw.AssertNumberOfCalls(t, "Charge", 2)
	repo.AssertExpectations(t)
	gw.AssertExpectations(t)
}

func TestGetPaymentStatus_ShouldReturnPayment_WhenExists(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	existing := &model.Payment{
		ID:        "pay-6",
		BillingID: "billing-6",
		Amount:    20000,
		Status:    string(paymentconstants.PaymentStatusSuccess),
	}
	repo.On("GetByID", mock.Anything, "pay-6").Return(existing, nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.GetPaymentStatusRequest{PaymentID: "pay-6"}

	result, err := uc.GetPaymentStatus(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, "pay-6", result.ID)
	assert.Equal(t, string(paymentconstants.PaymentStatusSuccess), result.Status)
	repo.AssertExpectations(t)
}

func TestRefundPayment_ShouldSucceed_WhenIdempotencyKeyNotFound(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	paidAt := time.Now().Add(-1 * time.Hour)
	existing := &model.Payment{
		ID:             "pay-first-refund",
		BillingID:      "bill-first",
		Amount:         15000,
		PaymentMethod:  "qris",
		TransactionRef: "txn-first-refund",
		Status:         string(paymentconstants.PaymentStatusSuccess),
		PaidAt:         &paidAt,
	}
	repo.On("GetByIdempotencyKey", mock.Anything, "refund-key-new").Return(nil, repository.ErrNotFound)
	repo.On("GetByID", mock.Anything, "pay-first-refund").Return(existing, nil)
	gw.On("Refund", mock.Anything, "txn-first-refund").Return(nil)
	repo.On("UpdatePaymentWithStatusCheck", mock.Anything, mock.AnythingOfType("*model.Payment"), string(paymentconstants.PaymentStatusSuccess)).Return(nil)
	repo.On("UpdatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.RefundPaymentRequest{
		PaymentID:      "pay-first-refund",
		IdempotencyKey: "refund-key-new",
	}

	result, err := uc.RefundPayment(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, string(paymentconstants.PaymentStatusRefunded), result.Status)
	gw.AssertCalled(t, "Refund", mock.Anything, "txn-first-refund")
	repo.AssertExpectations(t)
}

func TestRefundPayment_ShouldReturnExisting_WhenIdempotencyKeyExists(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	existing := &model.Payment{
		ID:             "existing-refund-id",
		BillingID:      "bill-1",
		Amount:         15000,
		Status:         string(paymentconstants.PaymentStatusRefunded),
		IdempotencyKey: "refund-key-1",
	}
	repo.On("GetByIdempotencyKey", mock.Anything, "refund-key-1").Return(existing, nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.RefundPaymentRequest{
		PaymentID:      "different-payment",
		IdempotencyKey: "refund-key-1",
	}

	result, err := uc.RefundPayment(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, "existing-refund-id", result.ID)
	assert.Equal(t, string(paymentconstants.PaymentStatusRefunded), result.Status)
	gw.AssertNotCalled(t, "Refund")
	repo.AssertExpectations(t)
}
