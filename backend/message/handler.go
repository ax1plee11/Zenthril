package message

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"zenthril-backend/auth"
	"zenthril-backend/models"
)

const (
	maxCiphertextBytes = 64 << 10
	maxKeyIDLength     = 128
	maxAADFieldLength  = 256
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	channelID := chi.URLParam(r, "channelId")

	payload, err := decodeEncryptedPayloadRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if err := validateEncryptedPayload(payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := validateEnvelopeClaims(payload, channelID, userID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	msg, err := h.svc.SendMessage(r.Context(), channelID, userID, payload)
	if err != nil {
		if errors.Is(err, ErrNotChannelMember) {
			writeError(w, http.StatusForbidden, "forbidden", "You do not have access to this channel")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to send message")
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

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
		if errors.Is(err, ErrNotChannelMember) {
			writeError(w, http.StatusForbidden, "forbidden", "You do not have access to this channel")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get messages")
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	payload, err := decodeEncryptedPayloadRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if err := validateEncryptedPayload(payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := validateEnvelopeClaims(payload, "", userID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	msg, err := h.svc.EditMessage(r.Context(), messageID, userID, payload)
	if err != nil {
		if errors.Is(err, ErrNotChannelMember) {
			writeError(w, http.StatusForbidden, "forbidden", "You do not have access to this channel")
			return
		}
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

func validateEnvelopeClaims(payload models.EncryptedPayload, channelID, userID string) error {
	if payload.ProtocolVersion != models.CryptoProtocolVersion {
		return nil
	}
	// SECURITY: v2 context fields are authenticated by AES-GCM AAD on the
	// client and must not contradict the authenticated server route context.
	if payload.ChannelID != "" && channelID != "" && payload.ChannelID != channelID {
		return errors.New("payload channel_id does not match route channel")
	}
	if payload.SenderUserID != "" && payload.SenderUserID != userID {
		return errors.New("payload sender_user_id does not match authenticated user")
	}
	return nil
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	err := h.svc.DeleteMessage(r.Context(), messageID, userID)
	if err != nil {
		if errors.Is(err, ErrNotChannelMember) {
			writeError(w, http.StatusForbidden, "forbidden", "You do not have access to this channel")
			return
		}
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

func decodeEncryptedPayloadRequest(r *http.Request) (models.EncryptedPayload, error) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return models.EncryptedPayload{}, err
	}

	var wrapped struct {
		Payload *models.EncryptedPayload `json:"payload"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Payload != nil {
		return *wrapped.Payload, nil
	}

	var payload models.EncryptedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return models.EncryptedPayload{}, err
	}
	return payload, nil
}

func validateEncryptedPayload(payload models.EncryptedPayload) error {
	ciphertext := strings.TrimSpace(payload.Ciphertext)
	iv := strings.TrimSpace(payload.IV)
	keyID := strings.TrimSpace(payload.KeyID)
	tag := strings.TrimSpace(payload.Tag)
	if ciphertext == "" || iv == "" || keyID == "" || tag == "" {
		return errors.New("ciphertext, iv, key_id and tag are required")
	}
	if len(keyID) > maxKeyIDLength {
		return fmt.Errorf("key_id must be at most %d characters", maxKeyIDLength)
	}
	if payload.ProtocolVersion != models.LegacyCryptoProtocolVersion && payload.ProtocolVersion != models.CryptoProtocolVersion {
		return fmt.Errorf("protocol_version must be %d or %d", models.LegacyCryptoProtocolVersion, models.CryptoProtocolVersion)
	}
	ciphertextBytes, err := decodeBase64(ciphertext)
	if err != nil {
		return errors.New("ciphertext must be base64")
	}
	if len(ciphertextBytes) == 0 || len(ciphertextBytes) > maxCiphertextBytes {
		return fmt.Errorf("ciphertext must decode to 1..%d bytes", maxCiphertextBytes)
	}
	ivBytes, err := decodeBase64(iv)
	if err != nil {
		return errors.New("iv must be base64")
	}
	if len(ivBytes) != 12 {
		return errors.New("iv must decode to 12 bytes")
	}
	tagBytes, err := decodeBase64(tag)
	if err != nil {
		return errors.New("tag must be base64")
	}
	if len(tagBytes) != 16 {
		return errors.New("tag must decode to 16 bytes")
	}
	if payload.ProtocolVersion == models.CryptoProtocolVersion {
		if err := validateAADV2Fields(payload); err != nil {
			return err
		}
	}
	return nil
}

func validateAADV2Fields(payload models.EncryptedPayload) error {
	required := map[string]string{
		"sender_device_id":  payload.SenderDeviceID,
		"session_id":        payload.SessionID,
		"client_message_id": payload.ClientMessageID,
		"cipher_suite":      payload.CipherSuite,
	}
	for name, value := range required {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("%s is required for protocol_version %d", name, models.CryptoProtocolVersion)
		}
		if len(value) > maxAADFieldLength {
			return fmt.Errorf("%s must be at most %d characters", name, maxAADFieldLength)
		}
	}
	if payload.CipherSuite != models.CipherSuiteV2 {
		return errors.New("unsupported cipher_suite")
	}
	return nil
}

func decodeBase64(value string) ([]byte, error) {
	if out, err := base64.StdEncoding.DecodeString(value); err == nil {
		return out, nil
	}
	if out, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return out, nil
	}
	if out, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return out, nil
	}
	return base64.URLEncoding.DecodeString(value)
}
