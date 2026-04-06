package friends

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"veltrix-backend/auth"
)

// FriendNotifier — интерфейс для WS-уведомлений.
type FriendNotifier interface {
	BroadcastToUser(userID string, msg []byte)
}

type Handler struct {
	svc      *Service
	notifier FriendNotifier
}

func NewHandler(svc *Service, n FriendNotifier) *Handler {
	return &Handler{svc: svc, notifier: n}
}

// GET /api/v1/friends — список друзей + запросы
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	list, err := h.svc.ListFriends(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list friends")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// POST /api/v1/friends/request — отправить запрос { user_id }
func (h *Handler) SendRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "user_id is required")
		return
	}

	// Получаем username отправителя для уведомления
	username, _ := h.svc.GetUsername(r.Context(), userID)

	err := h.svc.SendRequest(r.Context(), userID, req.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyFriends):
			writeError(w, http.StatusConflict, "already_friends", "Already friends")
		case errors.Is(err, ErrRequestPending):
			writeError(w, http.StatusConflict, "request_pending", "Request already pending")
		case errors.Is(err, ErrCannotSelfAdd):
			writeError(w, http.StatusBadRequest, "cannot_add_self", "Cannot add yourself")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to send request")
		}
		return
	}

	// WS-уведомление адресату
	if h.notifier != nil {
		msg, _ := json.Marshal(map[string]string{
			"type":          "friend.request",
			"from_user_id":  userID,
			"from_username": username,
		})
		h.notifier.BroadcastToUser(req.UserID, msg)
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/friends/{userId}/accept — принять запрос
func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	otherID := chi.URLParam(r, "userId")
	if err := h.svc.AcceptRequest(r.Context(), userID, otherID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Request not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to accept")
		return
	}

	// WS-уведомление тому кто отправил запрос — его приняли
	if h.notifier != nil {
		username, _ := h.svc.GetUsername(r.Context(), userID)
		msg, _ := json.Marshal(map[string]string{
			"type":          "friend.accepted",
			"from_user_id":  userID,
			"from_username": username,
		})
		h.notifier.BroadcastToUser(otherID, msg)
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/friends/{userId} — удалить друга или отклонить запрос
func (h *Handler) Decline(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	otherID := chi.URLParam(r, "userId")
	if err := h.svc.DeclineRequest(r.Context(), userID, otherID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to remove")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}
