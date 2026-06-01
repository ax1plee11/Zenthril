package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	LegacyCryptoProtocolVersion = 1
	CryptoProtocolVersion       = 2
	CipherSuiteV2               = "X25519-HKDF-SHA256-AES-256-GCM"
)

type EncryptedPayload struct {
	Ciphertext      string `json:"ciphertext"`
	IV              string `json:"iv"`
	KeyID           string `json:"key_id"`
	Tag             string `json:"tag"`
	ProtocolVersion int    `json:"protocol_version"`
	ChannelID       string `json:"channel_id,omitempty"`
	SenderUserID    string `json:"sender_user_id,omitempty"`
	SenderDeviceID  string `json:"sender_device_id,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	ClientMessageID string `json:"client_message_id,omitempty"`
	CipherSuite     string `json:"cipher_suite,omitempty"`
}

type Message struct {
	ID        uuid.UUID        `json:"id"`
	ChannelID uuid.UUID        `json:"channel_id"`
	AuthorID  uuid.UUID        `json:"author_id"`
	Payload   EncryptedPayload `json:"payload"`
	Edited    bool             `json:"edited"`
	Deleted   bool             `json:"deleted"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
