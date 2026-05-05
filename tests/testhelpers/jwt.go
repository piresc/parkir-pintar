// Package testhelpers provides shared utilities for both E2E test layers.
package testhelpers

import (
	"parkir-pintar/pkg/auth"
	"parkir-pintar/pkg/config"
)

// GenerateTestJWT creates a valid JWT token for test authentication.
// Uses pkg/auth.GenerateToken with sensible defaults: 60-min expiration, "parkir-pintar" issuer.
// Panics on error since test helpers should not silently fail.
func GenerateTestJWT(userID, role, secret string) string {
	cfg := config.JWTConfig{
		Secret:     secret,
		Expiration: 60,
		Issuer:     "parkir-pintar",
	}

	token, _, err := auth.GenerateToken(userID, role, cfg)
	if err != nil {
		panic("testhelpers: GenerateTestJWT failed: " + err.Error())
	}

	return token
}
