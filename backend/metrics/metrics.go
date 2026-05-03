package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	ActiveConnections  int64
	TotalConnections   int64
	TotalMessages      int64
	MessagesSent       int64
	MessagesReceived   int64
	EncryptionOps      int64
	DecryptionOps      int64
	EncryptionTimeNs   int64
	DecryptionTimeNs   int64
	DBQueries          int64
	DBQueryTimeNs      int64
	DBErrors           int64
	HTTPRequests       int64
	HTTPErrors         int64
	HTTPResponseTimeNs int64
	LoginAttempts      int64
	LoginSuccesses     int64
	LoginFailures      int64
	mu                 sync.RWMutex
	messageLatencies   []time.Duration
	queryLatencies     []time.Duration
	httpLatencies      []time.Duration
}

var globalMetrics = &Metrics{
	messageLatencies: make([]time.Duration, 0, 10000),
	queryLatencies:   make([]time.Duration, 0, 10000),
	httpLatencies:    make([]time.Duration, 0, 10000),
}

func Global() *Metrics {
	return globalMetrics
}

func (m *Metrics) IncrementConnections() {
	atomic.AddInt64(&m.ActiveConnections, 1)
	atomic.AddInt64(&m.TotalConnections, 1)
}

func (m *Metrics) DecrementConnections() {
	atomic.AddInt64(&m.ActiveConnections, -1)
}

func (m *Metrics) IncrementMessagesSent() {
	atomic.AddInt64(&m.MessagesSent, 1)
	atomic.AddInt64(&m.TotalMessages, 1)
}

func (m *Metrics) IncrementMessagesReceived() {
	atomic.AddInt64(&m.MessagesReceived, 1)
}

func (m *Metrics) RecordMessageLatency(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messageLatencies) < 10000 {
		m.messageLatencies = append(m.messageLatencies, d)
	}
}

func (m *Metrics) RecordEncryption(duration time.Duration) {
	atomic.AddInt64(&m.EncryptionOps, 1)
	atomic.AddInt64(&m.EncryptionTimeNs, int64(duration))
}

func (m *Metrics) RecordDecryption(duration time.Duration) {
	atomic.AddInt64(&m.DecryptionOps, 1)
	atomic.AddInt64(&m.DecryptionTimeNs, int64(duration))
}

func (m *Metrics) RecordDBQuery(duration time.Duration, err error) {
	atomic.AddInt64(&m.DBQueries, 1)
	atomic.AddInt64(&m.DBQueryTimeNs, int64(duration))
	if err != nil {
		atomic.AddInt64(&m.DBErrors, 1)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.queryLatencies) < 10000 {
		m.queryLatencies = append(m.queryLatencies, duration)
	}
}

func (m *Metrics) RecordHTTPRequest(duration time.Duration, statusCode int) {
	atomic.AddInt64(&m.HTTPRequests, 1)
	atomic.AddInt64(&m.HTTPResponseTimeNs, int64(duration))
	if statusCode >= 400 {
		atomic.AddInt64(&m.HTTPErrors, 1)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.httpLatencies) < 10000 {
		m.httpLatencies = append(m.httpLatencies, duration)
	}
}

func (m *Metrics) RecordLoginAttempt(success bool) {
	atomic.AddInt64(&m.LoginAttempts, 1)
	if success {
		atomic.AddInt64(&m.LoginSuccesses, 1)
	} else {
		atomic.AddInt64(&m.LoginFailures, 1)
	}
}

type Snapshot struct {
	Timestamp          time.Time `json:"timestamp"`
	ActiveConnections  int64     `json:"active_connections"`
	TotalConnections   int64     `json:"total_connections"`
	TotalMessages      int64     `json:"total_messages"`
	MessagesSent       int64     `json:"messages_sent"`
	MessagesReceived   int64     `json:"messages_received"`
	EncryptionOps      int64     `json:"encryption_ops"`
	DecryptionOps      int64     `json:"decryption_ops"`
	AvgEncryptionMs    float64   `json:"avg_encryption_ms"`
	AvgDecryptionMs    float64   `json:"avg_decryption_ms"`
	DBQueries          int64     `json:"db_queries"`
	AvgDBQueryMs       float64   `json:"avg_db_query_ms"`
	DBErrors           int64     `json:"db_errors"`
	HTTPRequests       int64     `json:"http_requests"`
	HTTPErrors         int64     `json:"http_errors"`
	AvgHTTPResponseMs  float64   `json:"avg_http_response_ms"`
	LoginAttempts      int64     `json:"login_attempts"`
	LoginSuccesses     int64     `json:"login_successes"`
	LoginFailures      int64     `json:"login_failures"`
	MessageLatencyP50  float64   `json:"message_latency_p50_ms"`
	MessageLatencyP95  float64   `json:"message_latency_p95_ms"`
	MessageLatencyP99  float64   `json:"message_latency_p99_ms"`
	DBLatencyP50       float64   `json:"db_latency_p50_ms"`
	DBLatencyP95       float64   `json:"db_latency_p95_ms"`
	DBLatencyP99       float64   `json:"db_latency_p99_ms"`
	HTTPLatencyP50     float64   `json:"http_latency_p50_ms"`
	HTTPLatencyP95     float64   `json:"http_latency_p95_ms"`
	HTTPLatencyP99     float64   `json:"http_latency_p99_ms"`
}

func (m *Metrics) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s := Snapshot{
		Timestamp:         time.Now(),
		ActiveConnections: atomic.LoadInt64(&m.ActiveConnections),
		TotalConnections:  atomic.LoadInt64(&m.TotalConnections),
		TotalMessages:     atomic.LoadInt64(&m.TotalMessages),
		MessagesSent:      atomic.LoadInt64(&m.MessagesSent),
		MessagesReceived:  atomic.LoadInt64(&m.MessagesReceived),
		EncryptionOps:     atomic.LoadInt64(&m.EncryptionOps),
		DecryptionOps:     atomic.LoadInt64(&m.DecryptionOps),
		DBQueries:         atomic.LoadInt64(&m.DBQueries),
		DBErrors:          atomic.LoadInt64(&m.DBErrors),
		HTTPRequests:      atomic.LoadInt64(&m.HTTPRequests),
		HTTPErrors:        atomic.LoadInt64(&m.HTTPErrors),
		LoginAttempts:     atomic.LoadInt64(&m.LoginAttempts),
		LoginSuccesses:    atomic.LoadInt64(&m.LoginSuccesses),
		LoginFailures:     atomic.LoadInt64(&m.LoginFailures),
	}

	if s.EncryptionOps > 0 {
		s.AvgEncryptionMs = float64(atomic.LoadInt64(&m.EncryptionTimeNs)) / float64(s.EncryptionOps) / 1e6
	}
	if s.DecryptionOps > 0 {
		s.AvgDecryptionMs = float64(atomic.LoadInt64(&m.DecryptionTimeNs)) / float64(s.DecryptionOps) / 1e6
	}
	if s.DBQueries > 0 {
		s.AvgDBQueryMs = float64(atomic.LoadInt64(&m.DBQueryTimeNs)) / float64(s.DBQueries) / 1e6
	}
	if s.HTTPRequests > 0 {
		s.AvgHTTPResponseMs = float64(atomic.LoadInt64(&m.HTTPResponseTimeNs)) / float64(s.HTTPRequests) / 1e6
	}

	s.MessageLatencyP50, s.MessageLatencyP95, s.MessageLatencyP99 = calculatePercentiles(m.messageLatencies)
	s.DBLatencyP50, s.DBLatencyP95, s.DBLatencyP99 = calculatePercentiles(m.queryLatencies)
	s.HTTPLatencyP50, s.HTTPLatencyP95, s.HTTPLatencyP99 = calculatePercentiles(m.httpLatencies)

	return s
}

func calculatePercentiles(latencies []time.Duration) (p50, p95, p99 float64) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	p50Idx := len(sorted) * 50 / 100
	p95Idx := len(sorted) * 95 / 100
	p99Idx := len(sorted) * 99 / 100

	if p50Idx >= len(sorted) {
		p50Idx = len(sorted) - 1
	}
	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}
	if p99Idx >= len(sorted) {
		p99Idx = len(sorted) - 1
	}

	return float64(sorted[p50Idx]) / 1e6,
		float64(sorted[p95Idx]) / 1e6,
		float64(sorted[p99Idx]) / 1e6
}

func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.ActiveConnections, 0)
	atomic.StoreInt64(&m.TotalConnections, 0)
	atomic.StoreInt64(&m.TotalMessages, 0)
	atomic.StoreInt64(&m.MessagesSent, 0)
	atomic.StoreInt64(&m.MessagesReceived, 0)
	atomic.StoreInt64(&m.EncryptionOps, 0)
	atomic.StoreInt64(&m.DecryptionOps, 0)
	atomic.StoreInt64(&m.EncryptionTimeNs, 0)
	atomic.StoreInt64(&m.DecryptionTimeNs, 0)
	atomic.StoreInt64(&m.DBQueries, 0)
	atomic.StoreInt64(&m.DBQueryTimeNs, 0)
	atomic.StoreInt64(&m.DBErrors, 0)
	atomic.StoreInt64(&m.HTTPRequests, 0)
	atomic.StoreInt64(&m.HTTPErrors, 0)
	atomic.StoreInt64(&m.HTTPResponseTimeNs, 0)
	atomic.StoreInt64(&m.LoginAttempts, 0)
	atomic.StoreInt64(&m.LoginSuccesses, 0)
	atomic.StoreInt64(&m.LoginFailures, 0)

	m.mu.Lock()
	m.messageLatencies = make([]time.Duration, 0, 10000)
	m.queryLatencies = make([]time.Duration, 0, 10000)
	m.httpLatencies = make([]time.Duration, 0, 10000)
	m.mu.Unlock()
}
