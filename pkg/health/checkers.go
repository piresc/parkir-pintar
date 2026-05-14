package health

import (
	"context"
	"fmt"

	"parkir-pintar/pkg/database"
	"parkir-pintar/pkg/redis"
)

// Checker name constants.
const (
	CheckerNamePostgres = "postgres"
	CheckerNameRedis    = "redis"
)

// postgresChecker verifies PostgreSQL connectivity via ping.
type postgresChecker struct {
	db *database.PostgresClient
}

// NewPostgresChecker creates a health checker for PostgreSQL.
func NewPostgresChecker(db *database.PostgresClient) Checker {
	return &postgresChecker{db: db}
}

func (c *postgresChecker) Check(ctx context.Context) error {
	if err := c.db.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping failed: %w", err)
	}
	return nil
}

func (c *postgresChecker) Name() string {
	return CheckerNamePostgres
}

// redisChecker verifies Redis connectivity via ping.
type redisChecker struct {
	redis *redis.Client
}

// NewRedisChecker creates a health checker for Redis.
func NewRedisChecker(rc *redis.Client) Checker {
	return &redisChecker{redis: rc}
}

func (c *redisChecker) Check(ctx context.Context) error {
	if _, err := c.redis.GetClient().Ping(ctx).Result(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

func (c *redisChecker) Name() string {
	return CheckerNameRedis
}
