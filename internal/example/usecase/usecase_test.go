// Package usecase implements the business logic layer for the example domain.
//
// Best practices applied (from coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - Manual mock implementing repository.Repository interface
//   (go.uber.org/mock/mockgen would generate this; manual mock used here
//    as a working example that compiles without code generation)
// - Each test is isolated with its own mock setup
// - Error wrapping is verified via strings.Contains
package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/example/model"
)

// mockRepository is a manual mock of repository.Repository for testing.
type mockRepository struct {
	getByIDFn func(ctx context.Context, id string) (*model.Example, error)
	listFn    func(ctx context.Context, limit, offset int) ([]model.Example, error)
	createFn  func(ctx context.Context, example *model.Example) error
	updateFn  func(ctx context.Context, example *model.Example) error
	deleteFn  func(ctx context.Context, id string) error
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*model.Example, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockRepository) List(ctx context.Context, limit, offset int) ([]model.Example, error) {
	return m.listFn(ctx, limit, offset)
}

func (m *mockRepository) Create(ctx context.Context, example *model.Example) error {
	return m.createFn(ctx, example)
}

func (m *mockRepository) Update(ctx context.Context, example *model.Example) error {
	return m.updateFn(ctx, example)
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func TestCreate_ShouldReturnExample_WhenValidRequest(t *testing.T) {
	// Arrange
	repo := &mockRepository{
		createFn: func(_ context.Context, _ *model.Example) error {
			return nil
		},
	}
	uc := New(repo)
	req := model.CreateExampleRequest{
		Name:        "Test Item",
		Description: "A test item",
	}

	// Act
	result, err := uc.Create(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Test Item", result.Name)
	assert.Equal(t, "A test item", result.Description)
	assert.Equal(t, "active", result.Status)
	assert.NotEmpty(t, result.ID)
	assert.False(t, result.CreatedAt.IsZero())
}

func TestGetByID_ShouldReturnExample_WhenRecordExists(t *testing.T) {
	// Arrange
	now := time.Now()
	expected := &model.Example{
		ID:          "test-id",
		Name:        "Found Item",
		Description: "desc",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*model.Example, error) {
			if id == "test-id" {
				return expected, nil
			}
			return nil, errors.New("not found")
		},
	}
	uc := New(repo)

	// Act
	result, err := uc.GetByID(context.Background(), "test-id")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.Name, result.Name)
}

func TestGetByID_ShouldReturnError_WhenRepositoryFails(t *testing.T) {
	// Arrange
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, _ string) (*model.Example, error) {
			return nil, errors.New("db connection lost")
		},
	}
	uc := New(repo)

	// Act
	result, err := uc.GetByID(context.Background(), "any-id")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "usecase get by id")
}
