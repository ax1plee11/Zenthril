package guild

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"zenthril-backend/auth"
)

type GuildNotifier interface {
	BroadcastToUser(userID string, msg []byte)
	BroadcastToGuild(guildID string, msg []byte)
}

type Handler struct {
	svc      *Service
	notifier GuildNotifier
}

func NewHandler(svc *Service, n GuildNotifier) *Handler {
	return &Handler{svc: svc, notifier: n}
}

func (h *Handler) GetGuildMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")

	members, err := h.svc.GetGuildMembers(r.Context(), guildID, userID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "You are not a member of this guild")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get members")
		return
	}

	writeJSON(w, http.StatusOK, members)
}

func (h *Handler) CreateGuild(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}

	guild, err := h.svc.CreateGuild(r.Context(), userID, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create guild")
		return
	}

	writeJSON(w, http.StatusCreated, guild)
}

func (h *Handler) GetUserGuilds(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guilds, err := h.svc.GetUserGuilds(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get guilds")
		return
	}

	writeJSON(w, http.StatusOK, guilds)
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")

	var req struct {
		ExpiresIn *int `json:"expires_in"`
		MaxUses   *int `json:"max_uses"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	invite, err := h.svc.CreateInvite(r.Context(), guildID, userID, req.ExpiresIn, req.MaxUses)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "You are not a member of this guild")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create invite")
		return
	}

	writeJSON(w, http.StatusCreated, invite)
}

func (h *Handler) JoinByInvite(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	code := chi.URLParam(r, "code")

	guild, err := h.svc.JoinByInvite(r.Context(), userID, code)
	if err != nil {
		if errors.Is(err, ErrInviteExpired) {
			writeError(w, http.StatusGone, "invite_expired_or_invalid", "Invite has expired or is invalid")
			return
		}
		if errors.Is(err, ErrGuildBanned) {
			// SECURITY: banned users must not be able to rejoin by consuming an invite.
			writeError(w, http.StatusForbidden, "guild_banned", "You are banned from this guild")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to join guild")
		return
	}

	if h.notifier != nil {
		username, _ := h.svc.GetUsername(r.Context(), userID)
		msg, _ := json.Marshal(map[string]string{
			"type":     "guild.member_joined",
			"guild_id": guild.ID.String(),
			"user_id":  userID,
			"username": username,
		})
		h.notifier.BroadcastToGuild(guild.ID.String(), msg)
	}

	writeJSON(w, http.StatusOK, guild)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")
	targetUserID := chi.URLParam(r, "userId")

	err := h.svc.RemoveMember(r.Context(), guildID, requesterID, targetUserID)
	if err != nil {
		if errors.Is(err, ErrGuildLocked) {
			writeError(w, http.StatusForbidden, "guild_locked", err.Error())
			return
		}
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")

	var req struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}
	if req.Type == "" {
		req.Type = "text"
	}

	ch, err := h.svc.CreateChannel(r.Context(), guildID, userID, req.Name, req.Type)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create channel")
		return
	}

	writeJSON(w, http.StatusCreated, ch)
}

func (h *Handler) GetGuildChannels(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")

	channels, err := h.svc.GetGuildChannels(r.Context(), guildID, userID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "You are not a member of this guild")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get channels")
		return
	}

	writeJSON(w, http.StatusOK, channels)
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")

	var req struct {
		Name        string `json:"name"`
		Permissions int64  `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}

	role, err := h.svc.CreateRole(r.Context(), guildID, userID, req.Name, req.Permissions)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create role")
		return
	}

	writeJSON(w, http.StatusCreated, role)
}

func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")
	targetUserID := chi.URLParam(r, "userId")

	var req struct {
		RoleID string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.RoleID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "role_id is required")
		return
	}

	err := h.svc.AssignRole(r.Context(), guildID, requesterID, targetUserID, req.RoleID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Member or role not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to assign role")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) MuteMember(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")
	targetUserID := chi.URLParam(r, "userId")

	var req struct {
		DurationSeconds int `json:"duration_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.DurationSeconds <= 0 {
		req.DurationSeconds = 300
	}

	err := h.svc.MuteMember(r.Context(), guildID, requesterID, targetUserID, req.DurationSeconds)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to mute member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) BanMember(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	guildID := chi.URLParam(r, "guildId")
	targetUserID := chi.URLParam(r, "userId")

	err := h.svc.BanMember(r.Context(), guildID, requesterID, targetUserID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to ban member")
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
