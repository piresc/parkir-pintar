package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgmetrics "parkir-pintar/pkg/metrics"
)

func TestNewParkingMetrics(t *testing.T) {
	base, err := pkgmetrics.NewMetrics("test-service", "")
	require.NoError(t, err)
	defer func() { _ = base.Shutdown(context.Background()) }()

	pm, err := NewParkingMetrics(base, base.Meter())
	require.NoError(t, err)
	require.NotNil(t, pm)

	assert.NotNil(t, pm.ActiveParkingSessions)
	assert.NotNil(t, pm.OccupiedSpots)
	assert.NotNil(t, pm.ReservationsTotal)
}

func TestParkingMetrics_SetActiveParkingSessions_NoPanic(t *testing.T) {
	base, err := pkgmetrics.NewMetrics("test-service", "")
	require.NoError(t, err)
	defer func() { _ = base.Shutdown(context.Background()) }()

	pm, err := NewParkingMetrics(base, base.Meter())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		pm.SetActiveParkingSessions(context.Background(), 42)
	})
}

func TestParkingMetrics_SetOccupiedSpots_NoPanic(t *testing.T) {
	base, err := pkgmetrics.NewMetrics("test-service", "")
	require.NoError(t, err)
	defer func() { _ = base.Shutdown(context.Background()) }()

	pm, err := NewParkingMetrics(base, base.Meter())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		pm.SetOccupiedSpots(context.Background(), 10)
	})
}

func TestParkingMetrics_IncReservations_NoPanic(t *testing.T) {
	base, err := pkgmetrics.NewMetrics("test-service", "")
	require.NoError(t, err)
	defer func() { _ = base.Shutdown(context.Background()) }()

	pm, err := NewParkingMetrics(base, base.Meter())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		pm.IncReservations(context.Background(), "confirmed")
		pm.IncReservations(context.Background(), "cancelled")
	})
}
