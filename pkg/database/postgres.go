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

	// Configure connection pool with safe defaults
	if cfg.MaxConns > 0 {
		db.SetMaxOpenConns(cfg.MaxConns)
	} else {
		db.SetMaxOpenConns(25)
	}
	if cfg.IdleConns > 0 {
		db.SetMaxIdleConns(cfg.IdleConns)
	} else {
		db.SetMaxIdleConns(5)
	}
	if cfg.MaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Minute)
	} else {
		db.SetConnMaxLifetime(30 * time.Minute)
	}
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Verify connection with 10s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
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
