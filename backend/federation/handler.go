package federation

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Announce(w http.ResponseWriter, r *http.Request) {
	// SECURITY-HARDENING: federation announcements are bounded before JSON decoding.
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req struct {
		Domain    string `json:"domain"`
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	node, err := h.svc.Announce(r.Context(), req.Domain, req.PublicKey)
	if err != nil {
		if errors.Is(err, ErrInvalidNode) {
			writeError(w, http.StatusBadRequest, "invalid_node", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to announce federation node")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": "accepted",
		"node":   node,
	})
}

func (h *Handler) Peers(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.svc.ListPeers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list federation peers")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"peers": nodes,
	})
}

func (h *Handler) Inbox(w http.ResponseWriter, r *http.Request) {
	// SECURITY: federation inbox accepts encrypted message envelopes only and keeps
	// a strict body limit while federation remains alpha.
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var req MessageEnvelope
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	message, err := h.svc.ReceiveMessage(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidFederationMessage) {
			writeError(w, http.StatusBadRequest, "invalid_federation_message", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to store federation message")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":  "accepted",
		"message": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
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
