package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSpotRepo struct {
	mock.Mock
}

func (m *mockSpotRepo) UpsertSpot(ctx context.Context, spot SpotData) error {
	args := m.Called(ctx, spot)
	return args.Error(0)
}

func (m *mockSpotRepo) DeleteSpot(ctx context.Context, spotID string) error {
	args := m.Called(ctx, spotID)
	return args.Error(0)
}

func TestSpotSync_HandleSpotUpdated_ShouldUpsertSpot(t *testing.T) {
	repo := new(mockSpotRepo)
	syncer := NewSpotSync(repo)

	spot := SpotData{
		ID:          "spot-1",
		FloorNumber: 1,
		SpotNumber:  5,
		VehicleType: "car",
		SpotCode:    "F1-C-005",
		Status:      "reserved",
	}

	repo.On("UpsertSpot", mock.Anything, spot).Return(nil)

	err := syncer.HandleSpotUpdated(t.Context(), spot)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSpotSync_HandleSpotUpdated_ShouldReturnError_WhenUpsertFails(t *testing.T) {
	repo := new(mockSpotRepo)
	syncer := NewSpotSync(repo)

	spot := SpotData{
		ID:     "spot-1",
		Status: "reserved",
	}

	repo.On("UpsertSpot", mock.Anything, spot).Return(assert.AnError)

	err := syncer.HandleSpotUpdated(t.Context(), spot)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestSpotSync_HandleNATSEvent_ShouldUpsertSpot(t *testing.T) {
	repo := new(mockSpotRepo)
	syncer := NewSpotSync(repo)

	data := []byte(`{"id":"spot-2","floor_number":3,"spot_number":10,"vehicle_type":"motorcycle","spot_code":"F3-M-010","status":"available"}`)
	expectedSpot := SpotData{
		ID:          "spot-2",
		FloorNumber: 3,
		SpotNumber:  10,
		VehicleType: "motorcycle",
		SpotCode:    "F3-M-010",
		Status:      "available",
	}

	repo.On("UpsertSpot", mock.Anything, expectedSpot).Return(nil)

	syncer.HandleNATSEvent(t.Context(), "spot.updated", data)
	repo.AssertExpectations(t)
}

func TestSpotSync_HandleNATSEvent_ShouldLogWarning_WhenUnmarshalFails(t *testing.T) {
	repo := new(mockSpotRepo)
	syncer := NewSpotSync(repo)

	syncer.HandleNATSEvent(t.Context(), "spot.updated", []byte("invalid json"))

	repo.AssertNotCalled(t, "UpsertSpot")
}
