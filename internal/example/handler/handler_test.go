package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/example/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockUsecase implements usecase.Usecase for handler tests.
type mockUsecase struct {
	getByIDFn func(ctx context.Context, id string) (*model.Example, error)
	listFn    func(ctx context.Context, limit, offset int) ([]model.Example, error)
	createFn  func(ctx context.Context, req model.CreateExampleRequest) (*model.Example, error)
	updateFn  func(ctx context.Context, id string, req model.UpdateExampleRequest) (*model.Example, error)
	deleteFn  func(ctx context.Context, id string) error
}

func (m *mockUsecase) GetByID(ctx context.Context, id string) (*model.Example, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockUsecase) List(ctx context.Context, limit, offset int) ([]model.Example, error) {
	return m.listFn(ctx, limit, offset)
}
func (m *mockUsecase) Create(ctx context.Context, req model.CreateExampleRequest) (*model.Example, error) {
	return m.createFn(ctx, req)
}
func (m *mockUsecase) Update(ctx context.Context, id string, req model.UpdateExampleRequest) (*model.Example, error) {
	return m.updateFn(ctx, id, req)
}
func (m *mockUsecase) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func TestGet_ShouldReturn200_WhenExampleExists(t *testing.T) {
	// Arrange
	now := time.Now()
	uc := &mockUsecase{
		getByIDFn: func(_ context.Context, id string) (*model.Example, error) {
			return &model.Example{ID: id, Name: "Test", Status: "active", CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.GET("/examples/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/examples/abc-123", nil)

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "success", body["status"])
}

func TestGet_ShouldReturn404_WhenNotFound(t *testing.T) {
	// Arrange
	uc := &mockUsecase{
		getByIDFn: func(_ context.Context, _ string) (*model.Example, error) {
			return nil, model.ErrNotFound
		},
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.GET("/examples/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/examples/missing", nil)

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGet_ShouldReturn500_WhenInternalError(t *testing.T) {
	// Arrange
	uc := &mockUsecase{
		getByIDFn: func(_ context.Context, _ string) (*model.Example, error) {
			return nil, errors.New("db connection lost")
		},
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.GET("/examples/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/examples/abc", nil)

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestList_ShouldReturn200_WhenExamplesExist(t *testing.T) {
	// Arrange
	now := time.Now()
	uc := &mockUsecase{
		listFn: func(_ context.Context, _, _ int) ([]model.Example, error) {
			return []model.Example{
				{ID: "1", Name: "First", Status: "active", CreatedAt: now, UpdatedAt: now},
				{ID: "2", Name: "Second", Status: "active", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.GET("/examples", h.List)

	req := httptest.NewRequest(http.MethodGet, "/examples?limit=10&offset=0", nil)

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_ShouldReturn500_WhenUsecaseFails(t *testing.T) {
	// Arrange
	uc := &mockUsecase{
		listFn: func(_ context.Context, _, _ int) ([]model.Example, error) {
			return nil, errors.New("db error")
		},
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.GET("/examples", h.List)

	req := httptest.NewRequest(http.MethodGet, "/examples", nil)

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreate_ShouldReturn201_WhenValidRequest(t *testing.T) {
	// Arrange
	now := time.Now()
	uc := &mockUsecase{
		createFn: func(_ context.Context, req model.CreateExampleRequest) (*model.Example, error) {
			return &model.Example{ID: "new-id", Name: req.Name, Description: req.Description, Status: "active", CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.POST("/examples", h.Create)

	body := `{"name":"Test","description":"A test"}`
	req := httptest.NewRequest(http.MethodPost, "/examples", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreate_ShouldReturn400_WhenInvalidJSON(t *testing.T) {
	// Arrange
	uc := &mockUsecase{}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.POST("/examples", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/examples", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_ShouldReturn200_WhenValidRequest(t *testing.T) {
	// Arrange
	now := time.Now()
	uc := &mockUsecase{
		updateFn: func(_ context.Context, id string, req model.UpdateExampleRequest) (*model.Example, error) {
			return &model.Example{ID: id, Name: req.Name, Status: req.Status, CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.PUT("/examples/:id", h.Update)

	body := `{"name":"Updated","description":"Updated desc","status":"inactive"}`
	req := httptest.NewRequest(http.MethodPut, "/examples/abc-123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDelete_ShouldReturn200_WhenSuccessful(t *testing.T) {
	// Arrange
	uc := &mockUsecase{
		deleteFn: func(_ context.Context, _ string) error { return nil },
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.DELETE("/examples/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/examples/abc-123", nil)

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDelete_ShouldReturn500_WhenUsecaseFails(t *testing.T) {
	// Arrange
	uc := &mockUsecase{
		deleteFn: func(_ context.Context, _ string) error { return errors.New("db error") },
	}
	h := New(uc)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.DELETE("/examples/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/examples/abc-123", nil)

	// Act
	engine.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
