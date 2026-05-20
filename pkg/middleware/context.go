package middleware

import (
	"github.com/gin-gonic/gin"
)

const (
	KeyTransactionID = "transactionid"
	KeyMsisdn        = "x-msisdn"
	KeyAppVersion    = "appversion"
	KeyOSVersion     = "osversion"
	KeyDeviceID      = "x-device"
)

func (m *Middleware) SetContextValues() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(KeyTransactionID, c.GetHeader("transactionid"))

		c.Set(KeyMsisdn, c.GetHeader("x-msisdn"))

		appVersion := c.GetHeader("appversion")
		if appVersion == "" {
			appVersion = c.GetHeader("mytelkomsel-mobile-app-version")
		}
		if appVersion == "" {
			appVersion = c.GetHeader("mytelkomsel-web-app-version")
		}
		c.Set(KeyAppVersion, appVersion)

		c.Set(KeyOSVersion, c.GetHeader("osversion"))

		c.Set(KeyDeviceID, c.GetHeader("x-device"))

		c.Next()
	}
}
