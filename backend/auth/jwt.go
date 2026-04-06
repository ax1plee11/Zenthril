package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenTTL = 24 * time.Hour

type claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken создаёт JWT-токен для пользователя со сроком действия 24 часа.
func GenerateToken(userID string, secret string) (string, error) {
	now := time.Now()
	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ValidateToken проверяет JWT-токен и возвращает userID.
func ValidateToken(tokenStr, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}

	return c.UserID, nil
}
