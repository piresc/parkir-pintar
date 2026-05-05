package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// NormalizeMsisdn returns middleware that normalizes the x-msisdn header:
//   - Empty → "00000"
//   - Leading "+" is stripped (e.g. "+6281234" → "6281234")
//   - Leading "0" is converted to "62" (e.g. "081234" → "6281234")
//
// The normalized value is set back in both the request header and gin.Context.
func (m *Middleware) NormalizeMsisdn() gin.HandlerFunc {
	return func(c *gin.Context) {
		msisdn := c.GetHeader("x-msisdn")

		switch {
		case len(msisdn) < 1:
			msisdn = "00000"
		case strings.HasPrefix(msisdn, "+"):
			msisdn = msisdn[1:]
		case strings.HasPrefix(msisdn, "0"):
			msisdn = "62" + msisdn[1:]
		}

		c.Request.Header.Set("x-msisdn", msisdn)
		c.Set(KeyMsisdn, msisdn)
		c.Next()
	}
}
