package benchmarks

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// BenchmarkWebSocketThroughput тестирует пропускную способность WebSocket
func BenchmarkWebSocketThroughput(b *testing.B) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Создаём тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Эхо-сервер
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// Подключаемся к серверу
	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	message := []byte("test message for benchmarking")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			b.Fatal(err)
		}

		_, _, err := conn.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWebSocketConcurrentConnections тестирует множественные одновременные подключения
func BenchmarkWebSocketConcurrentConnections(b *testing.B) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	concurrencyLevels := []int{10, 50, 100, 500}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("connections_%d", concurrency), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(concurrency)

				for j := 0; j < concurrency; j++ {
					go func() {
						defer wg.Done()

						wsURL := "ws" + server.URL[4:]
						conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
						if err != nil {
							return
						}
						defer conn.Close()

						message := []byte("test")
						conn.WriteMessage(websocket.TextMessage, message)
						conn.ReadMessage()
					}()
				}

				wg.Wait()
			}
		})
	}
}

// BenchmarkWebSocketLatency измеряет задержку WebSocket
func BenchmarkWebSocketLatency(b *testing.B) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	message := []byte("latency test")
	latencies := make([]time.Duration, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			b.Fatal(err)
		}

		_, _, err := conn.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}

		latencies[i] = time.Since(start)
	}
	b.StopTimer()

	// Вычисляем статистику
	if b.N > 0 {
		p50, p95, p99 := calculateLatencyPercentiles(latencies)
		b.ReportMetric(float64(p50)/1e6, "p50_ms")
		b.ReportMetric(float64(p95)/1e6, "p95_ms")
		b.ReportMetric(float64(p99)/1e6, "p99_ms")
	}
}

func calculateLatencyPercentiles(latencies []time.Duration) (p50, p95, p99 time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	// Простая сортировка
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

	return sorted[p50Idx], sorted[p95Idx], sorted[p99Idx]
}

// TestWebSocketStressTest стресс-тест для WebSocket (не бенчмарк, но полезен для исследования)
func TestWebSocketStressTest(t *testing.T) {
	if os.Getenv("RUN_STRESS_TESTS") != "1" {
		t.Skip("Skipping stress test by default; set RUN_STRESS_TESTS=1 to run it")
	}
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("Skipping stress test in CI environment")
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// Симулируем 1000 одновременных подключений
	const numConnections = 1000
	const messagesPerConnection = 100

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	errors := make(chan error, numConnections)

	start := time.Now()

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			wsURL := "ws" + server.URL[4:]
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				errors <- fmt.Errorf("connection %d failed: %w", id, err)
				return
			}
			defer conn.Close()

			for j := 0; j < messagesPerConnection; j++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				message := []byte(fmt.Sprintf("message %d from connection %d", j, id))
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					errors <- fmt.Errorf("write failed: %w", err)
					return
				}

				_, _, err := conn.ReadMessage()
				if err != nil {
					errors <- fmt.Errorf("read failed: %w", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalMessages := numConnections * messagesPerConnection

	errorCount := 0
	for err := range errors {
		t.Logf("Error: %v", err)
		errorCount++
	}

	t.Logf("Stress test completed:")
	t.Logf("  Connections: %d", numConnections)
	t.Logf("  Messages per connection: %d", messagesPerConnection)
	t.Logf("  Total messages: %d", totalMessages)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f msg/sec", float64(totalMessages)/duration.Seconds())
	t.Logf("  Errors: %d", errorCount)

	if errorCount > numConnections/2 {
		t.Errorf("Too many errors: %d/%d", errorCount, numConnections)
	}
}
