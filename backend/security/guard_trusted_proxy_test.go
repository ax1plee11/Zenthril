package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func requestWithClientIP(remoteAddr, xff, xri string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.RemoteAddr = remoteAddr
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	if xri != "" {
		r.Header.Set("X-Real-IP", xri)
	}
	return r
}

// SECURITY regression: extractIP previously trusted X-Forwarded-For and
// X-Real-IP unconditionally, and that value keyed IPRateLimit,
// BruteForceProtect and AuthRateLimit. Any caller could therefore choose its own
// rate-limit bucket by rotating a request header, which disables brute-force
// protection and the registration, refresh and logout budgets.
func TestExtractIPIgnoresForwardedHeadersByDefault(t *testing.T) {
	t.Parallel()

	g := NewGuard(nil, nil)
	if hops := g.TrustedProxyHops(); hops != 0 {
		t.Fatalf("default TrustedProxyHops = %d, want 0", hops)
	}

	cases := []struct {
		name   string
		xff    string
		xri    string
		remote string
	}{
		{"direct connection with spoofed xff", "203.0.113.9", "", "198.51.100.5:44321"},
		{"direct connection with spoofed xri", "", "203.0.113.9", "198.51.100.5:44321"},
		{"both headers spoofed", "203.0.113.9", "203.0.113.10", "198.51.100.5:44321"},
		{"multiple spoofed xff entries", "203.0.113.9, 203.0.113.10, 203.0.113.11", "", "198.51.100.5:44321"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := g.extractIP(requestWithClientIP(tc.remote, tc.xff, tc.xri))
			if got != "198.51.100.5" {
				t.Fatalf("extractIP = %q, want the transport address 198.51.100.5", got)
			}
		})
	}
}

// SECURITY regression: rotating a header must not create a new rate-limit
// bucket when no trusted proxy is configured. This is the bypass the previous
// implementation permitted.
func TestExtractIPRotationCannotChangeBucket(t *testing.T) {
	t.Parallel()

	g := NewGuard(nil, nil)
	const transport = "198.51.100.5:44321"

	first := g.extractIP(requestWithClientIP(transport, "203.0.113.1", ""))
	second := g.extractIP(requestWithClientIP(transport, "203.0.113.2", ""))
	third := g.extractIP(requestWithClientIP(transport, "", "203.0.113.3"))

	if first != second || second != third {
		t.Fatalf("header rotation changed the identity: %q / %q / %q", first, second, third)
	}
}

// With exactly one trusted proxy, the address that proxy observed is the last
// X-Forwarded-For entry, because each proxy appends one entry.
func TestExtractIPResolvesClientBehindOneTrustedProxy(t *testing.T) {
	t.Parallel()

	g := NewGuardWithTrustedProxies(nil, nil, 1)
	r := requestWithClientIP("192.0.2.7:1234", "198.51.100.5", "")
	if got := g.extractIP(r); got != "198.51.100.5" {
		t.Fatalf("extractIP = %q, want 198.51.100.5", got)
	}
}

// With two trusted proxies the client controls the leading entry, so the value
// the first proxy observed sits one position before the last.
func TestExtractIPSkipsClientControlledEntryWithTwoTrustedProxies(t *testing.T) {
	t.Parallel()

	g := NewGuardWithTrustedProxies(nil, nil, 2)
	r := requestWithClientIP("192.0.2.7:1234", "203.0.113.9, 198.51.100.5, 192.0.2.7", "")
	if got := g.extractIP(r); got != "198.51.100.5" {
		t.Fatalf("extractIP = %q, want 198.51.100.5", got)
	}
}

// SECURITY: a chain shorter than the declared number of proxies carries no
// trustworthy value and must fall back to the transport address rather than
// returning a client-controlled entry.
func TestExtractIPFallsBackWhenChainTooShort(t *testing.T) {
	t.Parallel()

	g := NewGuardWithTrustedProxies(nil, nil, 3)
	r := requestWithClientIP("192.0.2.7:1234", "203.0.113.9, 198.51.100.5", "")
	if got := g.extractIP(r); got != "192.0.2.7" {
		t.Fatalf("extractIP = %q, want the proxy address 192.0.2.7", got)
	}
}

// SECURITY: a negative hop count must never enable header trust.
func TestNewGuardClampsNegativeTrustedProxyHops(t *testing.T) {
	t.Parallel()

	g := NewGuardWithTrustedProxies(nil, nil, -5)
	if hops := g.TrustedProxyHops(); hops != 0 {
		t.Fatalf("TrustedProxyHops = %d, want 0", hops)
	}
	r := requestWithClientIP("198.51.100.5:44321", "203.0.113.9", "")
	if got := g.extractIP(r); got != "198.51.100.5" {
		t.Fatalf("extractIP = %q, want the transport address", got)
	}
}

func TestResolveForwardedClientIPRejectsMalformedEntries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		xff  string
		hops int
	}{
		{"selected entry is not an IP", "not-an-ip", 1},
		{"selected entry is blank", "   ", 1},
		{"empty header", "", 1},
		{"chain shorter than declared hops", "198.51.100.5", 3},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if ip, ok := resolveForwardedClientIP(tc.xff, tc.hops); ok {
				t.Fatalf("resolveForwardedClientIP(%q, %d) = %q, want rejection", tc.xff, tc.hops, ip)
			}
		})
	}
}

// A client that supplies nothing leaves a chain exactly as long as the declared
// proxy depth, and that entry is still trustworthy.
func TestResolveForwardedClientIPAcceptsExactLengthChain(t *testing.T) {
	t.Parallel()

	ip, ok := resolveForwardedClientIP("198.51.100.5", 1)
	if !ok || ip != "198.51.100.5" {
		t.Fatalf("resolveForwardedClientIP = %q, %v; want 198.51.100.5, true", ip, ok)
	}
}

func TestClientIPFromRemoteAddrHandlesMissingPort(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"198.51.100.5:44321": "198.51.100.5",
		"198.51.100.5":       "198.51.100.5",
		"[2001:db8::1]:8080": "2001:db8::1",
		"":                   "",
	}
	for in, want := range cases {
		if got := clientIPFromRemoteAddr(in); got != want {
			t.Fatalf("clientIPFromRemoteAddr(%q) = %q, want %q", in, got, want)
		}
	}
}

// The rate-limit key must be derived from the resolved identity, so two requests
// from the same transport with different spoofed headers collapse to one bucket.
func TestAuthRateLimitKeyIsStableUnderSpoofing(t *testing.T) {
	t.Parallel()

	g := NewGuard(nil, nil)
	transport := "198.51.100.5:44321"

	first := authRateLimitKey("register", g.extractIP(requestWithClientIP(transport, "203.0.113.1", "")))
	second := authRateLimitKey("register", g.extractIP(requestWithClientIP(transport, "203.0.113.2", "")))
	if first != second {
		t.Fatalf("spoofing produced two buckets: %q and %q", first, second)
	}
	if !strings.HasPrefix(first, "security:rl:register:") {
		t.Fatalf("unexpected key format: %q", first)
	}
}
