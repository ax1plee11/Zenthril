package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"zenthril-backend/config"
)

func TestMetricsAuthRequiresBearerToken(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Environment:  "production",
		MetricsToken: "metrics-token-minimum-32-chars!!!!",
	}
	handler := metricsAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics?token=metrics-token-minimum-32-chars!!!!", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("query token status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer metrics-token-minimum-32-chars!!!!")
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("bearer token status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestOperationalTokenAuthProtectsHealthInProduction(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Environment:  "production",
		MetricsToken: "metrics-token-minimum-32-chars!!!!",
	}
	handler := operationalTokenAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("health without token status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer metrics-token-minimum-32-chars!!!!")
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("health with token status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestCORSMiddlewareRejectsUnknownOrigins(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		CORSAllowedOrigins: []string{"https://app.example.com"},
	}
	handler := corsMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unknown origin status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	req = httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("allowed preflight status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("allow origin = %q, want allowed origin", got)
	}
}

func TestFederationAuthFailsClosed(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Environment: "production"}
	handler := federationAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/federation/v1/announce", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("disabled federation status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
}

func TestFederationAuthRequiresDedicatedToken(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Environment:       "production",
		FederationEnabled: true,
		FederationToken:   "federation-token-minimum-32-chars",
	}
	handler := federationAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/federation/v1/peers", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing federation token status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/federation/v1/peers", nil)
	req.Header.Set("Authorization", "Bearer federation-token-minimum-32-chars")
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("valid federation token status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
