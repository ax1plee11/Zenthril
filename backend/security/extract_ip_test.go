package security

import (
	"net/http/httptest"
	"testing"
)

func TestExtractIP_XForwardedFor_First(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	if got := extractIP(req); got != "203.0.113.1" {
		t.Fatalf("got %q want 203.0.113.1", got)
	}
}

func TestExtractIP_XRealIP(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "198.51.100.2")
	if got := extractIP(req); got != "198.51.100.2" {
		t.Fatalf("got %q want 198.51.100.2", got)
	}
}

func TestExtractIP_RemoteAddr(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	if got := extractIP(req); got != "192.0.2.1" {
		t.Fatalf("got %q want 192.0.2.1", got)
	}
}
