// Package commonauth contains JWT helpers shared between the auth-service
// (which issues tokens) and the api-gateway middleware (which validates them).
// Both sides use the same HMAC secret loaded from JWT_SECRET in the env.
package commonauth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// GenerateJWT issues a signed JWT for the given user. Claims: sub, exp, iat, tokenType.
func GenerateJWT(userID uuid.UUID, tokenType TokenType, secret string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":       userID.String(),
		"exp":       time.Now().Add(ttl).Unix(),
		"iat":       time.Now().Unix(),
		"tokenType": string(tokenType),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken validates a JWT (HMAC only) and returns its claims.
func ParseToken(tokenStr, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// ValidateRefreshToken parses a refresh token and returns its user id.
func ValidateRefreshToken(tokenStr, secret string) (uuid.UUID, error) {
	claims, err := ParseToken(tokenStr, secret)
	if err != nil {
		return uuid.Nil, err
	}
	if claims["tokenType"] != string(TokenTypeRefresh) {
		return uuid.Nil, errors.New("invalid token type")
	}
	return UserIDFromClaims(claims)
}

// UserIDFromClaims extracts and parses the sub claim as a UUID.
func UserIDFromClaims(claims jwt.MapClaims) (uuid.UUID, error) {
	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, errors.New("invalid sub claim")
	}
	return uuid.Parse(sub)
}
