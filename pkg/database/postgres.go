// Package database provides PostgreSQL connection management with sqlx,
// connection pooling, and traced wrapper for automatic OTEL span creation.
package database

import (
	"context"
	"fmt"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/jmoiron/sqlx"

	"parkir-pintar/pkg/config"
)

// PostgresClient represents a PostgreSQL database client with connection pooling.
type PostgresClient struct {
	db *sqlx.DB
}

// NewPostgresClient creates a new PostgreSQL client with the given configuration.
// It builds a DSN from config, sets connection pool parameters, and verifies
// connectivity with a 10-second timeout ping.
func NewPostgresClient(cfg config.DatabaseConfig) (*PostgresClient, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.PathEscape(cfg.Username),
		url.PathEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	if cfg.Schema != "" && cfg.Schema != "public" {
		dsn += fmt.Sprintf("&search_path=%s,public", url.QueryEscape(cfg.Schema))
	}

	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Configure connection pool
	if cfg.MaxConns > 0 {
		db.SetMaxOpenConns(cfg.MaxConns)
	}
	if cfg.IdleConns > 0 {
		db.SetMaxIdleConns(cfg.IdleConns)
	}
	if cfg.MaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Minute)
	}

	// Verify connection with 10s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	if cfg.Schema != "" && cfg.Schema != "public" {
		_, err := db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", cfg.Schema))
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set search_path to %s: %w", cfg.Schema, err)
		}
	}

	return &PostgresClient{db: db}, nil
}

// GetDB returns the underlying sqlx.DB instance for direct query access.
func (p *PostgresClient) GetDB() *sqlx.DB {
	return p.db
}

// Close closes the database connection pool.
func (p *PostgresClient) Close() error {
	return p.db.Close()
}

// Ping verifies the database connection is still alive.
func (p *PostgresClient) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
