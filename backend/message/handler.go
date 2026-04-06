package message

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"veltrix-backend/auth"
	"veltrix-backend/models"
)

// Handler содержит HTTP-обработчики для работы с сообщениями.
type Handler struct {
	svc *Service
}

// NewHandler создаёт новый Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SendMessage обрабатывает POST /api/v1/channels/:channelId/messages
// Response 201: Message или 429
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	channelID := chi.URLParam(r, "channelId")

	var payload models.EncryptedPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if payload.Ciphertext == "" || payload.IV == "" || payload.KeyID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "ciphertext, iv and key_id are required")
		return
	}

	msg, err := h.svc.SendMessage(r.Context(), channelID, userID, payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to send message")
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

// GetHistory обрабатывает GET /api/v1/channels/:channelId/messages?before=<id>&limit=50
// Response 200: []Message
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	channelID := chi.URLParam(r, "channelId")
	userID, _ := auth.UserIDFromContext(r.Context())

	var before *string
	if b := r.URL.Query().Get("before"); b != "" {
		before = &b
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	messages, err := h.svc.GetHistory(r.Context(), channelID, userID, before, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get messages")
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// EditMessage обрабатывает PATCH /api/v1/messages/:messageId
// Response 200: Message
func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	var payload models.EncryptedPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if payload.Ciphertext == "" || payload.IV == "" || payload.KeyID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "ciphertext, iv and key_id are required")
		return
	}

	msg, err := h.svc.EditMessage(r.Context(), messageID, userID, payload)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "You are not the author of this message")
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to edit message")
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

// DeleteMessage обрабатывает DELETE /api/v1/messages/:messageId
// Response 204
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	err := h.svc.DeleteMessage(r.Context(), messageID, userID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "You are not the author of this message")
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete message")
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
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
