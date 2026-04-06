package models

import (
	"time"

	"github.com/google/uuid"
)

// User представляет зарегистрированного пользователя.
type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
}
