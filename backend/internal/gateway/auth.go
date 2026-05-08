package gateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserID    string
	DeviceID  string
	SessionID string
}

type Authenticator interface {
	AuthenticateWebSocket(ctx context.Context, token string) (UserClaims, error)
}

type JWTAuthenticator struct {
	secret string
}

func NewJWTAuthenticator(secret string) *JWTAuthenticator {
	return &JWTAuthenticator{secret: secret}
}

func (a *JWTAuthenticator) AuthenticateWebSocket(ctx context.Context, tokenString string) (UserClaims, error) {
	if err := ctx.Err(); err != nil {
		return UserClaims{}, err
	}
	if tokenString == "" {
		return UserClaims{}, errors.New("missing websocket token")
	}
	if a.secret == "" {
		return UserClaims{}, errors.New("jwt secret is not configured")
	}

	claims := jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.secret), nil
	})
	if err != nil {
		return UserClaims{}, fmt.Errorf("parse websocket token: %w", err)
	}
	if !token.Valid || claims.UserID == "" {
		return UserClaims{}, errors.New("invalid websocket token")
	}

	return UserClaims{
		UserID:    claims.UserID,
		DeviceID:  claims.DeviceID,
		SessionID: claims.ID,
	}, nil
}

type jwtClaims struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id,omitempty"`
	jwt.RegisteredClaims
}
