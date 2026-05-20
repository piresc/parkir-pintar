package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStubSensorGateway_ShouldReturnOccupied_WhenOccupiedResultTrue(t *testing.T) {
	gw := &StubSensorGateway{OccupiedResult: true}

	reading, err := gw.CheckSpotOccupancy(context.Background(), 1, 5)

	require.NoError(t, err)
	assert.True(t, reading.Occupied)
	assert.False(t, reading.DetectedAt.IsZero())
}

func TestStubSensorGateway_ShouldReturnNotOccupied_WhenOccupiedResultFalse(t *testing.T) {
	gw := &StubSensorGateway{OccupiedResult: false}

	reading, err := gw.CheckSpotOccupancy(context.Background(), 2, 10)

	require.NoError(t, err)
	assert.False(t, reading.Occupied)
}

func TestStubSensorGateway_ShouldReturnError_WhenErrResultSet(t *testing.T) {
	expectedErr := errors.New("sensor offline")
	gw := &StubSensorGateway{ErrResult: expectedErr}

	reading, err := gw.CheckSpotOccupancy(context.Background(), 1, 1)

	assert.Nil(t, reading)
	assert.ErrorIs(t, err, expectedErr)
}

func TestNewStubSensorGateway_ShouldDefaultToOccupied(t *testing.T) {
	gw := NewStubSensorGateway()

	assert.True(t, gw.OccupiedResult)
	assert.Nil(t, gw.ErrResult)
}
