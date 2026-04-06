package auth

import (
	"context"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"

// Middleware возвращает HTTP middleware для проверки JWT-токена.
// Добавляет userID в контекст запроса при успешной валидации.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing_token", "Authorization header required")
			return
		}

		userID, err := ValidateToken(token, s.jwtSecret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_token", "Token is invalid or expired")
			return
		}

		// Проверяем blacklist
		blacklisted, err := s.IsTokenBlacklisted(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Token validation failed")
			return
		}
		if blacklisted {
			writeError(w, http.StatusUnauthorized, "token_revoked", "Token has been revoked")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext извлекает userID из контекста запроса.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}
