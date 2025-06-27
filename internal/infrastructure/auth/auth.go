// internal/infrastructure/auth/auth.go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// This is a very basic example. In a real app, use a proper JWT library with refresh tokens, revocation, etc.
const (
	// For simplicity, token expires in 24 hours.
	tokenExpiration = time.Hour * 24
)

var (
	jwtSecret []byte // This should be loaded from config
)

// SetJWTSecret initializes the JWT secret key. This function should be called once at application startup.
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

// GenerateJWT generates a new JWT token for a given user ID.
func GenerateJWT(userID string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", fmt.Errorf("JWT secret not set. Call auth.SetJWTSecret() first.")
	}
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(tokenExpiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT validates a JWT token and returns the user ID if valid.
func ValidateJWT(tokenString string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", fmt.Errorf("JWT secret not set. Call auth.SetJWTSecret() first.")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return "", fmt.Errorf("user_id claim not found or not string")
		}
		return userID, nil
	}
	return "", fmt.Errorf("invalid token")
}
