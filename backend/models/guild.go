package models

import (
	"time"

	"github.com/google/uuid"
)

type Guild struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `json:"owner_id"`
	NodeID    string    `json:"node_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Channel struct {
	ID        uuid.UUID `json:"id"`
	GuildID   uuid.UUID `json:"guild_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

type Role struct {
	ID          uuid.UUID `json:"id"`
	GuildID     uuid.UUID `json:"guild_id"`
	Name        string    `json:"name"`
	Level       int       `json:"level"`
	Permissions int64     `json:"permissions"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	Position    int       `json:"position"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GuildMember struct {
	GuildID    uuid.UUID  `json:"guild_id"`
	UserID     uuid.UUID  `json:"user_id"`
	RoleID     *uuid.UUID `json:"role_id,omitempty"`
	JoinedAt   time.Time  `json:"joined_at"`
	Banned     bool       `json:"banned"`
	MutedUntil *time.Time `json:"muted_until,omitempty"`
}

type RoleAssignment struct {
	GuildID    uuid.UUID  `json:"guild_id"`
	UserID     uuid.UUID  `json:"user_id"`
	RoleID     uuid.UUID  `json:"role_id"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
	AssignedAt time.Time  `json:"assigned_at"`
}

type Invite struct {
	Code      string     `json:"code"`
	GuildID   uuid.UUID  `json:"guild_id"`
	CreatedBy uuid.UUID  `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	Uses      int        `json:"uses"`
}
