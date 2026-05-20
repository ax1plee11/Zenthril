package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

type claims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func GenerateToken(userID string, secret string) (string, error) {
	return generateTypedToken(userID, "access", accessTokenTTL, secret)
}

func generateTypedToken(userID, tokenType string, ttl time.Duration, secret string) (string, error) {
	now := time.Now()
	c := claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func ValidateToken(tokenStr, secret string) (string, error) {
	return validateTypedToken(tokenStr, secret, "access")
}

func ValidateRefreshToken(tokenStr, secret string) (string, error) {
	return validateTypedToken(tokenStr, secret, "refresh")
}

func ValidateRefreshTokenWithID(tokenStr, secret string) (userID, tokenID string, err error) {
	claims, err := parseTypedToken(tokenStr, secret, "refresh")
	if err != nil {
		return "", "", err
	}
	if claims.ID == "" {
		return "", "", errors.New("missing token id")
	}
	return claims.UserID, claims.ID, nil
}

func validateTypedToken(tokenStr, secret, expectedType string) (string, error) {
	claims, err := parseTypedToken(tokenStr, secret, expectedType)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func parseTypedToken(tokenStr, secret, expectedType string) (*claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if c.TokenType != expectedType {
		return nil, fmt.Errorf("wrong token type: expected %s, got %s", expectedType, c.TokenType)
	}

	return c, nil
}
