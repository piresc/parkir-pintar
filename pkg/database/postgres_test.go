package database

import (
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/tracing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresClient_ShouldReturnError_WhenConnectionFails(t *testing.T) {
	// Arrange: invalid config that cannot connect
	cfg := config.DatabaseConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     5432,
		Username: "test",
		Password: "test",
		Database: "testdb",
		SSLMode:  "disable",
	}

	client, err := NewPostgresClient(cfg)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to ping postgres")
}

func TestNewTracedPostgresClient_ShouldReturnValidClient_WhenCreated(t *testing.T) {
	tracer := tracing.NewNoOpTracer()

	// We can't create a real PostgresClient without a DB, so test the constructor
	// with a nil-safe approach using the traced wrapper constructor directly
	pgClient := &PostgresClient{db: nil}
	traced := NewTracedPostgresClient(pgClient, tracer)

	require.NotNil(t, traced)
	assert.Equal(t, pgClient, traced.PostgresClient)
}

func TestTracedPostgresClient_GetDB_ShouldDelegateToUnderlying(t *testing.T) {
	pgClient := &PostgresClient{db: nil}
	traced := NewTracedPostgresClient(pgClient, tracing.NewNoOpTracer())

	assert.Nil(t, traced.GetDB())
}
