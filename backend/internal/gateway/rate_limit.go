package gateway

import (
	"sync"
	"time"
)

const (
	gatewayConnectionMessagesPerMinute = 120
	gatewayUserMessagesPerMinute       = 300
	userRateLimitWindow                = time.Minute
)

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
