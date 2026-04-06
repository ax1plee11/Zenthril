package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// Handler содержит HTTP-обработчики для аутентификации.
type Handler struct {
	svc *Service
}

// NewHandler создаёт новый Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register обрабатывает POST /api/v1/auth/register
// Response 201: { user_id, token } или 409: { error: "username_taken" }
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

	user, token, err := h.svc.Register(r.Context(), req.Username, req.Password, req.PublicKey)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			writeError(w, http.StatusConflict, "username_taken", "Username is already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"user_id": user.ID.String(),
		"token":   token,
	})
}

// Login обрабатывает POST /api/v1/auth/login
// Response 200: { token, user } или 401: { error: "invalid_credentials" }
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	user, token, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Login failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// Logout обрабатывает POST /api/v1/auth/logout
// Response 204
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		writeError(w, http.StatusBadRequest, "missing_token", "Authorization header required")
		return
	}

	if err := h.svc.Logout(r.Context(), token); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Logout failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// extractBearerToken извлекает токен из заголовка Authorization: Bearer <token>
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
