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
