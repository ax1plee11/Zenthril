package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
)

var (
	ErrDraining              = errors.New("gateway is draining")
	ErrConnectionMissing     = errors.New("connection missing")
	ErrConnectionLimit       = errors.New("connection limit reached")
	ErrUserConnectionLimit   = errors.New("user connection limit reached")
	ErrIPConnectionLimit     = errors.New("ip connection limit reached")
	ErrChannelAccessDenied   = errors.New("channel access denied")
)

type RegistryOptions struct {
	NodeID                string
	MaxConnections        int
	MaxConnectionsPerUser int
	Logger                *slog.Logger
	ChannelAccess         ChannelAccessChecker
	ConnectionGuard       *ConnectionGuard
	DistributedLimiter    DistributedRateLimiter
}

type Registry struct {
	nodeID                string
	maxConnections        int
	maxConnectionsPerUser int
	logger                *slog.Logger
	channelAccess         ChannelAccessChecker
	connectionGuard       *ConnectionGuard
	distributedLimiter    DistributedRateLimiter

	draining   atomic.Bool
	mu         sync.RWMutex
	byID       map[string]*Connection
	byUser     map[string]map[string]*Connection
	byUserRate map[string]*rateLimiter
	byChannel  map[string]map[string]*Connection
	byIP       map[string]int
}

type Connection struct {
	ID       string
	UserID   string
	DeviceID string
	NodeID   string
	ClientIP string

	send          chan []byte
	done          chan struct{}
	closeOnce     sync.Once
	subscriptions map[string]struct{}
}

type Stats struct {
	NodeID      string `json:"node_id"`
	Draining    bool   `json:"draining"`
	Connections int    `json:"connections"`
	Users       int    `json:"users"`
	Channels    int    `json:"channels"`
}

func NewRegistry(opts RegistryOptions) *Registry {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.MaxConnections <= 0 {
		opts.MaxConnections = 50000
	}
	if opts.MaxConnectionsPerUser <= 0 {
		opts.MaxConnectionsPerUser = 5
	}
	if opts.ConnectionGuard == nil {
		opts.ConnectionGuard = NewConnectionGuard(20, opts.MaxConnectionsPerUser)
	}
	return &Registry{
		nodeID:                opts.NodeID,
		maxConnections:        opts.MaxConnections,
		maxConnectionsPerUser: opts.MaxConnectionsPerUser,
		logger:                opts.Logger,
		channelAccess:         opts.ChannelAccess,
		connectionGuard:       opts.ConnectionGuard,
		distributedLimiter:    opts.DistributedLimiter,
		byID:                  make(map[string]*Connection),
		byUser:                make(map[string]map[string]*Connection),
		byUserRate:            make(map[string]*rateLimiter),
		byChannel:             make(map[string]map[string]*Connection),
		byIP:                  make(map[string]int),
	}
}

func NewConnection(id, userID, deviceID, nodeID, clientIP string, queueSize int) *Connection {
	if queueSize <= 0 {
		queueSize = 256
	}
	return &Connection{
		ID:            id,
		UserID:        userID,
		DeviceID:      deviceID,
		NodeID:        nodeID,
		ClientIP:      clientIP,
		send:          make(chan []byte, queueSize),
		done:          make(chan struct{}),
		subscriptions: make(map[string]struct{}),
	}
}

func (r *Registry) Register(conn *Connection) error {
	if r.draining.Load() {
		return ErrDraining
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.byID) >= r.maxConnections {
		return ErrConnectionLimit
	}

	ipCount := r.byIP[conn.ClientIP]
	if !r.connectionGuard.AllowIP(conn.ClientIP, ipCount) {
		r.logger.Warn("security gateway ip connection limit", "ip", conn.ClientIP, "count", ipCount)
		return ErrIPConnectionLimit
	}

	userCount := len(r.byUser[conn.UserID])
	if !r.connectionGuard.AllowUser(conn.UserID, userCount) {
		r.logger.Warn("security gateway user connection limit", "user_id", conn.UserID, "count", userCount)
		return ErrUserConnectionLimit
	}

	r.byID[conn.ID] = conn
	if r.byUser[conn.UserID] == nil {
		r.byUser[conn.UserID] = make(map[string]*Connection)
	}
	if r.byUserRate[conn.UserID] == nil {
		r.byUserRate[conn.UserID] = newRateLimiter(gatewayUserMessagesPerMinute)
	}
	r.byUser[conn.UserID][conn.ID] = conn
	if conn.ClientIP != "" {
		r.byIP[conn.ClientIP]++
	}
	return nil
}

func (r *Registry) Unregister(connectionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, ok := r.byID[connectionID]
	if !ok {
		return
	}
	delete(r.byID, connectionID)
	if userConnections := r.byUser[conn.UserID]; userConnections != nil {
		delete(userConnections, connectionID)
		if len(userConnections) == 0 {
			delete(r.byUser, conn.UserID)
			delete(r.byUserRate, conn.UserID)
		}
	}
	if conn.ClientIP != "" {
		if count := r.byIP[conn.ClientIP]; count <= 1 {
			delete(r.byIP, conn.ClientIP)
		} else {
			r.byIP[conn.ClientIP]--
		}
	}
	for channelID := range conn.subscriptions {
		if channelConnections := r.byChannel[channelID]; channelConnections != nil {
			delete(channelConnections, connectionID)
			if len(channelConnections) == 0 {
				delete(r.byChannel, channelID)
			}
		}
	}
	conn.close()
}

func (r *Registry) AllowUserMessage(ctx context.Context, userID string) bool {
	if r.distributedLimiter != nil {
		allowed, err := r.distributedLimiter.Allow(ctx, userID, gatewayUserMessagesPerMinute, userRateLimitWindow)
		if err != nil {
			r.logger.Warn("security gateway distributed rate limiter failed", "user_id", userID, "error", err)
		} else if !allowed {
			return false
		}
	}

	r.mu.Lock()
	limiter := r.byUserRate[userID]
	if limiter == nil {
		limiter = newRateLimiter(gatewayUserMessagesPerMinute)
		r.byUserRate[userID] = limiter
	}
	r.mu.Unlock()
	return limiter.Allow()
}

func (r *Registry) Subscribe(ctx context.Context, connectionID, channelID, userID string) error {
	if r.channelAccess != nil {
		ok, err := r.channelAccess.UserHasChannelAccess(ctx, userID, channelID)
		if err != nil {
			return err
		}
		if !ok {
			// SECURITY: channel subscriptions require the same ACL as legacy hub.Subscribe.
			// WEAKNESS FIXED: unauthenticated channel eavesdropping via subscribe is blocked.
			r.logger.Warn("security gateway subscribe forbidden", "user_id", userID, "connection_id", connectionID, "channel_id", channelID)
			return ErrChannelAccessDenied
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	conn, ok := r.byID[connectionID]
	if !ok {
		return ErrConnectionMissing
	}
	if r.byChannel[channelID] == nil {
		r.byChannel[channelID] = make(map[string]*Connection)
	}
	conn.subscriptions[channelID] = struct{}{}
	r.byChannel[channelID][connectionID] = conn
	return nil
}

func (r *Registry) Unsubscribe(connectionID, channelID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, ok := r.byID[connectionID]
	if !ok {
		return ErrConnectionMissing
	}
	delete(conn.subscriptions, channelID)
	if channelConnections := r.byChannel[channelID]; channelConnections != nil {
		delete(channelConnections, connectionID)
		if len(channelConnections) == 0 {
			delete(r.byChannel, channelID)
		}
	}
	return nil
}

func (r *Registry) DeliverUser(ctx context.Context, userID string, env Envelope) int {
	payload, err := json.Marshal(env)
	if err != nil {
		r.logger.Error("marshal gateway envelope", "error", err)
		return 0
	}

	r.mu.RLock()
	connections := make([]*Connection, 0, len(r.byUser[userID]))
	for _, conn := range r.byUser[userID] {
		connections = append(connections, conn)
	}
	r.mu.RUnlock()

	return deliver(ctx, connections, payload)
}

func (r *Registry) DeliverChannel(ctx context.Context, channelID string, env Envelope) int {
	payload, err := json.Marshal(env)
	if err != nil {
		r.logger.Error("marshal gateway envelope", "error", err)
		return 0
	}

	r.mu.RLock()
	connections := make([]*Connection, 0, len(r.byChannel[channelID]))
	for _, conn := range r.byChannel[channelID] {
		connections = append(connections, conn)
	}
	r.mu.RUnlock()

	return deliver(ctx, connections, payload)
}

func (r *Registry) StartDraining() {
	r.draining.Store(true)
}

func (r *Registry) Stats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return Stats{
		NodeID:      r.nodeID,
		Draining:    r.draining.Load(),
		Connections: len(r.byID),
		Users:       len(r.byUser),
		Channels:    len(r.byChannel),
	}
}

func (c *Connection) Send() <-chan []byte {
	return c.send
}

func (c *Connection) Done() <-chan struct{} {
	return c.done
}

func (c *Connection) close() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}

func deliver(ctx context.Context, connections []*Connection, payload []byte) int {
	delivered := 0
	for _, conn := range connections {
		select {
		case <-ctx.Done():
			return delivered
		case <-conn.done:
		case conn.send <- payload:
			delivered++
		default:
		}
	}
	return delivered
}
