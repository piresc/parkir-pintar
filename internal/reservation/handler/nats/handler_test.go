package nats

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/pkg/events"
)

// --- Mock usecase ---

type MockUsecase struct {
	mock.Mock
}

func (m *MockUsecase) ConfirmReservation(ctx context.Context, req *model.ConfirmReservationRequest) (*model.Reservation, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *MockUsecase) FailReservation(ctx context.Context, req *model.FailReservationRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// --- Mock NATS message ---

type MockMsg struct {
	mock.Mock
	data    []byte
	subject string
}

func (m *MockMsg) Ack() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMsg) Nak() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMsg) NakWithDelay(delay time.Duration) error {
	args := m.Called(delay)
	return args.Error(0)
}

func (m *MockMsg) Term() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMsg) TermWithReason(reason string) error {
	args := m.Called(reason)
	return args.Error(0)
}

func (m *MockMsg) InProgress() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMsg) Metadata() (*jetstream.MsgMetadata, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jetstream.MsgMetadata), args.Error(1)
}

func (m *MockMsg) Data() []byte {
	return m.data
}

func (m *MockMsg) Subject() string {
	return m.subject
}

func (m *MockMsg) Reply() string {
	return ""
}

func (m *MockMsg) Headers() nats.Header {
	return nil
}

func (m *MockMsg) DoubleAck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// --- Tests ---

func TestHandleMessage_PaymentSuccess(t *testing.T) {
	uc := new(MockUsecase)

	event := events.PaymentResultEvent{
		PaymentID:     "pay-123",
		ReservationID: "res-456",
		Amount:        50000,
		Status:        "success",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "payment.result.success"}
	uc.On("ConfirmReservation", mock.Anything, mock.MatchedBy(func(req *model.ConfirmReservationRequest) bool {
		return req.ReservationID == "res-456"
	})).Return(&model.Reservation{ID: "res-456", Status: "confirmed"}, nil)
	msg.On("Ack").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleMessage(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}

func TestHandleMessage_PaymentFailed(t *testing.T) {
	uc := new(MockUsecase)

	event := events.PaymentResultEvent{
		PaymentID:     "pay-789",
		ReservationID: "res-101",
		Amount:        50000,
		Status:        "failed",
		Reason:        "insufficient funds",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "payment.result.failed"}
	uc.On("FailReservation", mock.Anything, mock.MatchedBy(func(req *model.FailReservationRequest) bool {
		return req.ReservationID == "res-101"
	})).Return(nil)
	msg.On("Ack").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleMessage(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	uc := new(MockUsecase)

	msg := &MockMsg{data: []byte("not valid json"), subject: "payment.result.success"}
	msg.On("Nak").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleMessage(msg)

	msg.AssertCalled(t, "Nak")
	uc.AssertNotCalled(t, "ConfirmReservation", mock.Anything, mock.Anything)
	uc.AssertNotCalled(t, "FailReservation", mock.Anything, mock.Anything)
}

func TestHandleMessage_UnknownStatus(t *testing.T) {
	uc := new(MockUsecase)

	event := events.PaymentResultEvent{
		PaymentID:     "pay-000",
		ReservationID: "res-000",
		Status:        "unknown_status",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "payment.result.unknown"}
	msg.On("Ack").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleMessage(msg)

	msg.AssertCalled(t, "Ack")
	uc.AssertNotCalled(t, "ConfirmReservation", mock.Anything, mock.Anything)
	uc.AssertNotCalled(t, "FailReservation", mock.Anything, mock.Anything)
}

func TestHandleMessage_ConfirmReservationError(t *testing.T) {
	uc := new(MockUsecase)

	event := events.PaymentResultEvent{
		PaymentID:     "pay-123",
		ReservationID: "res-456",
		Status:        "success",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "payment.result.success"}
	uc.On("ConfirmReservation", mock.Anything, mock.MatchedBy(func(req *model.ConfirmReservationRequest) bool {
		return req.ReservationID == "res-456"
	})).Return(nil, assert.AnError)
	msg.On("Nak").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleMessage(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Nak")
	msg.AssertNotCalled(t, "Ack")
}

func TestHandleMessage_FailReservationError(t *testing.T) {
	uc := new(MockUsecase)

	event := events.PaymentResultEvent{
		PaymentID:     "pay-789",
		ReservationID: "res-101",
		Status:        "failed",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "payment.result.failed"}
	uc.On("FailReservation", mock.Anything, mock.MatchedBy(func(req *model.FailReservationRequest) bool {
		return req.ReservationID == "res-101"
	})).Return(assert.AnError)
	msg.On("Nak").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleMessage(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Nak")
	msg.AssertNotCalled(t, "Ack")
}

func TestHandleMessage_AckError(t *testing.T) {
	uc := new(MockUsecase)

	event := events.PaymentResultEvent{
		PaymentID:     "pay-123",
		ReservationID: "res-456",
		Status:        "success",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "payment.result.success"}
	uc.On("ConfirmReservation", mock.Anything, mock.MatchedBy(func(req *model.ConfirmReservationRequest) bool {
		return req.ReservationID == "res-456"
	})).Return(&model.Reservation{ID: "res-456"}, nil)
	msg.On("Ack").Return(assert.AnError)

	handler := &Handler{uc: uc}
	handler.handleMessage(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}
