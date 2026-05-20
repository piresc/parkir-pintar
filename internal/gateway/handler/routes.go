package handler

import (
	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/middleware"
)

// RegisterRoutes wires all gateway HTTP routes onto the given Gin engine.
func (h *Handler) RegisterRoutes(engine *gin.Engine, mw *middleware.Middleware, jwtSecret string) {
	public := engine.Group("/api/v1")
	public.POST("/auth/login", h.Login)

	api := engine.Group("/api/v1")
	api.Use(mw.JWTAuth(jwtSecret))

	api.POST("/reservations", h.CreateReservation)
	api.GET("/reservations", h.ListByDriver)
	api.GET("/reservations/:id", h.GetReservation)
	api.DELETE("/reservations/:id", h.CancelReservation)
	api.POST("/reservations/:id/checkin", h.CheckIn)
	api.POST("/reservations/:id/checkout", h.CheckOut)
	api.POST("/reservations/:id/confirm", h.ConfirmReservation)
	api.POST("/reservations/:id/complete", h.CompleteCheckout)

	api.GET("/availability", h.GetAvailability)
	api.GET("/floors/:floor", h.GetFloorMap)
	api.GET("/spots/:id", h.GetSpotDetails)

	api.GET("/payments/:id/status", h.GetPaymentStatus)

	api.POST("/presence/stream", h.StreamPresence)
}
