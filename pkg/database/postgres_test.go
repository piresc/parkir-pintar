// Best practices applied from coding standards knowledge base:
// - Go testify patterns: table-driven tests, descriptive test names (Test[Function]_Should[Result]_When[Condition])
// - Mock at the right level: test constructor validation and traced wrapper delegation
// - Test connection handling: include tests for connection failures and timeouts
// - Error handling: verify wrapped errors with context
// - Use t.Helper() for common setup, test error conditions as thoroughly as success cases

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

	// Act
	client, err := NewPostgresClient(cfg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to ping postgres")
}

func TestNewTracedPostgresClient_ShouldReturnValidClient_WhenCreated(t *testing.T) {
	// Arrange
	tracer := tracing.NewNoOpTracer()

	// We can't create a real PostgresClient without a DB, so test the constructor
	// with a nil-safe approach using the traced wrapper constructor directly
	pgClient := &PostgresClient{db: nil}
	traced := NewTracedPostgresClient(pgClient, tracer)

	// Assert
	require.NotNil(t, traced)
	assert.Equal(t, pgClient, traced.PostgresClient)
}

func TestTracedPostgresClient_GetDB_ShouldDelegateToUnderlying(t *testing.T) {
	// Arrange
	pgClient := &PostgresClient{db: nil}
	traced := NewTracedPostgresClient(pgClient, tracing.NewNoOpTracer())

	// Act & Assert: GetDB delegates to underlying client
	assert.Nil(t, traced.GetDB())
}


