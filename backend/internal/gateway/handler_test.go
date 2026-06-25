package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"

	"zenthril-backend/internal/event"
)

type testAuthenticator struct{}

func (testAuthenticator) AuthenticateWebSocket(ctx context.Context, token string) (UserClaims, error) {
	return UserClaims{UserID: "user-1", DeviceID: "device-1"}, nil
}

type countingAuthenticator struct {
	called bool
}

func (a *countingAuthenticator) AuthenticateWebSocket(ctx context.Context, token string) (UserClaims, error) {
	a.called = true
	return UserClaims{UserID: "user-1", DeviceID: "device-1"}, nil
}

type failingChannelAccess struct{}

func (failingChannelAccess) UserHasChannelAccess(context.Context, string, string) (bool, error) {
	return false, errors.New("database password leaked in internal error")
}

func TestGatewayRejectsCrossSiteWebSocketOrigin(t *testing.T) {
	t.Parallel()
	authenticator := &countingAuthenticator{}

	handler := NewHandler(HandlerOptions{
		NodeID:         "node-a",
		Registry:       NewRegistry(RegistryOptions{NodeID: "node-a"}),
		Bus:            event.NewMemoryBus(),
		Authenticator:  authenticator,
		Logger:         slog.Default(),
		AllowedOrigins: []string{"https://app.example.com"},
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	headers := http.Header{}
	headers.Set("Origin", "https://evil.example.com")
	headers.Set("Authorization", "Bearer test-token")

	wsURL := "ws" + server.URL[len("http"):]
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("cross-site websocket origin was accepted")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		if resp == nil {
			t.Fatal("expected forbidden response, got nil response")
		}
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	if authenticator.called {
		t.Fatal("authenticator was called before rejecting cross-site Origin")
	}
}

func TestGatewayExtractTokenIgnoresQueryCredentials(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/ws?ticket=query-token", nil)
	if got := extractToken(req); got != "" {
		t.Fatalf("query token was accepted: %q", got)
	}

	req.Header.Set("Authorization", "Bearer header-token")
	if got := extractToken(req); got != "header-token" {
		t.Fatalf("header token = %q, want header-token", got)
	}
}

func TestGatewayRegistrationErrorIsGeneric(t *testing.T) {
	t.Parallel()
	registry := NewRegistry(RegistryOptions{NodeID: "node-a"})
	registry.StartDraining()
	handler := NewHandler(HandlerOptions{
		NodeID:         "node-a",
		Registry:       registry,
		Bus:            event.NewMemoryBus(),
		Authenticator:  testAuthenticator{},
		Logger:         slog.Default(),
		AllowedOrigins: []string{"https://app.example.com"},
	})

	conn := dialGatewayForTest(t, handler)
	defer conn.Close()

	env := readGatewayEnvelopeForTest(t, conn)
	if env.Type != EventError {
		t.Fatalf("type = %q, want %q", env.Type, EventError)
	}
	payload := decodeGatewayErrorForTest(t, env)
	if payload["message"] != "gateway is unavailable" {
		t.Fatalf("message = %q, want generic gateway unavailable", payload["message"])
	}
	if payload["message"] == ErrDraining.Error() {
		t.Fatal("internal registry error was disclosed")
	}
}

func TestGatewaySubscribeErrorIsGeneric(t *testing.T) {
	t.Parallel()
	handler := NewHandler(HandlerOptions{
		NodeID:         "node-a",
		Registry:       NewRegistry(RegistryOptions{NodeID: "node-a", ChannelAccess: failingChannelAccess{}}),
		Bus:            event.NewMemoryBus(),
		Authenticator:  testAuthenticator{},
		Logger:         slog.Default(),
		AllowedOrigins: []string{"https://app.example.com"},
	})

	conn := dialGatewayForTest(t, handler)
	defer conn.Close()
	_ = readGatewayEnvelopeForTest(t, conn) // connected event

	if err := conn.WriteJSON(ClientCommand{Type: CommandSubscribeChannel, ChannelID: "channel-1"}); err != nil {
		t.Fatalf("write subscribe command: %v", err)
	}
	env := readGatewayEnvelopeForTest(t, conn)
	if env.Type != EventError {
		t.Fatalf("type = %q, want %q", env.Type, EventError)
	}
	payload := decodeGatewayErrorForTest(t, env)
	if payload["message"] != "subscription failed" {
		t.Fatalf("message = %q, want generic subscription failed", payload["message"])
	}
	if payload["message"] == "database password leaked in internal error" {
		t.Fatal("internal subscribe error was disclosed")
	}
}

func dialGatewayForTest(t *testing.T, handler http.Handler) *websocket.Conn {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	headers := http.Header{}
	headers.Set("Origin", "https://app.example.com")
	headers.Set("Authorization", "Bearer test-token")

	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial gateway: %v", err)
	}
	return conn
}

func readGatewayEnvelopeForTest(t *testing.T, conn *websocket.Conn) Envelope {
	t.Helper()
	var env Envelope
	if err := conn.ReadJSON(&env); err != nil {
		t.Fatalf("read gateway envelope: %v", err)
	}
	return env
}

func decodeGatewayErrorForTest(t *testing.T, env Envelope) map[string]string {
	t.Helper()
	var payload map[string]string
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		t.Fatalf("decode gateway error payload: %v", err)
	}
	return payload
}
