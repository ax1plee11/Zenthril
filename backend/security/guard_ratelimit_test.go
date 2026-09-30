package security

import "testing"

// SECURITY regression: endpoints that succeed on the happy path were never
// limited, because BruteForceProtect only reacts to 401 responses. These tests
// pin the budget boundary and the per-scope key isolation.
func TestRateLimitAllowsUsesInclusiveBoundary(t *testing.T) {
	t.Parallel()

	const limit = 3
	for count := int64(1); count <= limit; count++ {
		if !rateLimitAllows(count, limit) {
			t.Fatalf("count %d should be allowed within limit %d", count, limit)
		}
	}
	if rateLimitAllows(limit+1, limit) {
		t.Fatalf("count %d must be rejected beyond limit %d", limit+1, limit)
	}
}

func TestRateLimitAllowsRejectsUnlimitedConfiguration(t *testing.T) {
	t.Parallel()

	// A zero or negative limit must never behave as "allow everything".
	if rateLimitAllows(1, 0) {
		t.Fatal("a zero limit must reject all requests")
	}
	if rateLimitAllows(1, -1) {
		t.Fatal("a negative limit must reject all requests")
	}
}

func TestAuthRateLimitKeySeparatesScopesAndClients(t *testing.T) {
	t.Parallel()

	register := authRateLimitKey("register", "203.0.113.5")
	refresh := authRateLimitKey("refresh", "203.0.113.5")
	otherIP := authRateLimitKey("register", "198.51.100.9")

	if register == refresh {
		t.Fatal("different scopes must not share a counter")
	}
	if register == otherIP {
		t.Fatal("different client IPs must not share a counter")
	}
	if register != "security:rl:register:203.0.113.5" {
		t.Fatalf("unexpected key format: %q", register)
	}
}
