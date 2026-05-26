package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookieBackedRefreshRequiresOriginWhenSecureCookiesEnabled(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, true)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "refresh"})

	if handler.allowCookieBackedRequest(req) {
		t.Fatal("secure cookie-backed request without Origin was accepted")
	}

	req.Header.Set("Origin", "https://app.example.com")
	if !handler.allowCookieBackedRequest(req) {
		t.Fatal("secure cookie-backed request with Origin was rejected")
	}
}

func TestSecureRefreshAllowsNonCookieAPITokenFlowWithoutOrigin(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, true)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)

	if !handler.allowCookieBackedRequest(req) {
		t.Fatal("non-cookie API token flow without Origin was rejected")
	}
}

func TestCookieBackedRefreshAllowsLocalDevelopmentWithoutOrigin(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, false)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)

	if !handler.allowCookieBackedRequest(req) {
		t.Fatal("development cookie-backed request without Origin was rejected")
	}
}
