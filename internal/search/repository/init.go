// module using sqlx with parameterized queries for SQL injection prevention.
// - Use parameterized queries to prevent SQL injection
package repository

import (
	"github.com/jmoiron/sqlx"
)

type sqlxRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

type sqlxReadModelRepository struct {
	db *sqlx.DB
}

func NewReadModelRepository(db *sqlx.DB) ReadModelRepository {
	return &sqlxReadModelRepository{db: db}
}
