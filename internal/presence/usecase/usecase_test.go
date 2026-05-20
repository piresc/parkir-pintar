package usecase

import (
	"context"
	"errors"
	"testing"

	"parkir-pintar/internal/presence/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyPresence_ShouldReturnVerified_WhenSpotIsOccupied(t *testing.T) {
	stub := &repository.StubSensorGateway{OccupiedResult: true}
	uc := NewUsecase(stub)

	result, err := uc.VerifyPresence(context.Background(), "res-123", 1, 5)

	require.NoError(t, err)
	assert.True(t, result.Verified)
	assert.Equal(t, "spot occupied, presence confirmed", result.Message)
}

func TestVerifyPresence_ShouldReturnNotVerified_WhenSpotIsNotOccupied(t *testing.T) {
	stub := &repository.StubSensorGateway{OccupiedResult: false}
	uc := NewUsecase(stub)

	result, err := uc.VerifyPresence(context.Background(), "res-456", 2, 10)

	require.NoError(t, err)
	assert.False(t, result.Verified)
	assert.Equal(t, "spot not occupied, driver may be at wrong spot", result.Message)
}

func TestVerifyPresence_ShouldAssumePresence_WhenSensorFails(t *testing.T) {
	stub := &repository.StubSensorGateway{ErrResult: errors.New("sensor timeout")}
	uc := NewUsecase(stub)

	result, err := uc.VerifyPresence(context.Background(), "res-789", 3, 1)

	require.NoError(t, err)
	assert.True(t, result.Verified)
	assert.Equal(t, "sensor unavailable, presence assumed", result.Message)
}

func TestNewUsecase_ShouldReturnUsecaseInstance(t *testing.T) {
	stub := repository.NewStubSensorGateway()
	uc := NewUsecase(stub)

	assert.NotNil(t, uc)
}
