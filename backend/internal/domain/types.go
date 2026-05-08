package domain

import "time"

type UserID string
type GuildID string
type ChannelID string
type MessageID string
type DeviceID string
type EventID string

type User struct {
	ID        UserID
	Username  string
	CreatedAt time.Time
}

type Guild struct {
	ID        GuildID
	OwnerID   UserID
	Name      string
	CreatedAt time.Time
}

type ChannelType string

const (
	ChannelTypeText  ChannelType = "text"
	ChannelTypeVoice ChannelType = "voice"
)

type Channel struct {
	ID        ChannelID
	GuildID   GuildID
	Name      string
	Type      ChannelType
	CreatedAt time.Time
}

type Device struct {
	ID                 DeviceID
	UserID             UserID
	IdentityPublicKey  []byte
	SignedPreKey       []byte
	SignedPreKeySig    []byte
	OneTimePreKeyCount int
	CreatedAt          time.Time
	LastSeenAt         time.Time
}
