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

	"parkir-pintar/internal/search/sync"
	"parkir-pintar/pkg/events"
)

// --- Mock SpotSync ---

type MockSpotSync struct {
	mock.Mock
}

func (m *MockSpotSync) HandleSpotUpdated(ctx context.Context, spot sync.SpotData) error {
	args := m.Called(ctx, spot)
	return args.Error(0)
}

// --- Mock RedisCache ---

type MockRedisCache struct {
	mock.Mock
}

func (m *MockRedisCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
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

// --- Mock SpotRepository for constructing real SpotSync ---

type MockSpotRepository struct {
	mock.Mock
}

func (m *MockSpotRepository) UpsertSpot(ctx context.Context, spot sync.SpotData) error {
	args := m.Called(ctx, spot)
	return args.Error(0)
}

func (m *MockSpotRepository) DeleteSpot(ctx context.Context, spotID string) error {
	args := m.Called(ctx, spotID)
	return args.Error(0)
}

// --- Tests ---

func TestHandleSpotUpdated_Success(t *testing.T) {
	repo := new(MockSpotRepository)
	redis := new(MockRedisCache)
	spotSync := sync.NewSpotSync(repo)

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
	repo.On("UpsertSpot", mock.Anything, sync.SpotData{
		ID:          "spot-1",
		FloorNumber: 2,
		SpotNumber:  15,
		VehicleType: "car",
		SpotCode:    "A-15",
		Status:      "available",
	}).Return(nil)
	redis.On("Delete", mock.Anything, mock.Anything).Return(nil)
	msg.On("Ack").Return(nil)

	handler := &Handler{spotSync: spotSync, redis: redis, floorCount: 3}
	handler.handleSpotUpdated(msg)

	repo.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}

func TestHandleSpotUpdated_InvalidJSON(t *testing.T) {
	repo := new(MockSpotRepository)
	redis := new(MockRedisCache)
	spotSync := sync.NewSpotSync(repo)

	msg := &MockMsg{data: []byte("not valid json"), subject: "spot.updated"}
	msg.On("Term").Return(nil)

	handler := &Handler{spotSync: spotSync, redis: redis, floorCount: 3}
	handler.handleSpotUpdated(msg)

	msg.AssertCalled(t, "Term")
	repo.AssertNotCalled(t, "UpsertSpot", mock.Anything, mock.Anything)
}

func TestHandleSpotUpdated_UpsertError(t *testing.T) {
	repo := new(MockSpotRepository)
	redis := new(MockRedisCache)
	spotSync := sync.NewSpotSync(repo)

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
	repo.On("UpsertSpot", mock.Anything, sync.SpotData{
		ID:          "spot-2",
		FloorNumber: 1,
		SpotNumber:  5,
		VehicleType: "motorcycle",
		SpotCode:    "B-05",
		Status:      "reserved",
	}).Return(assert.AnError)
	msg.On("Nak").Return(nil)

	handler := &Handler{spotSync: spotSync, redis: redis, floorCount: 3}
	handler.handleSpotUpdated(msg)

	repo.AssertExpectations(t)
	msg.AssertCalled(t, "Nak")
	msg.AssertNotCalled(t, "Ack")
}

func TestHandleSpotUpdated_AckError(t *testing.T) {
	repo := new(MockSpotRepository)
	redis := new(MockRedisCache)
	spotSync := sync.NewSpotSync(repo)

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
	repo.On("UpsertSpot", mock.Anything, sync.SpotData{
		ID:          "spot-3",
		FloorNumber: 3,
		SpotNumber:  10,
		VehicleType: "car",
		SpotCode:    "C-10",
		Status:      "occupied",
	}).Return(nil)
	redis.On("Delete", mock.Anything, mock.Anything).Return(nil)
	msg.On("Ack").Return(assert.AnError)

	handler := &Handler{spotSync: spotSync, redis: redis, floorCount: 3}
	handler.handleSpotUpdated(msg)

	repo.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
}

func TestHandleSpotUpdated_CacheInvalidationError(t *testing.T) {
	repo := new(MockSpotRepository)
	redis := new(MockRedisCache)
	spotSync := sync.NewSpotSync(repo)

	event := events.SpotUpdatedEvent{
		SpotID:      "spot-4",
		FloorNumber: 1,
		SpotNumber:  1,
		VehicleType: "car",
		SpotCode:    "A-01",
		Status:      "available",
		UpdatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	msg := &MockMsg{data: data, subject: "spot.updated"}
	repo.On("UpsertSpot", mock.Anything, sync.SpotData{
		ID:          "spot-4",
		FloorNumber: 1,
		SpotNumber:  1,
		VehicleType: "car",
		SpotCode:    "A-01",
		Status:      "available",
	}).Return(nil)
	// Cache deletion fails but should not prevent ack
	redis.On("Delete", mock.Anything, mock.Anything).Return(assert.AnError)
	msg.On("Ack").Return(nil)

	handler := &Handler{spotSync: spotSync, redis: redis, floorCount: 2}
	handler.handleSpotUpdated(msg)

	repo.AssertExpectations(t)
	msg.AssertCalled(t, "Ack")
	// Verify cache invalidation was attempted for all expected keys
	// floorCount=2: availability:car, availability:motorcycle, floormap:1, floormap:2
	redis.AssertNumberOfCalls(t, "Delete", 4)
}
