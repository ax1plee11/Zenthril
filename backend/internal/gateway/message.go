package gateway

import (
	"encoding/json"
	"time"
)

const (
	CommandSubscribeChannel   = "subscribe_channel"
	CommandUnsubscribeChannel = "unsubscribe_channel"
	CommandPing               = "ping"

	EventConnected = "gateway.connected"
	EventError     = "gateway.error"
	EventPong      = "pong"
)

type Envelope struct {
	ID        string          `json:"id,omitempty"`
	Type      string          `json:"type"`
	UserID    string          `json:"user_id,omitempty"`
	GuildID   string          `json:"guild_id,omitempty"`
	ChannelID string          `json:"channel_id,omitempty"`
	Sequence  uint64          `json:"sequence,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	SentAt    time.Time       `json:"sent_at"`
}

type ClientCommand struct {
	Type      string          `json:"type"`
	ChannelID string          `json:"channel_id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func ErrorEnvelope(code, message string) Envelope {
	payload, _ := json.Marshal(map[string]string{
		"code":    code,
		"message": message,
	})
	return Envelope{
		Type:   EventError,
		Data:   payload,
		SentAt: time.Now().UTC(),
	}
}
