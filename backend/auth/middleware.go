package auth

import (
	"context"
	"log/slog"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing_token", "Authorization header required")
			return
		}

		userID, err := ValidateToken(token, s.jwtSecret)
		if err != nil {
			slog.Warn("security access token rejected", "reason", "invalid_or_expired", "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "invalid_token", "Token is invalid or expired")
			return
		}

		blacklisted, err := s.IsTokenBlacklisted(r.Context(), token)
		if err != nil {
			slog.Error("security token blacklist check failed", "error", err, "path", r.URL.Path)
			writeError(w, http.StatusInternalServerError, "internal_error", "Token validation failed")
			return
		}
		if blacklisted {
			// SECURITY-HARDENING: reject Redis-revoked access tokens before reaching handlers.
			slog.Warn("security revoked access token rejected", "user_id", userID, "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "token_revoked", "Token has been revoked")
			return
		}

		banned, err := s.IsGloballyBanned(r.Context(), userID)
		if err == nil && banned {
			reason, _ := s.GetGlobalBanReason(r.Context(), userID)
			if reason == "" {
				reason = "Your account has been permanently banned"
			}
			writeError(w, http.StatusForbidden, "account_banned", reason)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}
