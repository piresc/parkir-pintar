package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_NoEndpoint(t *testing.T) {
	cfg := Config{
		ServiceName:     "test-service",
		OTLPEndpoint:    "",
		TraceSampleRate: 1.0,
		MetricInterval:  10 * time.Second,
	}

	providers, err := Init(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)
	assert.NotNil(t, providers.TracerProvider)
	assert.NotNil(t, providers.MeterProvider)
	assert.NotNil(t, providers.LoggerProvider)
}

func TestInit_NoEndpoint_Shutdown(t *testing.T) {
	cfg := Config{
		ServiceName:  "test-service",
		OTLPEndpoint: "",
	}

	providers, err := Init(context.Background(), cfg)
	require.NoError(t, err)

	err = providers.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestInit_WithEndpoint(t *testing.T) {
	cfg := Config{
		ServiceName:     "test-service",
		OTLPEndpoint:    "localhost:0",
		TraceSampleRate: 0.5,
		MetricInterval:  5 * time.Second,
	}

	providers, err := Init(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)
	assert.NotNil(t, providers.TracerProvider)
	assert.NotNil(t, providers.MeterProvider)
	assert.NotNil(t, providers.LoggerProvider)

	// Shutdown with a short timeout to avoid hanging.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = providers.Shutdown(ctx)
	_ = err
}

func TestInit_DefaultMetricInterval(t *testing.T) {
	cfg := Config{
		ServiceName:     "test-service",
		OTLPEndpoint:    "localhost:0",
		TraceSampleRate: 1.0,
		MetricInterval:  0, // should default
	}

	providers, err := Init(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = providers.Shutdown(ctx)
}

func TestProviders_Shutdown_NilProviders(t *testing.T) {
	p := &Providers{}
	err := p.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestProviders_Shutdown_PartialNil(t *testing.T) {
	cfg := Config{
		ServiceName:  "test-service",
		OTLPEndpoint: "",
	}
	providers, err := Init(context.Background(), cfg)
	require.NoError(t, err)

	providers.LoggerProvider = nil
	err = providers.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestConfig_ZeroValues(t *testing.T) {
	cfg := Config{}
	providers, err := Init(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)

	err = providers.Shutdown(context.Background())
	assert.NoError(t, err)
}
