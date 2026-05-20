package device

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"zenthril-backend/auth"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	device, err := h.svc.RegisterDevice(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidDeviceKey) {
			writeError(w, http.StatusBadRequest, "invalid_device_key", err.Error())
			return
		}
		if errors.Is(err, ErrDeviceNotFound) {
			writeError(w, http.StatusForbidden, "device_owner_mismatch", "Device belongs to another user")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to register device")
		return
	}
	writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) ListOwn(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}
	h.listUserDevices(w, r, userID)
}

func (h *Handler) ListUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "userId is required")
		return
	}
	h.listUserDevices(w, r, userID)
}

func (h *Handler) RevokeOwn(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	deviceID := chi.URLParam(r, "deviceId")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "deviceId is required")
		return
	}

	if err := h.svc.RevokeDevice(r.Context(), userID, deviceID); err != nil {
		if errors.Is(err, ErrInvalidDeviceKey) {
			writeError(w, http.StatusBadRequest, "invalid_device_key", err.Error())
			return
		}
		if errors.Is(err, ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device_not_found", "Device not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to revoke device")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ClaimKeyBundle(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := auth.UserIDFromContext(r.Context())
	if !ok || requesterID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var req struct {
		UserID   string `json:"user_id"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.UserID == "" || req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "user_id and device_id are required")
		return
	}

	bundle, err := h.svc.ClaimKeyBundle(r.Context(), requesterID, req.UserID, req.DeviceID)
	if err != nil {
		if errors.Is(err, ErrInvalidDeviceKey) {
			writeError(w, http.StatusBadRequest, "invalid_device_key", err.Error())
			return
		}
		if errors.Is(err, ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device_not_found", "Device not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to claim key bundle")
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func (h *Handler) listUserDevices(w http.ResponseWriter, r *http.Request, userID string) {
	devices, err := h.svc.ListUserDevices(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrInvalidDeviceKey) {
			writeError(w, http.StatusBadRequest, "invalid_device_key", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list devices")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
