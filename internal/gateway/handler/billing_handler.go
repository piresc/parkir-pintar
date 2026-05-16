// Package handler — billing REST endpoint for the API Gateway.
// Transcodes REST requests to gRPC calls to the billing microservice.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"

	billingv1 "parkir-pintar/proto/billing/v1"
)

// BillingHandler provides the reservation billing breakdown endpoint via gRPC.
type BillingHandler struct {
	billing billingv1.BillingServiceClient
}

// NewBillingHandler creates a new BillingHandler with the given billing gRPC client.
func NewBillingHandler(billing billingv1.BillingServiceClient) *BillingHandler {
	return &BillingHandler{billing: billing}
}

// RegisterRoutes registers billing REST routes on the Gin engine with JWT auth.
func (bh *BillingHandler) RegisterRoutes(engine *gin.Engine, mw *middleware.Middleware, jwtSecret string) {
	api := engine.Group("/api/v1")
	api.Use(mw.JWTAuth(jwtSecret))

	api.GET("/reservations/:id/billing", bh.GetReservationBilling)
}

// GetReservationBilling handles GET /api/v1/reservations/:id/billing.
// Returns the billing breakdown for a given reservation via the billing gRPC service.
func (bh *BillingHandler) GetReservationBilling(c *gin.Context) {
	reservationID := c.Param("id")
	if reservationID == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	resp, err := bh.billing.GenerateInvoice(contextWithAuth(c), &billingv1.GenerateInvoiceRequest{
		ReservationId: reservationID,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
