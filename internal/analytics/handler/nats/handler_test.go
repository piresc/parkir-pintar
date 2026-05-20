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

	"parkir-pintar/internal/analytics/model"
)

// --- Mock usecase ---

type MockUsecase struct {
	mock.Mock
}

func (m *MockUsecase) GetPeakHours(ctx context.Context) ([]model.PeakHourStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.PeakHourStats), args.Error(1)
}

func (m *MockUsecase) GetIdleHours(ctx context.Context) ([]model.PeakHourStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.PeakHourStats), args.Error(1)
}

func (m *MockUsecase) PredictResources(ctx context.Context, horizon time.Duration) ([]model.ResourcePrediction, error) {
	args := m.Called(ctx, horizon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ResourcePrediction), args.Error(1)
}

func (m *MockUsecase) GetUsagePatterns(ctx context.Context) (*model.UsagePattern, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UsagePattern), args.Error(1)
}

func (m *MockUsecase) RecordEvent(ctx context.Context, event model.ReservationEvent) error {
	args := m.Called(ctx, event)
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

func TestHandleReservationEvent_Success(t *testing.T) {
	uc := new(MockUsecase)

	event := model.ReservationEvent{
		ReservationID: "res-123",
		DriverID:      "driver-1",
		SpotID:        "spot-1",
		VehicleType:   "car",
		Status:        "confirmed",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "reservation.analytics.confirmed"}
	uc.On("RecordEvent", mock.Anything, mock.MatchedBy(func(e model.ReservationEvent) bool {
		return e.ReservationID == "res-123" && e.Status == "confirmed"
	})).Return(nil)
	msg.On("Ack").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleReservationEvent(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}

func TestHandleReservationEvent_InvalidJSON(t *testing.T) {
	uc := new(MockUsecase)

	msg := &MockMsg{data: []byte("invalid json"), subject: "reservation.analytics.confirmed"}
	msg.On("Term").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleReservationEvent(msg)

	msg.AssertCalled(t, "Term")
	uc.AssertNotCalled(t, "RecordEvent", mock.Anything, mock.Anything)
}

func TestHandleReservationEvent_UsecaseError(t *testing.T) {
	uc := new(MockUsecase)

	event := model.ReservationEvent{
		ReservationID: "res-123",
		DriverID:      "driver-1",
		SpotID:        "spot-1",
		VehicleType:   "car",
		Status:        "confirmed",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "reservation.analytics.confirmed"}
	uc.On("RecordEvent", mock.Anything, mock.MatchedBy(func(e model.ReservationEvent) bool {
		return e.ReservationID == "res-123"
	})).Return(assert.AnError)
	msg.On("Nak").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleReservationEvent(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Nak")
	msg.AssertNotCalled(t, "Ack")
}

func TestHandleReservationEvent_AckError(t *testing.T) {
	uc := new(MockUsecase)

	event := model.ReservationEvent{
		ReservationID: "res-123",
		DriverID:      "driver-1",
		SpotID:        "spot-1",
		VehicleType:   "car",
		Status:        "confirmed",
		Timestamp:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "reservation.analytics.confirmed"}
	uc.On("RecordEvent", mock.Anything, mock.MatchedBy(func(e model.ReservationEvent) bool {
		return e.ReservationID == "res-123"
	})).Return(nil)
	msg.On("Ack").Return(assert.AnError)

	handler := &Handler{uc: uc}
	handler.handleReservationEvent(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}
