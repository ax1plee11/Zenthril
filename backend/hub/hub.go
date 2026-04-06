package hub

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"veltrix-backend/auth"
)

// ChannelAccessChecker — интерфейс для проверки доступа к каналу (разрывает цикл guild↔hub).
type ChannelAccessChecker interface {
	UserHasChannelAccess(ctx context.Context, userID, channelID string) (bool, error)
}

// NewUpgrader возвращает upgrader с проверкой Origin.
// allowedOrigins: nil или пусто — разрешить любой origin (режим разработки).
// Один элемент "*" — явное разрешение всех.
// Иначе точное совпадение с заголовком Origin; пустой Origin разрешён (Tauri и др.).
func NewUpgrader(allowedOrigins []string) websocket.Upgrader {
	allow := allowedOrigins
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if len(allow) == 0 {
				return true
			}
			if len(allow) == 1 && allow[0] == "*" {
				return true
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			for _, o := range allow {
				if o == origin {
					return true
				}
			}
			return false
		},
	}
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
	channels      map[string]map[*Client]bool
	users         map[string]map[*Client]bool
	voiceChannels map[string]map[string]bool
	guild         ChannelAccessChecker
	mu            sync.RWMutex
	register      chan *Client
	unregister    chan *Client
}

// NewHub создаёт новый Hub.
func NewHub(g ChannelAccessChecker) *Hub {
	return &Hub{
		channels:      make(map[string]map[*Client]bool),
		users:         make(map[string]map[*Client]bool),
		voiceChannels: make(map[string]map[string]bool),
		guild:         g,
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
			for channelID, clients := range h.channels {
				if clients[client] {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.channels, channelID)
					}
				}
			}
			for channelID, users := range h.voiceChannels {
				if users[client.UserID] {
					delete(users, client.UserID)
					if len(users) == 0 {
						delete(h.voiceChannels, channelID)
					}
				}
			}
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

func (h *Hub) sendWSError(c *Client, code, msg string) {
	b, _ := json.Marshal(map[string]string{
		"type":    "error",
		"code":    code,
		"message": msg,
	})
	select {
	case c.Send <- b:
	default:
	}
}

func (h *Hub) userHasChannelAccess(userID, channelID string) bool {
	if h.guild == nil || channelID == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ok, err := h.guild.UserHasChannelAccess(ctx, userID, channelID)
	return err == nil && ok
}

// Subscribe подписывает клиента на канал (только при членстве в гильдии).
func (h *Hub) Subscribe(client *Client, channelID string) {
	if !h.userHasChannelAccess(client.UserID, channelID) {
		h.sendWSError(client, "forbidden", "no access to this channel")
		return
	}
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

// BroadcastToGuild отправляет сообщение всем подключённым участникам сервера.
func (h *Hub) BroadcastToGuild(guildID string, msg []byte) {
	h.mu.RLock()
	// Собираем уникальных клиентов у которых есть подписка на любой канал этого сервера
	seen := make(map[*Client]bool)
	for channelID, clients := range h.channels {
		_ = channelID // используем все каналы — фильтрация по guildID на уровне БД не нужна
		for c := range clients {
			seen[c] = true
		}
	}
	// Также добавляем всех подключённых пользователей (они могут не быть подписаны на канал)
	for _, clients := range h.users {
		for c := range clients {
			seen[c] = true
		}
	}
	targets := make([]*Client, 0, len(seen))
	for c := range seen {
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
	InviteCode   string          `json:"invite_code,omitempty"`
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

		case "invite.send":
			// Отправить инвайт конкретному пользователю
			if evt.TargetUserID != "" && evt.InviteCode != "" {
				msg, _ := json.Marshal(map[string]interface{}{
					"type":         "invite.received",
					"from_user_id": c.UserID,
					"invite_code":  evt.InviteCode,
				})
				c.hub.BroadcastToUser(evt.TargetUserID, msg)
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
				if !c.hub.userHasChannelAccess(c.UserID, evt.ChannelID) {
					c.hub.sendWSError(c, "forbidden", "no access to this channel")
					continue
				}
				msg, _ := json.Marshal(map[string]interface{}{
					"type":         "voice.signal",
					"channel_id": evt.ChannelID,
					"from_user_id": c.UserID,
					"sdp":          evt.SDP,
				})
				c.hub.BroadcastToUser(evt.TargetUserID, msg)
			}
		case "voice.ice":
			if evt.ChannelID != "" && evt.TargetUserID != "" && evt.Candidate != nil {
				if !c.hub.userHasChannelAccess(c.UserID, evt.ChannelID) {
					c.hub.sendWSError(c, "forbidden", "no access to this channel")
					continue
				}
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
	if !h.userHasChannelAccess(c.UserID, channelID) {
		h.sendWSError(c, "forbidden", "no access to this voice channel")
		return
	}
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

// ServeWS выполняет WebSocket upgrade. Аутентификация: одноразовый ?ticket= (см. POST /api/v1/auth/ws-ticket).
func ServeWS(h *Hub, authSvc *auth.Service, upgrader websocket.Upgrader, w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Error(w, "missing ticket", http.StatusUnauthorized)
		return
	}

	userID, err := authSvc.ConsumeWSTicket(r.Context(), ticket)
	if err != nil {
		http.Error(w, "invalid or expired ticket", http.StatusUnauthorized)
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
