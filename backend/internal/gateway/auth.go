package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const maxWebSocketBearerTokenBytes = 4096

type UserClaims struct {
	UserID    string
	DeviceID  string
	SessionID string
}

type Authenticator interface {
	AuthenticateWebSocket(ctx context.Context, token string) (UserClaims, error)
}

type JWTAuthenticator struct {
	secret    string
	validator SessionValidator
}

func NewJWTAuthenticator(secret string, validator SessionValidator) *JWTAuthenticator {
	return &JWTAuthenticator{secret: secret, validator: validator}
}

func (a *JWTAuthenticator) AuthenticateWebSocket(ctx context.Context, tokenString string) (UserClaims, error) {
	if err := ctx.Err(); err != nil {
		return UserClaims{}, err
	}
	if tokenString == "" {
		return UserClaims{}, errors.New("missing websocket token")
	}
	tokenString = strings.TrimSpace(tokenString)
	if len(tokenString) > maxWebSocketBearerTokenBytes {
		// SECURITY: reject oversized bearer tokens before any Redis/session lookup.
		// WEAKNESS FIXED: malformed input cannot amplify load on the session backend.
		return UserClaims{}, errors.New("websocket token is too large")
	}
	if a.secret == "" {
		return UserClaims{}, errors.New("jwt secret is not configured")
	}

	claims := jwtClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			// SECURITY: reject alg=none and algorithm confusion by pinning exactly HS256.
			// WEAKNESS FIXED: gateway authentication cannot be bypassed with unsigned or wrong-alg tokens.
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(a.secret), nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return UserClaims{}, fmt.Errorf("parse websocket token: %w", err)
	}
	if !token.Valid || claims.UserID == "" {
		return UserClaims{}, errors.New("invalid websocket token")
	}

	// SECURITY: session lifecycle checks run only after local JWT validation.
	// WEAKNESS FIXED: random invalid tokens no longer cause Redis blacklist lookups.
	if a.validator != nil {
		blacklisted, err := a.validator.IsTokenBlacklisted(ctx, tokenString)
		if err != nil {
			return UserClaims{}, fmt.Errorf("check token blacklist: %w", err)
		}
		if blacklisted {
			return UserClaims{}, errors.New("token revoked")
		}

		// SECURITY: globally banned accounts must not establish realtime sessions.
		banned, err := a.validator.IsGloballyBanned(ctx, claims.UserID)
		if err != nil {
			return UserClaims{}, fmt.Errorf("check global ban: %w", err)
		}
		if banned {
			return UserClaims{}, errors.New("account banned")
		}
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

// NoopSessionValidator allows all sessions when no lifecycle backend is wired.
type NoopSessionValidator struct{}

func (NoopSessionValidator) IsTokenBlacklisted(context.Context, string) (bool, error) {
	return false, nil
}

func (NoopSessionValidator) IsGloballyBanned(context.Context, string) (bool, error) {
	return false, nil
}
