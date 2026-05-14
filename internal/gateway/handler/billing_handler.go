// Package handler — billing REST endpoint for the API Gateway.
// Queries billing_records directly via pgClient (same pattern as analytics).
package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"
)

// billingRecord is the response struct for the billing breakdown endpoint.
type billingRecord struct {
	ID              string    `json:"id" db:"id"`
	ReservationID   string    `json:"reservation_id" db:"reservation_id"`
	BookingFee      int64     `json:"booking_fee" db:"booking_fee"`
	ParkingFee      int64     `json:"parking_fee" db:"parking_fee"`
	OvernightFee    int64     `json:"overnight_fee" db:"overnight_fee"`
	CancellationFee int64     `json:"cancellation_fee" db:"cancellation_fee"`
	PenaltyAmount   int64     `json:"penalty_amount" db:"penalty_amount"`
	TotalAmount     int64     `json:"total_amount" db:"total_amount"`
	DurationMinutes int       `json:"duration_minutes" db:"duration_minutes"`
	BilledHours     int       `json:"billed_hours" db:"billed_hours"`
	IsOvernight     bool      `json:"is_overnight" db:"is_overnight"`
	Status          string    `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// BillingHandler provides the reservation billing breakdown endpoint.
type BillingHandler struct {
	db *sqlx.DB
}

// NewBillingHandler creates a new BillingHandler with the given sqlx.DB.
func NewBillingHandler(db *sqlx.DB) *BillingHandler {
	return &BillingHandler{db: db}
}

// RegisterRoutes registers billing REST routes on the Gin engine with JWT auth.
func (bh *BillingHandler) RegisterRoutes(engine *gin.Engine, mw *middleware.Middleware, jwtSecret string) {
	api := engine.Group("/api/v1")
	api.Use(mw.JWTAuth(jwtSecret))

	api.GET("/reservations/:id/billing", bh.GetReservationBilling)
}

// GetReservationBilling handles GET /api/v1/reservations/:id/billing.
// Returns the billing breakdown for a given reservation.
func (bh *BillingHandler) GetReservationBilling(c *gin.Context) {
	reservationID := c.Param("id")
	if reservationID == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	var record billingRecord
	err := bh.db.GetContext(c.Request.Context(), &record,
		"SELECT id, reservation_id, booking_fee, parking_fee, overnight_fee, cancellation_fee, penalty_amount, total_amount, duration_minutes, billed_hours, is_overnight, status, created_at, updated_at FROM billing_records WHERE reservation_id = $1",
		reservationID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "billing record not found for this reservation")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to retrieve billing record")
		return
	}

	response.Success(c, http.StatusOK, record)
}
