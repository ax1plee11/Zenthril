package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"zenthril-backend/auth"
	"zenthril-backend/config"
)

func TestMetricsAuthRequiresBearerToken(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Environment:  "production",
		MetricsToken: "metrics-token-minimum-32-chars!!!!",
		JWTSecret:    "test-secret-at-least-32-bytes-long!!",
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

func TestMetricsAuthAllowsAdminJWT(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Environment:  "production",
		MetricsToken: "metrics-token-minimum-32-chars!!!!",
		JWTSecret:    "test-secret-at-least-32-bytes-long!!",
		AdminUserIDs: []string{"admin-user"},
	}
	token, err := auth.GenerateToken("admin-user", cfg.JWTSecret)
	if err != nil {
		t.Fatal(err)
	}
	handler := metricsAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("admin token status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestOperationalTokenAuthProtectsHealthInProduction(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Environment:      "production",
		OperationalToken: "operational-token-minimum-32-chars",
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
	req.Header.Set("Authorization", "Bearer operational-token-minimum-32-chars")
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("health with token status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestReadinessHandlerReportsReady(t *testing.T) {
	t.Parallel()

	handler := readinessHandler("development", readinessDependency{
		Name: "postgres",
		Ping: func(context.Context) error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); !strings.Contains(got, `"status":"ready"`) || !strings.Contains(got, `"postgres":"ok"`) {
		t.Fatalf("ready body = %s", got)
	}
}

func TestReadinessHandlerReportsDependencyFailureInDevelopment(t *testing.T) {
	t.Parallel()

	handler := readinessHandler("development", readinessDependency{
		Name: "redis",
		Ping: func(context.Context) error { return errors.New("redis down") },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("not ready status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Body.String(); !strings.Contains(got, `"status":"not_ready"`) || !strings.Contains(got, `"redis":"down"`) {
		t.Fatalf("not ready body = %s", got)
	}
}

func TestReadinessHandlerHidesDependencyDetailsInProduction(t *testing.T) {
	t.Parallel()

	handler := readinessHandler("production", readinessDependency{
		Name: "postgres",
		Ping: func(context.Context) error { return errors.New("dsn refused") },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("not ready status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Body.String(); strings.Contains(got, "postgres") || strings.Contains(got, "dsn refused") {
		t.Fatalf("production readiness leaked details: %s", got)
	}
}

func TestBlockDebugEndpointsInProduction(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Environment: "production"}
	handler := blockDebugEndpoints(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("debug status = %d, want %d", rec.Code, http.StatusNotFound)
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
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("allowed preflight status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("allow origin = %q, want allowed origin", got)
	}
}

func TestCORSMiddlewareRejectsInvalidPreflight(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		CORSAllowedOrigins: []string{"https://app.example.com"},
	}
	handler := corsMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing preflight method status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	req = httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "X-Unsafe-Header")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unsafe preflight header status = %d, want %d", rec.Code, http.StatusForbidden)
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
