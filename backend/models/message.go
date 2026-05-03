package models

import (
	"time"

	"github.com/google/uuid"
)

type EncryptedPayload struct {
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	KeyID      string `json:"key_id"`
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
