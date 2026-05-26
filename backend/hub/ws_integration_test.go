package hub

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSRejectsBadOrigin(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := NewUpgrader([]string{"https://app.example.com"}, "test")
		_, _ = upgrader.Upgrade(w, r, nil)
	}))
	defer server.Close()

	headers := http.Header{"Origin": []string{"https://evil.example.com"}}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), headers)
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("bad Origin websocket dial succeeded")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %v, want %d", statusCode(resp), http.StatusForbidden)
	}
}

func TestWSAcceptsAllowedOriginAndPing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := NewUpgrader([]string{"https://app.example.com"}, "test")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var msg map[string]string
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg["type"] == "ping" {
			_ = conn.WriteJSON(map[string]string{"type": "pong"})
		}
	}))
	defer server.Close()

	headers := http.Header{"Origin": []string{"https://app.example.com"}}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), headers)
	if err != nil {
		t.Fatalf("allowed Origin websocket dial failed: %v status=%v", err, statusCode(resp))
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	var msg map[string]string
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read pong: %v", err)
	}
	if msg["type"] != "pong" {
		t.Fatalf("message type = %q, want pong", msg["type"])
	}
}

func TestWSRejectsMissingOrigin(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := NewUpgrader([]string{"https://app.example.com"}, "test")
		_, _ = upgrader.Upgrade(w, r, nil)
	}))
	defer server.Close()

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("missing Origin websocket dial succeeded")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %v, want %d", statusCode(resp), http.StatusForbidden)
	}
}

func TestWSTicketConsumedOnlyOnce(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	valid := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := NewUpgrader([]string{"https://app.example.com"}, "test")
		if !upgrader.CheckOrigin(r) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		mu.Lock()
		ok := valid && r.URL.Query().Get("ticket") == "ticket-1"
		if ok {
			valid = false
		}
		mu.Unlock()
		if !ok {
			http.Error(w, "invalid ticket", http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			_ = conn.Close()
		}
	}))
	defer server.Close()

	headers := http.Header{"Origin": []string{"https://app.example.com"}}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL)+"?ticket=ticket-1", headers)
	if err != nil {
		t.Fatalf("first ticket dial failed: %v status=%v", err, statusCode(resp))
	}
	_ = conn.Close()

	conn, resp, err = websocket.DefaultDialer.Dial(wsURL(server.URL)+"?ticket=ticket-1", headers)
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("reused websocket ticket was accepted")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want %d", statusCode(resp), http.StatusUnauthorized)
	}
}

func TestWSRateLimitClosesNoisyConn(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := NewUpgrader([]string{"https://app.example.com"}, "test")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for i := 0; ; i++ {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			if i >= 2 {
				_ = conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "rate limited"),
					time.Now().Add(time.Second),
				)
				return
			}
		}
	}))
	defer server.Close()

	headers := http.Header{"Origin": []string{"https://app.example.com"}}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), headers)
	if err != nil {
		t.Fatalf("websocket dial failed: %v status=%v", err, statusCode(resp))
	}
	defer conn.Close()

	for i := 0; i < 4; i++ {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`))
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if closeErr, ok := err.(*websocket.CloseError); !ok || closeErr.Code != websocket.ClosePolicyViolation {
		t.Fatalf("close error = %v, want policy violation close", err)
	}
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func statusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
