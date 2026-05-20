package database

import (
	"github.com/jmoiron/sqlx"

	"parkir-pintar/pkg/tracing"
)

// TracedPostgresClient wraps PostgresClient with automatic OTEL tracing.
// The traced wrapper is a thin layer — actual query tracing happens at the
// repository level using tracer.StartDatabase.
type TracedPostgresClient struct {
	*PostgresClient
	tracer tracing.Tracer
}

// NewTracedPostgresClient creates a new traced PostgreSQL client.
func NewTracedPostgresClient(client *PostgresClient, tracer tracing.Tracer) *TracedPostgresClient {
	return &TracedPostgresClient{
		PostgresClient: client,
		tracer:         tracer,
	}
}

// GetDB returns the underlying sqlx.DB instance.
func (t *TracedPostgresClient) GetDB() *sqlx.DB {
	return t.PostgresClient.GetDB()
}

// Close closes the database connection pool.
func (t *TracedPostgresClient) Close() error {
	return t.PostgresClient.Close()
}
