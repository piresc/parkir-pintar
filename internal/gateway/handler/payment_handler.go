package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/response"

	paymentv1 "parkir-pintar/proto/payment/v1"
)

func (h *Handler) GetPaymentStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "payment id is required")
		return
	}

	resp, err := h.payment.GetPaymentStatus(contextWithAuth(c), &paymentv1.GetPaymentStatusRequest{
		PaymentId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
