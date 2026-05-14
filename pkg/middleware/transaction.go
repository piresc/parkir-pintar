package middleware

import (
	"time"

	"parkir-pintar/pkg/config"

	"github.com/gin-gonic/gin"
)

// GenerateTransactionID returns middleware that creates a unique transaction ID
// with the format: {appID}{timestamp}{msisdnSuffix}.
//
// The appID prefix is environment-aware:
//   - "L" for local
//   - "D" for development
//   - "S" for staging
//   - "P" for production
//
// If an existing transactionid header is present, it is preserved in the
// "oldtransactionid" header before being replaced.
func (m *Middleware) GenerateTransactionID() gin.HandlerFunc {
	return func(c *gin.Context) {
		msisdn := c.GetHeader("x-msisdn")

		// Preserve old transaction ID if present
		if old := c.GetHeader("transactionid"); old != "" {
			c.Request.Header.Set("oldtransactionid", old)
		}

		txID := createTransactionID(m.config, msisdn)
		c.Request.Header.Set("transactionid", txID)
		c.Set(KeyTransactionID, txID)

		c.Next()
	}
}

// createTransactionID builds a transaction ID string from the environment
// prefix, current timestamp, and the last 5 digits of the MSISDN.
func createTransactionID(cfg *config.Config, msisdn string) string {
	appID := envAppID(cfg)

	suffix := msisdn
	if len(msisdn) >= 5 {
		suffix = msisdn[len(msisdn)-5:]
	}

	ts := time.Now().Format("060102150405000")
	return appID + ts + suffix
}

// Environment name constants.
const envLocal = "local"

// envAppID returns a short prefix based on the configured environment.
func envAppID(cfg *config.Config) string {
	if cfg == nil {
		return "L"
	}
	switch cfg.App.Environment {
	case envLocal:
		return "L"
	case "development":
		return "D"
	case "staging":
		return "S"
	case "production":
		return "P"
	default:
		return "L"
	}
}
