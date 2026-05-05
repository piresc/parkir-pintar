package middleware

import (
	"github.com/gin-gonic/gin"
)

// Context key constants for values stored in gin.Context.
const (
	KeyTransactionID = "transactionid"
	KeyMsisdn        = "x-msisdn"
	KeyAppVersion    = "appversion"
	KeyOSVersion     = "osversion"
	KeyDeviceID      = "x-device"
)

// SetContextValues returns middleware that extracts standard headers and
// stores them in the gin.Context for downstream handlers.
//
// Headers extracted: transactionid, x-msisdn, appversion (with fallback
// to mytelkomsel-mobile-app-version / mytelkomsel-web-app-version),
// osversion, x-device.
func (m *Middleware) SetContextValues() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Transaction ID
		c.Set(KeyTransactionID, c.GetHeader("transactionid"))

		// MSISDN
		c.Set(KeyMsisdn, c.GetHeader("x-msisdn"))

		// App version — try dedicated header first, then channel-specific fallbacks
		appVersion := c.GetHeader("appversion")
		if appVersion == "" {
			appVersion = c.GetHeader("mytelkomsel-mobile-app-version")
		}
		if appVersion == "" {
			appVersion = c.GetHeader("mytelkomsel-web-app-version")
		}
		c.Set(KeyAppVersion, appVersion)

		// OS version
		c.Set(KeyOSVersion, c.GetHeader("osversion"))

		// Device ID
		c.Set(KeyDeviceID, c.GetHeader("x-device"))

		c.Next()
	}
}
