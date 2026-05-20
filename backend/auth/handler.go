package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "username and password are required")
		return
	}

	// Security: Validate password complexity
	if err := ValidatePasswordComplexity(req.Password); err != nil {
		var message string
		switch {
		case errors.Is(err, ErrPasswordTooShort):
			message = "Password must be at least 8 characters"
		case errors.Is(err, ErrPasswordNoUppercase):
			message = "Password must contain at least one uppercase letter"
		case errors.Is(err, ErrPasswordNoLowercase):
			message = "Password must contain at least one lowercase letter"
		case errors.Is(err, ErrPasswordNoNumber):
			message = "Password must contain at least one number"
		case errors.Is(err, ErrPasswordNoSpecialChar):
			message = "Password must contain at least one special character"
		default:
			message = "Password does not meet complexity requirements"
		}
		writeError(w, http.StatusBadRequest, "weak_password", message)
		return
	}

	user, pair, err := h.svc.Register(r.Context(), req.Username, req.Password, req.PublicKey)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			writeError(w, http.StatusConflict, "username_taken", "Username is already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user_id":       user.ID.String(),
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
		"token":         pair.AccessToken,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	user, pair, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Login failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
		"token":         pair.AccessToken,
		"user":          user,
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	pair, err := h.svc.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh token is invalid or expired")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to refresh tokens")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
		"token":         pair.AccessToken,
	})
}

func (h *Handler) WSTicket(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	ticket, err := h.svc.IssueWSTicket(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to issue ticket")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ticket": ticket})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	accessToken := extractBearerToken(r)

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.svc.Logout(r.Context(), accessToken, body.RefreshToken); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Logout failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GlobalBan(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	targetUserID := r.PathValue("userId")
	if targetUserID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "userId is required")
		return
	}
	if targetUserID == requesterID {
		writeError(w, http.StatusBadRequest, "invalid_request", "Cannot ban yourself")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.svc.GlobalBan(r.Context(), targetUserID, requesterID, req.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to ban user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GlobalUnban(w http.ResponseWriter, r *http.Request) {
	_, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	targetUserID := r.PathValue("userId")
	if targetUserID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "userId is required")
		return
	}

	if err := h.svc.GlobalUnban(r.Context(), targetUserID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to unban user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	user, err := h.svc.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":  user.ID.String(),
		"username": user.Username,
	})
}

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
