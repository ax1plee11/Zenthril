package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestClientIPFromRequestIgnoresXForwardedFor verifies that a client cannot spoof
// their IP by setting the X-Forwarded-For header.
//
// SECURITY: this is the regression test for the XFF IP-spoofing vulnerability.
// If clientIPFromRequest is changed to trust X-Forwarded-For again without
// a trusted-proxy mechanism, this test will catch the regression.
func TestClientIPFromRequestIgnoresXForwardedFor(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	// Attacker tries to claim they are 127.0.0.1 to bypass IP rate limits.
	req.Header.Set("X-Forwarded-For", "127.0.0.1")

	ip := clientIPFromRequest(req)
	if ip == "127.0.0.1" {
		t.Fatal("clientIPFromRequest must not trust X-Forwarded-For; got spoofed IP 127.0.0.1")
	}
	if ip != "192.168.1.100" {
		t.Fatalf("expected RemoteAddr IP 192.168.1.100, got %q", ip)
	}
}

// TestClientIPFromRequestIgnoresXRealIP verifies X-Real-IP is also not trusted.
func TestClientIPFromRequestIgnoresXRealIP(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	req.Header.Set("X-Real-IP", "1.2.3.4")

	ip := clientIPFromRequest(req)
	if ip == "1.2.3.4" {
		t.Fatal("clientIPFromRequest must not trust X-Real-IP; got spoofed IP")
	}
	if ip != "10.0.0.5" {
		t.Fatalf("expected 10.0.0.5, got %q", ip)
	}
}

// TestClientIPFromRequestUsesRemoteAddr verifies normal extraction from RemoteAddr.
func TestClientIPFromRequestUsesRemoteAddr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		remoteAddr string
		wantIP     string
	}{
		{"203.0.113.42:8080", "203.0.113.42"},
		{"[::1]:9090", "[::1]"},
		{"192.0.2.1:443", "192.0.2.1"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.RemoteAddr = tc.remoteAddr
		got := clientIPFromRequest(req)
		if got != tc.wantIP {
			t.Errorf("remoteAddr=%q: got %q, want %q", tc.remoteAddr, got, tc.wantIP)
		}
	}
}

// TestClientIPFromRequestNoPort handles the rare case where RemoteAddr has no port.
func TestClientIPFromRequestNoPort(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.RemoteAddr = "192.0.2.99"

	ip := clientIPFromRequest(req)
	// When there's no colon the full string is returned as the address.
	if ip != "192.0.2.99" {
		t.Fatalf("expected 192.0.2.99, got %q", ip)
	}
}
