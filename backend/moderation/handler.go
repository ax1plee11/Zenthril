package moderation

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	moderator    *GuildModerator
	monitor      *ConnectionMonitor
	stateManager *GuildStateManager
}

func NewHandler(moderator *GuildModerator, monitor *ConnectionMonitor, stateManager *GuildStateManager) *Handler {
	return &Handler{
		moderator:    moderator,
		monitor:      monitor,
		stateManager: stateManager,
	}
}

func (h *Handler) ProcessGuildBan(w http.ResponseWriter, r *http.Request) {
	guildID := r.PathValue("guildId")
	if guildID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "guildId is required")
		return
	}

	adminID := r.Context().Value("userID").(string)

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "reason is required")
		return
	}

	report, err := h.moderator.ProcessGuildAction(r.Context(), guildID, adminID, req.Reason)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to ban guild")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) GlobalBanGuildAndMembers(w http.ResponseWriter, r *http.Request) {
	guildID := r.PathValue("guildId")
	if guildID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "guildId is required")
		return
	}

	adminID := r.Context().Value("userID").(string)

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "reason is required")
		return
	}

	report, err := h.moderator.ProcessCascadeAction(r.Context(), guildID, adminID, req.Reason)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to cascade ban")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) GetUserIPs(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "userId is required")
		return
	}

	ips, err := h.monitor.GetUserIPs(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get user data")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"ips":     ips,
	})
}

func (h *Handler) GetGuildMembersIPs(w http.ResponseWriter, r *http.Request) {
	guildID := r.PathValue("guildId")
	if guildID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "guildId is required")
		return
	}

	membersIPs, err := h.monitor.GetGuildMembersIPs(r.Context(), guildID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get guild data")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"guild_id":    guildID,
		"members_ips": membersIPs,
	})
}

func (h *Handler) LockGuild(w http.ResponseWriter, r *http.Request) {
	guildID := r.PathValue("guildId")
	if guildID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "guildId is required")
		return
	}

	adminID := r.Context().Value("userID").(string)

	var req struct {
		Reason    string `json:"reason"`
		AutoReset bool   `json:"auto_reset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = "Administrative action"
		req.AutoReset = true
	}

	err := h.stateManager.LockGuild(guildID, adminID, req.Reason, req.AutoReset)
	if err != nil {
		writeError(w, http.StatusForbidden, "forbidden", err.Error())
		return
	}

	_ = h.stateManager.NotifyGuildMembers(r.Context(), guildID,
		"Server has been locked by administrator. You cannot leave until the lock is removed.")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"locked":     true,
		"guild_id":   guildID,
		"reason":     req.Reason,
		"auto_reset": req.AutoReset,
	})
}

func (h *Handler) UnlockGuild(w http.ResponseWriter, r *http.Request) {
	guildID := r.PathValue("guildId")
	if guildID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "guildId is required")
		return
	}

	adminID := r.Context().Value("userID").(string)

	err := h.stateManager.UnlockGuild(guildID, adminID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	_ = h.stateManager.NotifyGuildMembers(r.Context(), guildID,
		"Server has been unlocked. You can now leave if you wish.")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"locked":   false,
		"guild_id": guildID,
	})
}

func (h *Handler) GetLockStatus(w http.ResponseWriter, r *http.Request) {
	guildID := r.PathValue("guildId")
	if guildID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "guildId is required")
		return
	}

	status := h.stateManager.GetGuildStatus(guildID)
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) GetAllLockedGuilds(w http.ResponseWriter, r *http.Request) {
	states := h.stateManager.GetLockedGuilds()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"guilds": states,
		"count":  len(states),
	})
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
