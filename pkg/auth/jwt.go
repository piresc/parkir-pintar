package auth

import (
	"errors"
	"fmt"
	"time"

	"parkir-pintar/pkg/config"

	"github.com/golang-jwt/jwt/v4"
)

// Claims represents the JWT claims payload with user identity fields.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT token for the given user with HS256.
// It sets user_id, role, exp (from cfg.Expiration minutes), iss (from cfg.Issuer), and iat.
func GenerateToken(userID, role string, cfg config.JWTConfig) (token string, expiresAt int64, err error) {
	if cfg.Secret == "" {
		return "", 0, errors.New("jwt secret is required")
	}
	if userID == "" {
		return "", 0, errors.New("user ID is required")
	}

	now := time.Now()
	exp := now.Add(time.Duration(cfg.Expiration) * time.Minute)

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    cfg.Issuer,
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := t.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, exp.Unix(), nil
}

// ValidateToken parses and validates a JWT token string using the provided secret.
// It verifies the signing method is HMAC and returns the extracted Claims.
func ValidateToken(tokenString, secret string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token string is required")
	}
	if secret == "" {
		return nil, errors.New("secret is required")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// Verify signing method is HMAC
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
