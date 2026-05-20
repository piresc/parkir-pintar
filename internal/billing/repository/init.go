// module using sqlx with parameterized queries for SQL injection prevention.
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
