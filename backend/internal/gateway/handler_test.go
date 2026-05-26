package gateway

import (
	"context"
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
