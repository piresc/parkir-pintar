// Package repository provides the data access layer for the example domain
// module using sqlx with parameterized queries for SQL injection prevention.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/example/model"
)

// Repository defines the data access interface for examples.
//
//go:generate mockgen -destination=../mocks/mock_repository.go -package=mocks parkir-pintar/internal/example/repository Repository
type Repository interface {
	GetByID(ctx context.Context, id string) (*model.Example, error)
	List(ctx context.Context, limit, offset int) ([]model.Example, error)
	Create(ctx context.Context, example *model.Example) error
	Update(ctx context.Context, example *model.Example) error
	Delete(ctx context.Context, id string) error
}

// repository is the sqlx-backed implementation of Repository.
type repository struct {
	db *sqlx.DB
}

// New creates a new Repository backed by the given sqlx.DB.
// The cache parameter is reserved for future caching support.
func New(db *sqlx.DB, cache interface{}) Repository {
	return &repository{db: db}
}

// GetByID retrieves a single example by its UUID.
func (r *repository) GetByID(ctx context.Context, id string) (*model.Example, error) {
	var example model.Example
	err := r.db.GetContext(ctx, &example, "SELECT * FROM examples WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", model.ErrNotFound, id)
		}
		return nil, fmt.Errorf("get example by id: %w", err)
	}
	return &example, nil
}

// List retrieves a paginated list of examples ordered by creation date.
func (r *repository) List(ctx context.Context, limit, offset int) ([]model.Example, error) {
	var examples []model.Example
	err := r.db.SelectContext(ctx, &examples,
		"SELECT * FROM examples ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list examples: %w", err)
	}
	return examples, nil
}

// Create inserts a new example into the database.
func (r *repository) Create(ctx context.Context, example *model.Example) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO examples (id, name, description, status, created_at, updated_at)
		 VALUES (:id, :name, :description, :status, :created_at, :updated_at)`,
		example,
	)
	if err != nil {
		return fmt.Errorf("create example: %w", err)
	}
	return nil
}

// Update modifies an existing example in the database.
func (r *repository) Update(ctx context.Context, example *model.Example) error {
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE examples SET name = :name, description = :description,
		 status = :status, updated_at = :updated_at WHERE id = :id`,
		example,
	)
	if err != nil {
		return fmt.Errorf("update example: %w", err)
	}
	return nil
}

// Delete removes an example by its UUID.
func (r *repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM examples WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete example: %w", err)
	}
	return nil
}
