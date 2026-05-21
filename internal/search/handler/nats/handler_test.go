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

	"parkir-pintar/internal/events"
	"parkir-pintar/internal/search"
	"parkir-pintar/internal/search/model"
)

// --- Mock Usecase ---

type MockUsecase struct {
	mock.Mock
}

func (m *MockUsecase) GetAvailability(ctx context.Context, req *model.GetAvailabilityRequest) ([]model.FloorAvailability, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FloorAvailability), args.Error(1)
}

func (m *MockUsecase) GetFloorMap(ctx context.Context, req *model.GetFloorMapRequest) ([]model.SpotDetails, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.SpotDetails), args.Error(1)
}

func (m *MockUsecase) GetSpotDetails(ctx context.Context, req *model.GetSpotDetailsRequest) (*model.SpotDetails, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SpotDetails), args.Error(1)
}

func (m *MockUsecase) HandleSpotUpdated(ctx context.Context, spot search.SpotData) error {
	args := m.Called(ctx, spot)
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

func TestHandleSpotUpdated_Success(t *testing.T) {
	uc := new(MockUsecase)

	event := events.SpotUpdatedEvent{
		SpotID:      "spot-1",
		FloorNumber: 2,
		SpotNumber:  15,
		VehicleType: "car",
		SpotCode:    "A-15",
		Status:      "available",
		UpdatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "spot.updated"}
	uc.On("HandleSpotUpdated", mock.Anything, search.SpotData{
		ID:          "spot-1",
		FloorNumber: 2,
		SpotNumber:  15,
		VehicleType: "car",
		SpotCode:    "A-15",
		Status:      "available",
	}).Return(nil)
	msg.On("Ack").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleSpotUpdated(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}

func TestHandleSpotUpdated_InvalidJSON(t *testing.T) {
	uc := new(MockUsecase)

	msg := &MockMsg{data: []byte("not valid json"), subject: "spot.updated"}
	msg.On("Term").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleSpotUpdated(msg)

	msg.AssertCalled(t, "Term")
	uc.AssertNotCalled(t, "HandleSpotUpdated", mock.Anything, mock.Anything)
}

func TestHandleSpotUpdated_UsecaseError(t *testing.T) {
	uc := new(MockUsecase)

	event := events.SpotUpdatedEvent{
		SpotID:      "spot-2",
		FloorNumber: 1,
		SpotNumber:  5,
		VehicleType: "motorcycle",
		SpotCode:    "B-05",
		Status:      "reserved",
		UpdatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "spot.updated"}
	uc.On("HandleSpotUpdated", mock.Anything, search.SpotData{
		ID:          "spot-2",
		FloorNumber: 1,
		SpotNumber:  5,
		VehicleType: "motorcycle",
		SpotCode:    "B-05",
		Status:      "reserved",
	}).Return(assert.AnError)
	msg.On("Nak").Return(nil)

	handler := &Handler{uc: uc}
	handler.handleSpotUpdated(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Nak")
	msg.AssertNotCalled(t, "Ack")
}

func TestHandleSpotUpdated_AckError(t *testing.T) {
	uc := new(MockUsecase)

	event := events.SpotUpdatedEvent{
		SpotID:      "spot-3",
		FloorNumber: 3,
		SpotNumber:  10,
		VehicleType: "car",
		SpotCode:    "C-10",
		Status:      "occupied",
		UpdatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "spot.updated"}
	uc.On("HandleSpotUpdated", mock.Anything, search.SpotData{
		ID:          "spot-3",
		FloorNumber: 3,
		SpotNumber:  10,
		VehicleType: "car",
		SpotCode:    "C-10",
		Status:      "occupied",
	}).Return(nil)
	msg.On("Ack").Return(assert.AnError)

	handler := &Handler{uc: uc}
	handler.handleSpotUpdated(msg)

	uc.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}
