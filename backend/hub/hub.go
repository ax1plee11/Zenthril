package hub

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"veltrix-backend/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // в проде ограничить по Origin
	},
}

// Client представляет подключённого WebSocket-клиента.
type Client struct {
	UserID   string
	ConnID   string
	Send     chan []byte
	GuildIDs []string
	conn     *websocket.Conn
	hub      *Hub
}

// Hub управляет подписками клиентов на каналы.
type Hub struct {
	channels      map[string]map[*Client]bool // channelID -> clients
	users         map[string]map[*Client]bool // userID -> clients
	voiceChannels map[string]map[string]bool  // channelID -> set of userIDs
	mu            sync.RWMutex
	register      chan *Client
	unregister    chan *Client
}

// NewHub создаёт новый Hub.
func NewHub() *Hub {
	return &Hub{
		channels:      make(map[string]map[*Client]bool),
		users:         make(map[string]map[*Client]bool),
		voiceChannels: make(map[string]map[string]bool),
		register:      make(chan *Client, 64),
		unregister:    make(chan *Client, 64),
	}
}

// Run запускает горутину обработки register/unregister.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.users[client.UserID] == nil {
				h.users[client.UserID] = make(map[*Client]bool)
			}
			h.users[client.UserID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			// Удаляем из всех каналов
			for channelID, clients := range h.channels {
				if clients[client] {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.channels, channelID)
					}
				}
			}
			// Удаляем из голосовых каналов
			for channelID, users := range h.voiceChannels {
				if users[client.UserID] {
					delete(users, client.UserID)
					if len(users) == 0 {
						delete(h.voiceChannels, channelID)
					}
				}
			}
			// Удаляем из users
			if clients, ok := h.users[client.UserID]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.users, client.UserID)
				}
			}
			h.mu.Unlock()
			close(client.Send)
		}
	}
}

// Subscribe подписывает клиента на канал.
func (h *Hub) Subscribe(client *Client, channelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channels[channelID] == nil {
		h.channels[channelID] = make(map[*Client]bool)
	}
	h.channels[channelID][client] = true
}

// Unsubscribe отписывает клиента от канала.
func (h *Hub) Unsubscribe(client *Client, channelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.channels[channelID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channelID)
		}
	}
}

// Broadcast отправляет сообщение всем клиентам, подписанным на канал.
func (h *Hub) Broadcast(channelID string, msg []byte) {
	h.mu.RLock()
	clients := h.channels[channelID]
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.Send <- msg:
		default:
			// Буфер переполнен — отключаем клиента
			h.unregister <- c
		}
	}
}

// BroadcastToUser отправляет сообщение всем соединениям конкретного пользователя.
func (h *Hub) BroadcastToUser(userID string, msg []byte) {
	h.mu.RLock()
	clients := h.users[userID]
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.Send <- msg:
		default:
			h.unregister <- c
		}
	}
}

// wsEvent — входящее событие от клиента.
type wsEvent struct {
	Type         string          `json:"type"`
	ChannelID    string          `json:"channel_id,omitempty"`
	TargetUserID string          `json:"target_user_id,omitempty"`
	SDP          json.RawMessage `json:"sdp,omitempty"`
	Candidate    json.RawMessage `json:"candidate,omitempty"`
}

// writePump читает из канала Send и пишет в WebSocket.
func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.Send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// readPump читает события от клиента и обрабатывает их.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws read error user=%s: %v", c.UserID, err)
			}
			return
		}

		var evt wsEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			continue
		}

		switch evt.Type {
		case "subscribe":
			if evt.ChannelID != "" {
				c.hub.Subscribe(c, evt.ChannelID)
			}
		case "unsubscribe":
			if evt.ChannelID != "" {
				c.hub.Unsubscribe(c, evt.ChannelID)
			}
		case "ping":
			pong, _ := json.Marshal(map[string]string{"type": "pong"})
			select {
			case c.Send <- pong:
			default:
			}

		case "voice.join":
			if evt.ChannelID != "" {
				c.hub.voiceJoin(c, evt.ChannelID)
			}
		case "voice.leave":
			if evt.ChannelID != "" {
				c.hub.voiceLeave(c, evt.ChannelID)
			}
		case "voice.signal":
			if evt.ChannelID != "" && evt.TargetUserID != "" && evt.SDP != nil {
				msg, _ := json.Marshal(map[string]interface{}{
					"type":       "voice.signal",
					"channel_id": evt.ChannelID,
					"from_user_id": c.UserID,
					"sdp":        evt.SDP,
				})
				c.hub.BroadcastToUser(evt.TargetUserID, msg)
			}
		case "voice.ice":
			if evt.ChannelID != "" && evt.TargetUserID != "" && evt.Candidate != nil {
				msg, _ := json.Marshal(map[string]interface{}{
					"type":         "voice.ice",
					"channel_id":   evt.ChannelID,
					"from_user_id": c.UserID,
					"candidate":    evt.Candidate,
				})
				c.hub.BroadcastToUser(evt.TargetUserID, msg)
			}
		}
	}
}

// voiceJoin добавляет клиента в голосовой канал и рассылает voice.user_joined.
func (h *Hub) voiceJoin(c *Client, channelID string) {
	h.mu.Lock()
	if h.voiceChannels[channelID] == nil {
		h.voiceChannels[channelID] = make(map[string]bool)
	}
	h.voiceChannels[channelID][c.UserID] = true
	h.mu.Unlock()

	msg, _ := json.Marshal(map[string]string{
		"type":       "voice.user_joined",
		"channel_id": channelID,
		"user_id":    c.UserID,
	})
	h.Broadcast(channelID, msg)
}

// voiceLeave убирает клиента из голосового канала и рассылает voice.user_left.
func (h *Hub) voiceLeave(c *Client, channelID string) {
	h.mu.Lock()
	if users, ok := h.voiceChannels[channelID]; ok {
		delete(users, c.UserID)
		if len(users) == 0 {
			delete(h.voiceChannels, channelID)
		}
	}
	h.mu.Unlock()

	msg, _ := json.Marshal(map[string]string{
		"type":       "voice.user_left",
		"channel_id": channelID,
		"user_id":    c.UserID,
	})
	h.Broadcast(channelID, msg)
}

// ServeWS выполняет WebSocket upgrade и аутентификацию через JWT query param ?token=<jwt>.
func ServeWS(h *Hub, authSvc *auth.Service, w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, err := authSvc.ValidateTokenPublic(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := &Client{
		UserID: userID,
		ConnID: r.Header.Get("X-Request-Id"),
		Send:   make(chan []byte, 256),
		conn:   conn,
		hub:    h,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}
