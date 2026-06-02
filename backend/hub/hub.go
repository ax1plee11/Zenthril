package hub

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"zenthril-backend/auth"
	"zenthril-backend/metrics"
)

const (
	maxWSMessageBytes          = 64 << 10
	maxWSMessagesPerMinute     = 120
	maxWSUserMessagesPerMinute = 300
	maxWSMalformedMessages     = 5
	wsPongWait                 = 60 * time.Second
	wsPingPeriod               = 45 * time.Second
	wsWriteWait                = 10 * time.Second
)

type ChannelAccessChecker interface {
	UserHasChannelAccess(ctx context.Context, userID, channelID string) (bool, error)
}

func NewUpgrader(allowedOrigins []string, environment string) websocket.Upgrader {
	allow := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allow[origin] = struct{}{}
	}
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return checkWSOrigin(r, allow, environment)
		},
	}
}

func checkWSOrigin(r *http.Request, allow map[string]struct{}, environment string) bool {
	origin := r.Header.Get("Origin")

	// SECURITY-HARDENING: fail closed when WS_ALLOWED_ORIGINS is missing.
	// VULNERABILITY FIXED: an unset allowlist no longer degrades into accepting
	// browser WebSocket upgrades from arbitrary sites.
	if len(allow) == 0 {
		slog.Warn("security websocket origin rejected", "reason", "missing_origin_config", "origin", origin, "environment", environment)
		metrics.Global().IncrementWSRejected()
		return false
	}

	if origin == "" {
		slog.Warn("security websocket origin rejected", "reason", "empty_origin", "environment", environment)
		metrics.Global().IncrementWSRejected()
		return false
	}

	if _, ok := allow[origin]; ok {
		return true
	}
	slog.Warn("security websocket origin rejected", "reason", "origin_not_allowed", "origin", origin, "environment", environment)
	metrics.Global().IncrementWSRejected()
	return false
}

type Client struct {
	UserID   string
	ConnID   string
	Send     chan []byte
	GuildIDs []string
	conn     *websocket.Conn
	hub      *Hub
	limiter  *wsRateLimiter
}

type Hub struct {
	channels      map[string]map[*Client]bool
	users         map[string]map[*Client]bool
	userLimiters  map[string]*wsRateLimiter
	voiceChannels map[string]map[string]bool
	guild         ChannelAccessChecker
	mu            sync.RWMutex
	register      chan *Client
	unregister    chan *Client
}

func NewHub(g ChannelAccessChecker) *Hub {
	return &Hub{
		channels:      make(map[string]map[*Client]bool),
		users:         make(map[string]map[*Client]bool),
		userLimiters:  make(map[string]*wsRateLimiter),
		voiceChannels: make(map[string]map[string]bool),
		guild:         g,
		register:      make(chan *Client, 64),
		unregister:    make(chan *Client, 64),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			metrics.Global().IncrementConnections()
			h.mu.Lock()
			if h.users[client.UserID] == nil {
				h.users[client.UserID] = make(map[*Client]bool)
			}
			if h.userLimiters[client.UserID] == nil {
				h.userLimiters[client.UserID] = newWSRateLimiter(maxWSUserMessagesPerMinute)
			}
			h.users[client.UserID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			if h.unregisterClient(client) {
				metrics.Global().DecrementConnections()
			}
		}
	}
}

func (h *Hub) unregisterClient(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	registered := false
	for channelID, clients := range h.channels {
		if clients[client] {
			registered = true
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
		if clients[client] {
			registered = true
			delete(clients, client)
		}
		if len(clients) == 0 {
			delete(h.users, client.UserID)
			delete(h.userLimiters, client.UserID)
		}
	}

	if registered {
		// RESILIENCE: close the send queue exactly once during connection cleanup.
		close(client.Send)
	}
	return registered
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

func (h *Hub) Broadcast(channelID string, msg []byte) {
	start := time.Now()
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
			metrics.Global().IncrementMessagesSent()
		default:
			h.unregister <- c
		}
	}

	metrics.Global().RecordMessageLatency(time.Since(start))
}

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

func (h *Hub) BroadcastExcept(channelID string, except *Client, msg []byte) {
	h.mu.RLock()
	clients := h.channels[channelID]
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		if c != except {
			targets = append(targets, c)
		}
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

func (h *Hub) BroadcastToGuild(guildID string, msg []byte) {
	h.mu.RLock()
	seen := make(map[*Client]bool)
	for _, clients := range h.users {
		for c := range clients {
			if clientInGuild(c, guildID) {
				seen[c] = true
			}
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

func (h *Hub) SetUserGuilds(userID string, guildIDs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.users[userID] {
		c.GuildIDs = dedupeGuildIDs(guildIDs)
	}
}

func (h *Hub) AddUserToGuild(userID, guildID string) {
	if guildID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.users[userID] {
		if !clientInGuild(c, guildID) {
			c.GuildIDs = append(c.GuildIDs, guildID)
		}
	}
}

func (h *Hub) RemoveUserFromGuild(userID, guildID string) {
	if guildID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.users[userID] {
		next := c.GuildIDs[:0]
		for _, id := range c.GuildIDs {
			if id != guildID {
				next = append(next, id)
			}
		}
		c.GuildIDs = next
	}
}

func dedupeGuildIDs(guildIDs []string) []string {
	seen := make(map[string]struct{}, len(guildIDs))
	out := make([]string, 0, len(guildIDs))
	for _, id := range guildIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func clientInGuild(c *Client, guildID string) bool {
	if guildID == "" {
		return false
	}
	for _, id := range c.GuildIDs {
		if id == guildID {
			return true
		}
	}
	return false
}

type wsEvent struct {
	Type         string          `json:"type"`
	ChannelID    string          `json:"channel_id,omitempty"`
	TargetUserID string          `json:"target_user_id,omitempty"`
	InviteCode   string          `json:"invite_code,omitempty"`
	SDP          json.RawMessage `json:"sdp,omitempty"`
	Candidate    json.RawMessage `json:"candidate,omitempty"`
}

func (evt wsEvent) valid() bool {
	switch evt.Type {
	case "subscribe", "unsubscribe":
		return evt.ChannelID != ""
	case "ping":
		return true
	case "typing":
		return evt.ChannelID != ""
	case "invite.send":
		return evt.TargetUserID != "" && evt.InviteCode != ""
	case "voice.join", "voice.leave":
		return evt.ChannelID != ""
	case "voice.signal":
		return evt.ChannelID != "" && evt.TargetUserID != "" && evt.SDP != nil
	case "voice.ice":
		return evt.ChannelID != "" && evt.TargetUserID != "" && evt.Candidate != nil
	default:
		return false
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			// RESILIENCE: ping/pong prevents dead TCP sessions from lingering forever.
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxWSMessageBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})
	malformedMessages := 0

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Debug("websocket read ended", "user_id", c.UserID, "conn_id", c.ConnID, "error", err)
			}
			return
		}
		metrics.Global().IncrementMessagesReceived()
		// SECURITY-HARDENING: enforce both per-connection and per-user limits to reduce flooding blast radius.
		// VULNERABILITY FIXED: one busy connection or one authenticated account cannot flood the hub unchecked.
		if !c.limiter.allow() {
			slog.Warn("security websocket connection rate limited", "user_id", c.UserID, "conn_id", c.ConnID)
			metrics.Global().IncrementWSRateLimitHits()
			c.hub.sendWSError(c, "rate_limited", "too many websocket messages")
			return
		}
		if !c.hub.allowUserMessage(c.UserID) {
			slog.Warn("security websocket user rate limited", "user_id", c.UserID, "conn_id", c.ConnID)
			metrics.Global().IncrementWSRateLimitHits()
			c.hub.sendWSError(c, "rate_limited", "too many websocket messages")
			return
		}

		var evt wsEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			malformedMessages++
			slog.Warn("security malformed websocket message", "user_id", c.UserID, "conn_id", c.ConnID, "count", malformedMessages)
			metrics.Global().IncrementWSMalformed()
			c.hub.sendWSError(c, "malformed_message", "invalid websocket message")
			if malformedMessages >= maxWSMalformedMessages {
				return
			}
			continue
		}
		if !evt.valid() {
			malformedMessages++
			// SECURITY-HARDENING: unknown or incomplete commands are counted as malformed to slow flooding.
			slog.Warn("security invalid websocket command", "user_id", c.UserID, "conn_id", c.ConnID, "type", evt.Type, "count", malformedMessages)
			metrics.Global().IncrementWSMalformed()
			c.hub.sendWSError(c, "malformed_message", "invalid websocket command")
			if malformedMessages >= maxWSMalformedMessages {
				return
			}
			continue
		}
		malformedMessages = 0

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
		case "typing":
			if evt.ChannelID != "" && c.hub.userHasChannelAccess(c.UserID, evt.ChannelID) {
				msg, _ := json.Marshal(map[string]string{
					"type":       "typing",
					"channel_id": evt.ChannelID,
					"user_id":    c.UserID,
				})
				c.hub.BroadcastExcept(evt.ChannelID, c, msg)
			}
		case "invite.send":
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
					"channel_id":   evt.ChannelID,
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

type wsRateLimiter struct {
	mu          sync.Mutex
	windowStart time.Time
	count       int
	limit       int
}

func newWSRateLimiter(limit int) *wsRateLimiter {
	return &wsRateLimiter{windowStart: time.Now(), limit: limit}
}

func (l *wsRateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.windowStart) >= time.Minute {
		l.windowStart = now
		l.count = 0
	}
	l.count++
	return l.count <= l.limit
}

func (h *Hub) allowUserMessage(userID string) bool {
	h.mu.Lock()
	limiter := h.userLimiters[userID]
	if limiter == nil {
		limiter = newWSRateLimiter(maxWSUserMessagesPerMinute)
		h.userLimiters[userID] = limiter
	}
	h.mu.Unlock()
	return limiter.allow()
}

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

func ServeWS(h *Hub, authSvc *auth.Service, upgrader websocket.Upgrader, w http.ResponseWriter, r *http.Request) {
	// SECURITY-HARDENING: reject cross-site WebSocket upgrades before consuming
	// one-time tickets. This prevents CSWSH and avoids burning valid tickets on
	// requests from untrusted browser origins.
	// VULNERABILITY FIXED: CSWSH attempts are denied before authentication state is touched.
	if upgrader.CheckOrigin != nil && !upgrader.CheckOrigin(r) {
		http.Error(w, "websocket origin not allowed", http.StatusForbidden)
		return
	}

	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		metrics.Global().IncrementWSRejected()
		http.Error(w, "missing ticket", http.StatusUnauthorized)
		return
	}

	userID, err := authSvc.ConsumeWSTicket(r.Context(), ticket)
	if err != nil {
		metrics.Global().IncrementWSRejected()
		http.Error(w, "invalid or expired ticket", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("websocket upgrade failed", "error", err)
		return
	}

	client := &Client{
		UserID:  userID,
		ConnID:  r.Header.Get("X-Request-Id"),
		Send:    make(chan []byte, 256),
		conn:    conn,
		hub:     h,
		limiter: newWSRateLimiter(maxWSMessagesPerMinute),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}
