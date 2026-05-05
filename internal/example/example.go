// Package example is the domain module entry point that wires handler → usecase → repository
// and registers routes under /api/v1/examples.
package example

import (
	"github.com/gin-gonic/gin"

	"parkir-pintar/internal/example/handler"
	"parkir-pintar/internal/example/repository"
	"parkir-pintar/internal/example/usecase"
	"parkir-pintar/pkg/database"
	"parkir-pintar/pkg/middleware"
)

// RegisterRoutes wires the example domain module and registers CRUD routes.
// It creates the repository, usecase, and handler layers, then mounts
// endpoints under /api/v1/examples with JWT authentication.
func RegisterRoutes(
	r *gin.Engine,
	mw *middleware.Middleware,
	db *database.TracedPostgresClient,
) {
	repo := repository.New(db.GetDB(), nil)
	uc := usecase.New(repo)
	h := handler.New(uc)

	api := r.Group("/api/v1")
	{
		api.GET("/examples", h.List)
		api.GET("/examples/:id", h.Get)
		api.POST("/examples", h.Create)
		api.PUT("/examples/:id", h.Update)
		api.DELETE("/examples/:id", h.Delete)
	}
}
