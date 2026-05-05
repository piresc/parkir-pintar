package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// SignatureConfig holds configuration for signature generation and validation.
type SignatureConfig struct {
	Secret         string
	ValidityWindow time.Duration
}

// GenerateSignature creates an HMAC-SHA256 signature of the payload using the secret.
// The output is hex-encoded.
func GenerateSignature(payload string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateSignature verifies an HMAC-SHA256 signature against the payload.
// It also checks that the timestamp is within the validity window.
// Returns an error if the signature is expired or does not match.
func ValidateSignature(signature, payload, secret string, validityWindow time.Duration, timestamp int64) error {
	if signature == "" {
		return errors.New("signature is required")
	}
	if payload == "" {
		return errors.New("payload is required")
	}
	if secret == "" {
		return errors.New("secret is required")
	}

	// Check expiration: if timestamp + validityWindow < now, signature is expired
	expiresAt := time.Unix(timestamp, 0).Add(validityWindow)
	if time.Now().After(expiresAt) {
		return fmt.Errorf("signature expired at %s", expiresAt.Format(time.RFC3339))
	}

	expected := GenerateSignature(payload, secret)

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("invalid signature")
	}

	return nil
}
