// Package usecase implements the business logic layer for the example domain
// module. It depends on the repository interface, not the concrete implementation.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"parkir-pintar/internal/example/model"
	"parkir-pintar/internal/example/repository"
)

// Usecase defines the business logic interface for examples.
//
//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/example/usecase Usecase
type Usecase interface {
	GetByID(ctx context.Context, id string) (*model.Example, error)
	List(ctx context.Context, limit, offset int) ([]model.Example, error)
	Create(ctx context.Context, req model.CreateExampleRequest) (*model.Example, error)
	Update(ctx context.Context, id string, req model.UpdateExampleRequest) (*model.Example, error)
	Delete(ctx context.Context, id string) error
}

// usecase is the concrete implementation of Usecase.
type usecase struct {
	repo repository.Repository
}

// New creates a new Usecase with the given repository dependency.
func New(repo repository.Repository) Usecase {
	return &usecase{repo: repo}
}

// GetByID retrieves a single example by ID.
func (u *usecase) GetByID(ctx context.Context, id string) (*model.Example, error) {
	example, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase get by id: %w", err)
	}
	return example, nil
}

// List retrieves a paginated list of examples.
func (u *usecase) List(ctx context.Context, limit, offset int) ([]model.Example, error) {
	examples, err := u.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("usecase list: %w", err)
	}
	return examples, nil
}

// Create creates a new example from the request payload.
func (u *usecase) Create(ctx context.Context, req model.CreateExampleRequest) (*model.Example, error) {
	now := time.Now()
	example := &model.Example{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := u.repo.Create(ctx, example); err != nil {
		return nil, fmt.Errorf("usecase create: %w", err)
	}
	return example, nil
}

// Update modifies an existing example identified by ID.
func (u *usecase) Update(ctx context.Context, id string, req model.UpdateExampleRequest) (*model.Example, error) {
	existing, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase update get: %w", err)
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Status = req.Status
	existing.UpdatedAt = time.Now()

	if err := u.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("usecase update: %w", err)
	}
	return existing, nil
}

// Delete removes an example by ID.
func (u *usecase) Delete(ctx context.Context, id string) error {
	if err := u.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("usecase delete: %w", err)
	}
	return nil
}
