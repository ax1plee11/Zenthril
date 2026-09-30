package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"zenthril-backend/config"
)

func corsHandlerFor(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Config{CORSAllowedOrigins: []string{"https://app.example"}}
	return CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

// SECURITY regression: browsers send Access-Control-Request-Headers as a single
// comma-separated header line (e.g. "authorization, content-type"). Using
// Header.Values without splitting on commas rejected every authenticated
// browser request that needed more than one header.
func TestCORSPreflightAllowsMultipleRequestedHeadersInOneLine(t *testing.T) {
	h := corsHandlerFor(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/channels/abc/messages", nil)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight requesting two headers got %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestCORSPreflightAllowsSingleRequestedHeader(t *testing.T) {
	h := corsHandlerFor(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "authorization")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight requesting one header got %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestCORSPreflightRejectsDisallowedHeader(t *testing.T) {
	h := corsHandlerFor(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "authorization, x-injected")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("preflight requesting a disallowed header got %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCORSPreflightRejectsDisallowedMethod(t *testing.T) {
	h := corsHandlerFor(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodTrace)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("preflight requesting a disallowed method got %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	h := corsHandlerFor(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "https://evil.example")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("unknown origin got %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCORSReflectsExactConfiguredOrigin(t *testing.T) {
	h := corsHandlerFor(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "https://app.example")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("allow-origin = %q, want exact configured origin", got)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credentials must be allowed for the configured origin")
	}
}
