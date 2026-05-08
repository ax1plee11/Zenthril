package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"zenthril-backend/internal/event"
)

type HandlerOptions struct {
	NodeID         string
	Registry       *Registry
	Bus            event.Bus
	Authenticator  Authenticator
	Logger         *slog.Logger
	AllowedOrigins []string
	ReadLimitBytes int64
	WriteTimeout   time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
}

type Handler struct {
	nodeID        string
	registry      *Registry
	bus           event.Bus
	authenticator Authenticator
	logger        *slog.Logger
	upgrader      websocket.Upgrader
	readLimit     int64
	writeTimeout  time.Duration
	pongWait      time.Duration
	pingPeriod    time.Duration
}

func NewHandler(opts HandlerOptions) *Handler {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.WriteTimeout <= 0 {
		opts.WriteTimeout = 10 * time.Second
	}
	if opts.PongWait <= 0 {
		opts.PongWait = 60 * time.Second
	}
	if opts.PingPeriod <= 0 {
		opts.PingPeriod = (opts.PongWait * 9) / 10
	}
	if opts.ReadLimitBytes <= 0 {
		opts.ReadLimitBytes = 1 << 20
	}

	allowedOrigins := map[string]struct{}{}
	for _, origin := range opts.AllowedOrigins {
		allowedOrigins[origin] = struct{}{}
	}

	return &Handler{
		nodeID:        opts.NodeID,
		registry:      opts.Registry,
		bus:           opts.Bus,
		authenticator: opts.Authenticator,
		logger:        opts.Logger,
		readLimit:     opts.ReadLimitBytes,
		writeTimeout:  opts.WriteTimeout,
		pongWait:      opts.PongWait,
		pingPeriod:    opts.PingPeriod,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				if len(allowedOrigins) == 0 {
					return true
				}
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				_, ok := allowedOrigins[origin]
				return ok
			},
		},
	}
}

func (h *Handler) Start(ctx context.Context) error {
	if h.bus == nil {
		return nil
	}
	return h.bus.Subscribe(ctx, event.Subscription{
		Topic: event.TopicGatewayDeliver,
		Group: h.nodeID,
		Handler: func(ctx context.Context, evt event.Event) error {
			var envelope Envelope
			if err := json.Unmarshal(evt.Data, &envelope); err != nil {
				return err
			}
			if envelope.SentAt.IsZero() {
				envelope.SentAt = time.Now().UTC()
			}
			delivered := 0
			if envelope.UserID != "" {
				delivered += h.registry.DeliverUser(ctx, envelope.UserID, envelope)
			}
			if envelope.ChannelID != "" {
				delivered += h.registry.DeliverChannel(ctx, envelope.ChannelID, envelope)
			}
			h.logger.Debug("gateway event delivered", "event_id", evt.ID, "delivered", delivered)
			return nil
		},
	})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.registry == nil || h.authenticator == nil {
		http.Error(w, "gateway is not ready", http.StatusServiceUnavailable)
		return
	}

	token := extractToken(r)
	claims, err := h.authenticator.AuthenticateWebSocket(r.Context(), token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	socket, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("websocket upgrade failed", "error", err)
		return
	}
	defer socket.Close()

	connectionID := uuid.NewString()
	conn := NewConnection(connectionID, claims.UserID, claims.DeviceID, h.nodeID, 256)
	if err := h.registry.Register(conn); err != nil {
		_ = socket.WriteJSON(ErrorEnvelope("gateway_unavailable", err.Error()))
		return
	}
	defer h.registry.Unregister(connectionID)

	socket.SetReadLimit(h.readLimit)
	_ = socket.SetReadDeadline(time.Now().Add(h.pongWait))
	socket.SetPongHandler(func(string) error {
		return socket.SetReadDeadline(time.Now().Add(h.pongWait))
	})

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	errCh := make(chan error, 2)
	go func() { errCh <- h.writeLoop(ctx, socket, conn) }()
	go func() { errCh <- h.readLoop(ctx, socket, conn) }()

	connected := Envelope{Type: EventConnected, UserID: claims.UserID, SentAt: time.Now().UTC()}
	if payload, err := json.Marshal(connected); err == nil {
		select {
		case conn.send <- payload:
		default:
		}
	}

	err = <-errCh
	cancel()
	if err != nil && !errors.Is(err, context.Canceled) && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		h.logger.Debug("websocket session ended", "connection_id", connectionID, "error", err)
	}
}

func (h *Handler) readLoop(ctx context.Context, socket *websocket.Conn, conn *Connection) error {
	for {
		var command ClientCommand
		if err := socket.ReadJSON(&command); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch command.Type {
		case CommandSubscribeChannel:
			if command.ChannelID == "" {
				h.sendError(conn, "bad_request", "channel_id is required")
				continue
			}
			if err := h.registry.Subscribe(conn.ID, command.ChannelID); err != nil {
				h.sendError(conn, "subscribe_failed", err.Error())
			}
		case CommandUnsubscribeChannel:
			if command.ChannelID == "" {
				h.sendError(conn, "bad_request", "channel_id is required")
				continue
			}
			if err := h.registry.Unsubscribe(conn.ID, command.ChannelID); err != nil {
				h.sendError(conn, "unsubscribe_failed", err.Error())
			}
		case CommandPing:
			h.sendEnvelope(conn, Envelope{Type: EventPong, SentAt: time.Now().UTC()})
		default:
			h.sendError(conn, "unknown_command", "unsupported websocket command")
		}
	}
}

func (h *Handler) writeLoop(ctx context.Context, socket *websocket.Conn, conn *Connection) error {
	ticker := time.NewTicker(h.pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-conn.Done():
			return nil
		case payload := <-conn.Send():
			if err := socket.SetWriteDeadline(time.Now().Add(h.writeTimeout)); err != nil {
				return err
			}
			if err := socket.WriteMessage(websocket.TextMessage, payload); err != nil {
				return err
			}
		case <-ticker.C:
			if err := socket.SetWriteDeadline(time.Now().Add(h.writeTimeout)); err != nil {
				return err
			}
			if err := socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				return err
			}
		}
	}
}

func (h *Handler) sendError(conn *Connection, code, message string) {
	h.sendEnvelope(conn, ErrorEnvelope(code, message))
}

func (h *Handler) sendEnvelope(conn *Connection, env Envelope) {
	if env.SentAt.IsZero() {
		env.SentAt = time.Now().UTC()
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return
	}
	select {
	case conn.send <- payload:
	default:
	}
}

func extractToken(r *http.Request) string {
	if token := strings.TrimSpace(r.URL.Query().Get("ticket")); token != "" {
		return token
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return token
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}
