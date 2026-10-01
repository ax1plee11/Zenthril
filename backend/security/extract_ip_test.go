package security

import (
	"net/http/httptest"
	"testing"
)

// These tests were written while extractIP trusted X-Forwarded-For and
// X-Real-IP unconditionally, and two of them asserted that behaviour as correct.
// That assertion encoded the vulnerability fixed in SEC-401: the resolved value
// keys IPRateLimit, BruteForceProtect and AuthRateLimit, so an untrusted header
// let any caller choose its own rate-limit bucket.
//
// The expectations below are updated to the new contract. The trust behaviour
// they used to assert is still covered, but only for a deployment that declares
// trusted proxies explicitly. See guard_trusted_proxy_test.go.

// SECURITY: with no declared trusted proxies, X-Forwarded-For is ignored and the
// transport address is used.
func TestExtractIP_XForwardedFor_IgnoredWithoutTrustedProxies(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	if got := NewGuard(nil, nil).extractIP(req); got != "192.0.2.1" {
		t.Fatalf("got %q want 192.0.2.1", got)
	}
}

// SECURITY: with no declared trusted proxies, X-Real-IP is ignored as well.
func TestExtractIP_XRealIP_IgnoredWithoutTrustedProxies(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	req.Header.Set("X-Real-IP", "198.51.100.2")
	if got := NewGuard(nil, nil).extractIP(req); got != "192.0.2.1" {
		t.Fatalf("got %q want 192.0.2.1", got)
	}
}

// A deployment behind one declared trusted proxy honours X-Forwarded-For and
// takes the entry that proxy appended, skipping the value the client supplied.
func TestExtractIP_XForwardedFor_TrustedProxy(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	if got := NewGuardWithTrustedProxies(nil, nil, 1).extractIP(req); got != "10.0.0.1" {
		t.Fatalf("got %q want 10.0.0.1", got)
	}
}

// A deployment behind one declared trusted proxy also honours X-Real-IP.
func TestExtractIP_XRealIP_TrustedProxy(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	req.Header.Set("X-Real-IP", "198.51.100.2")
	if got := NewGuardWithTrustedProxies(nil, nil, 1).extractIP(req); got != "198.51.100.2" {
		t.Fatalf("got %q want 198.51.100.2", got)
	}
}

// Unchanged contract: without any forwarded header the transport address is used.
func TestExtractIP_RemoteAddr(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	if got := NewGuard(nil, nil).extractIP(req); got != "192.0.2.1" {
		t.Fatalf("got %q want 192.0.2.1", got)
	}
}
