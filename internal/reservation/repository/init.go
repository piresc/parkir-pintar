// Package repository provides the data access layer for the reservation domain
// module using sqlx with parameterized queries for SQL injection prevention.
package repository

import (
	"github.com/jmoiron/sqlx"
)

// sqlxRepository is the sqlx-backed implementation of Repository.
type sqlxRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new Repository backed by the given sqlx.DB.
func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}
