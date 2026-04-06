package models

import (
	"time"

	"github.com/google/uuid"
)

// EncryptedPayload содержит зашифрованное содержимое сообщения.
type EncryptedPayload struct {
	Ciphertext string `json:"ciphertext"` // base64
	IV         string `json:"iv"`         // base64
	KeyID      string `json:"key_id"`
}

// Message представляет сообщение в канале.
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
