package hub

import (
	"net/http/httptest"
	"testing"
)

func TestNewUpgraderRejectsCrossSiteOrigins(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader([]string{"https://app.example.com"}, "production")
	req := httptest.NewRequest("GET", "http://api.example.com/ws", nil)
	req.Header.Set("Origin", "https://evil.example.com")

	if upgrader.CheckOrigin(req) {
		t.Fatal("cross-site websocket origin was accepted")
	}
}

func TestNewUpgraderAcceptsExactAllowedOrigin(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader([]string{"https://app.example.com"}, "production")
	req := httptest.NewRequest("GET", "http://api.example.com/ws", nil)
	req.Header.Set("Origin", "https://app.example.com")

	if !upgrader.CheckOrigin(req) {
		t.Fatal("allowed websocket origin was rejected")
	}
}

func TestNewUpgraderFailsClosedWithoutAllowedOrigins(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader(nil, "production")
	req := httptest.NewRequest("GET", "http://api.example.com/ws", nil)
	req.Header.Set("Origin", "https://app.example.com")

	if upgrader.CheckOrigin(req) {
		t.Fatal("websocket origin was accepted without WS_ALLOWED_ORIGINS")
	}
}

func TestNewUpgraderRejectsMissingOrigin(t *testing.T) {
	t.Parallel()

	upgrader := NewUpgrader([]string{"https://app.example.com"}, "production")
	req := httptest.NewRequest("GET", "http://api.example.com/ws", nil)

	if upgrader.CheckOrigin(req) {
		t.Fatal("websocket request without Origin was accepted")
	}
}
