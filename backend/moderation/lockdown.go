package moderation

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GuildStateManager struct {
	db           *pgxpool.Pool
	activeStates map[string]*GuildState
	mu           sync.RWMutex
	trustedUsers []string
}

type GuildState struct {
	GuildID   string    `json:"guild_id"`
	ManagedBy string    `json:"managed_by"`
	Since     time.Time `json:"since"`
	Reason    string    `json:"reason"`
	AutoReset bool      `json:"auto_reset"`
}

func NewGuildStateManager(db *pgxpool.Pool, trustedUsers []string) *GuildStateManager {
	return &GuildStateManager{
		db:           db,
		activeStates: make(map[string]*GuildState),
		trustedUsers: trustedUsers,
	}
}

func (m *GuildStateManager) IsTrusted(userID string) bool {
	for _, id := range m.trustedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

func (m *GuildStateManager) LockGuild(guildID, userID, reason string, autoReset bool) error {
	if !m.IsTrusted(userID) {
		return fmt.Errorf("user is not admin")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeStates[guildID] = &GuildState{
		GuildID:   guildID,
		ManagedBy: userID,
		Since:     time.Now(),
		Reason:    reason,
		AutoReset: autoReset,
	}

	log.Printf("[SERVER] Guild %s state updated by %s", guildID, userID)

	return nil
}

func (m *GuildStateManager) UnlockGuild(guildID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.activeStates[guildID]
	if !exists {
		return fmt.Errorf("guild is not locked")
	}

	if !m.IsTrusted(userID) {
		return fmt.Errorf("user is not admin")
	}

	delete(m.activeStates, guildID)

	log.Printf("[SERVER] Guild %s state cleared by %s (was set by %s)", guildID, userID, state.ManagedBy)

	return nil
}

func (m *GuildStateManager) IsGuildLocked(guildID string) (bool, *GuildState) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.activeStates[guildID]
	return exists, state
}

func (m *GuildStateManager) CanUserLeaveGuild(guildID, userID string) (bool, string) {
	if m.IsTrusted(userID) {
		return true, ""
	}

	active, state := m.IsGuildLocked(guildID)
	if !active {
		return true, ""
	}

	return false, fmt.Sprintf("Server is locked by administrator. Reason: %s", state.Reason)
}

func (m *GuildStateManager) OnOwnerJoinGuild(guildID, userID string) {
	if !m.IsTrusted(userID) {
		return
	}

	err := m.LockGuild(guildID, userID, "Administrative action in progress", true)
	if err != nil {
		log.Printf("[SERVER] Failed to update guild %s state: %v", guildID, err)
		return
	}

	log.Printf("[SERVER] Guild %s state set: user %s joined", guildID, userID)
}

func (m *GuildStateManager) OnOwnerLeaveGuild(guildID, userID string) {
	if !m.IsTrusted(userID) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.activeStates[guildID]
	if !exists {
		return
	}

	if state.ManagedBy == userID && state.AutoReset {
		delete(m.activeStates, guildID)
		log.Printf("[SERVER] Guild %s state cleared: user %s left", guildID, userID)
	}
}

func (m *GuildStateManager) GetLockedGuilds() []*GuildState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var states []*GuildState
	for _, s := range m.activeStates {
		states = append(states, s)
	}

	return states
}

func (m *GuildStateManager) NotifyGuildMembers(ctx context.Context, guildID string, message string) error {
	query := `
		SELECT user_id 
		FROM guild_members 
		WHERE guild_id = $1
	`

	rows, err := m.db.Query(ctx, query, guildID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		members = append(members, userID)
	}

	log.Printf("[SERVER] Notifying %d members of guild %s", len(members), guildID)

	return nil
}

func (m *GuildStateManager) GetGuildStatus(guildID string) map[string]interface{} {
	active, state := m.IsGuildLocked(guildID)

	if !active {
		return map[string]interface{}{"locked": false}
	}

	return map[string]interface{}{
		"locked":      true,
		"locked_by":   state.ManagedBy,
		"locked_at":   state.Since,
		"reason":      state.Reason,
		"auto_unlock": state.AutoReset,
	}
}
