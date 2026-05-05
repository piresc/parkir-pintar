// Package handler provides Gin HTTP handlers for the example domain module.
// Handlers bind JSON requests, delegate to the usecase layer, and return
// standardized responses via the response package.
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"parkir-pintar/internal/example/model"
	"parkir-pintar/internal/example/usecase"
	"parkir-pintar/pkg/response"
)

// Handler holds the usecase dependency for example HTTP handlers.
type Handler struct {
	uc usecase.Usecase
}

// New creates a new Handler with the given usecase.
func New(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// List handles GET /api/v1/examples with optional limit and offset query params.
func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	examples, err := h.uc.List(c.Request.Context(), limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, examples)
}

// Get handles GET /api/v1/examples/:id.
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	example, err := h.uc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			response.Error(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, example)
}

// Create handles POST /api/v1/examples.
func (h *Handler) Create(c *gin.Context) {
	var req model.CreateExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	example, err := h.uc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, example)
}

// Update handles PUT /api/v1/examples/:id.
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdateExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	example, err := h.uc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, example)
}

// Delete handles DELETE /api/v1/examples/:id.
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.uc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "example deleted"})
}
