package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"

	billingv1 "parkir-pintar/proto/billing/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"
)

type BillingHandler struct {
	billing     billingv1.BillingServiceClient
	reservation reservationv1.ReservationServiceClient
}

func NewBillingHandler(billing billingv1.BillingServiceClient, reservation reservationv1.ReservationServiceClient) *BillingHandler {
	return &BillingHandler{billing: billing, reservation: reservation}
}

func (bh *BillingHandler) RegisterRoutes(engine *gin.Engine, mw *middleware.Middleware, jwtSecret string) {
	api := engine.Group("/api/v1")
	api.Use(mw.JWTAuth(jwtSecret))

	api.GET("/reservations/:id/billing", bh.GetReservationBilling)
}

func (bh *BillingHandler) GetReservationBilling(c *gin.Context) {
	reservationID := c.Param("id")
	if reservationID == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	_, err := bh.reservation.GetReservation(contextWithAuth(c), &reservationv1.GetReservationRequest{
		ReservationId: reservationID,
	})
	if err != nil {
		writeGRPCError(c, err)
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
