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
