// - Use keyed fields in struct literals to prevent breakages during refactors
// Key fix from boilerplate-golang: Error always uses the provided statusCode
package response

import (
	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

type ErrorResponse struct {
	Status    string `json:"status"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// The response body always contains {"status": "success", "data": ...}.
func Success(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, SuccessResponse{
		Status: "success",
		Data:   data,
	})
}

func Error(c *gin.Context, statusCode int, err string) {
	c.JSON(statusCode, ErrorResponse{
		Status: "error",
		Error:  err,
	})
}

func ErrorWithRequestID(c *gin.Context, statusCode int, err, requestID string) {
	c.JSON(statusCode, ErrorResponse{
		Status:    "error",
		Error:     err,
		RequestID: requestID,
	})
}
