package gateway

import (
	"sync"
	"time"
)

const (
	gatewayConnectionMessagesPerMinute = 120
	gatewayUserMessagesPerMinute       = 300
	userRateLimitWindow                = time.Minute
	// SECURITY: anti-flooding constants for message-type specific limits.
	maxVoiceSignalsPerMinute  = 30
	maxVoiceICEPerMinute      = 60
	maxInviteSendPerMinute    = 10
	maxTypingEventsPerMinute  = 20
	// SECURITY: connection flood protection.
	maxReconnectAttempts     = 5
	reconnectBackoffSeconds  = 5
	maxConcurrentReconnects  = 100
)

// rateLimiter implements a fixed-window rate limiter.
type rateLimiter struct {
	mu          sync.Mutex
	windowStart time.Time
	count       int
	limit       int
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{windowStart: time.Now(), limit: limit}
}

func (l *rateLimiter) Allow() bool {
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

// SlidingWindowRateLimiter implements a sliding-window rate limiter
// for more precise rate control than fixed-window.
// SECURITY: prevents boundary-exploit flooding at window edges.
// WEAKNESS FIXED: only fixed-window rate limiting existed.
type SlidingWindowRateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	events   []time.Time
}

func NewSlidingWindowRateLimiter(window time.Duration, limit int) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		window: window,
		limit:  limit,
		events: make([]time.Time, 0),
	}
}

func (s *SlidingWindowRateLimiter) Allow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.window)

	// Remove expired events.
	i := 0
	for i < len(s.events) && s.events[i].Before(cutoff) {
		i++
	}
	s.events = s.events[i:]

	if len(s.events) >= s.limit {
		return false
	}
	s.events = append(s.events, now)
	return true
}

// AntiFloodConfig holds anti-flooding configuration parameters.
// ARCHITECTURE: centralizes all anti-flooding thresholds.
type AntiFloodConfig struct {
	MaxReconnectAttempts     int
	ReconnectBackoff         time.Duration
	MaxConcurrentReconnects  int
	MaxVoiceSignalsPerMinute int
	MaxVoiceICEPerMinute     int
	MaxInviteSendPerMinute   int
	MaxTypingEventsPerMinute int
}

// DefaultAntiFloodConfig returns the default anti-flooding configuration.
func DefaultAntiFloodConfig() *AntiFloodConfig {
	return &AntiFloodConfig{
		MaxReconnectAttempts:     maxReconnectAttempts,
		ReconnectBackoff:         reconnectBackoffSeconds * time.Second,
		MaxConcurrentReconnects:  maxConcurrentReconnects,
		MaxVoiceSignalsPerMinute: maxVoiceSignalsPerMinute,
		MaxVoiceICEPerMinute:     maxVoiceICEPerMinute,
		MaxInviteSendPerMinute:   maxInviteSendPerMinute,
		MaxTypingEventsPerMinute: maxTypingEventsPerMinute,
	}
}
