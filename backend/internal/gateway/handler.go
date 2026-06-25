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
					// SECURITY-HARDENING: the next-gen gateway also fails closed without WS_ALLOWED_ORIGINS.
					// VULNERABILITY FIXED: missing origin configuration cannot silently open the gateway.
					opts.Logger.Warn("security gateway websocket origin rejected", "reason", "missing_origin_config")
					return false
				}
				// SECURITY-HARDENING: when an allowlist exists, empty or unknown origins are rejected to prevent CSWSH.
				// VULNERABILITY FIXED: hostile pages cannot reuse a victim browser session to open this socket.
				origin := r.Header.Get("Origin")
				if origin == "" {
					opts.Logger.Warn("security gateway websocket origin rejected", "reason", "empty_origin")
					return false
				}
				_, ok := allowedOrigins[origin]
				if !ok {
					opts.Logger.Warn("security gateway websocket origin rejected", "reason", "origin_not_allowed", "origin", origin)
				}
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

	// SECURITY-HARDENING: reject untrusted WebSocket origins before authentication work.
	// This prevents CSWSH attempts from using the gateway as an auth oracle.
	// VULNERABILITY FIXED: origin validation happens before token validation or socket upgrade.
	if h.upgrader.CheckOrigin != nil && !h.upgrader.CheckOrigin(r) {
		http.Error(w, "websocket origin not allowed", http.StatusForbidden)
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
	clientIP := clientIPFromRequest(r)
	conn := NewConnection(connectionID, claims.UserID, claims.DeviceID, h.nodeID, clientIP, 256)
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
	connectionLimiter := newRateLimiter(gatewayConnectionMessagesPerMinute)
	malformedMessages := 0
	for {
		var command ClientCommand
		if err := socket.ReadJSON(&command); err != nil {
			h.logger.Warn("security gateway malformed websocket message", "connection_id", conn.ID, "user_id", conn.UserID, "error", err)
			return err
		}
		// SECURITY-HARDENING: enforce both per-connection and per-user command limits.
		// VULNERABILITY FIXED: flooding is bounded at connection and account scope.
		if !connectionLimiter.Allow() {
			h.logger.Warn("security gateway connection rate limited", "connection_id", conn.ID, "user_id", conn.UserID)
			h.sendError(conn, "rate_limited", "too many websocket commands")
			return nil
		}
		if !h.registry.AllowUserMessage(ctx, conn.UserID) {
			h.logger.Warn("security gateway user rate limited", "connection_id", conn.ID, "user_id", conn.UserID)
			h.sendError(conn, "rate_limited", "too many websocket commands")
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch command.Type {
		case CommandSubscribeChannel:
			if command.ChannelID == "" {
				malformedMessages++
				// SECURITY-HARDENING: incomplete commands count toward malformed flooding limits.
				if malformedMessages >= 5 {
					return nil
				}
				h.sendError(conn, "bad_request", "channel_id is required")
				continue
			}
			malformedMessages = 0
			if err := h.registry.Subscribe(ctx, conn.ID, command.ChannelID, conn.UserID); err != nil {
				if errors.Is(err, ErrChannelAccessDenied) {
					h.sendError(conn, "forbidden", "no access to this channel")
					continue
				}
				h.sendError(conn, "subscribe_failed", err.Error())
			}
		case CommandUnsubscribeChannel:
			if command.ChannelID == "" {
				malformedMessages++
				if malformedMessages >= 5 {
					return nil
				}
				h.sendError(conn, "bad_request", "channel_id is required")
				continue
			}
			malformedMessages = 0
			if err := h.registry.Unsubscribe(conn.ID, command.ChannelID); err != nil {
				h.sendError(conn, "unsubscribe_failed", err.Error())
			}
		case CommandPing:
			malformedMessages = 0
			h.sendEnvelope(conn, Envelope{Type: EventPong, SentAt: time.Now().UTC()})
		default:
			malformedMessages++
			h.logger.Warn("security gateway invalid command", "connection_id", conn.ID, "user_id", conn.UserID, "type", command.Type, "count", malformedMessages)
			if malformedMessages >= 5 {
				return nil
			}
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
	// SECURITY: next-gen gateway accepts credentials only from Authorization header.
	// Query-string credentials are avoided because URLs are commonly logged by
	// proxies, browsers, and observability tools. Legacy `/ws` uses one-time
	// tickets in its own handler and consumes them after Origin validation.
	// WEAKNESS FIXED: long-lived bearer credentials are no longer accepted from URLs.
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

func clientIPFromRequest(r *http.Request) string {
	// SECURITY-HARDENING: X-Forwarded-For is only trusted when the connection
	// arrives from a known reverse proxy (TrustedProxies config or middleware.RealIP).
	// Accepting the header unconditionally lets any client spoof their IP address,
	// bypassing per-IP connection limits and rate limiting.
	//
	// This function intentionally ignores X-Forwarded-For and X-Real-IP headers
	// because the gateway does not know how many trusted proxy hops sit in front
	// of it. IP extraction from those headers is the responsibility of a dedicated
	// reverse-proxy middleware (e.g. chi middleware.RealIP configured with a known
	// proxy CIDR) that runs before this handler and writes the resolved IP into
	// r.RemoteAddr. If that middleware is deployed, RemoteAddr already contains
	// the correct client IP and no further processing is needed here.
	//
	// VULNERABILITY FIXED: clients can no longer set X-Forwarded-For: 127.0.0.1
	// to bypass the per-IP connection limit in the gateway Registry.
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}
