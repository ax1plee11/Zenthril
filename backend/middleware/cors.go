package middleware

import (
	"net/http"
	"strings"

	"zenthril-backend/config"
)

// CORS adds strict CORS headers based on the configured allowed origins.
// SECURITY: exact-origin matching prevents wildcard bypasses and reflects only
// configured origins. Preflight is cached for 10 minutes.
func CORS(cfg config.Config) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(cfg.CORSAllowedOrigins))
	for _, origin := range cfg.CORSAllowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))

			w.Header().Add("Vary", "Origin")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")

			if origin != "" {
				if _, ok := allowed[origin]; !ok {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				if origin == "" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if !validPreflightRequest(r) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func validPreflightRequest(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Header.Get("Access-Control-Request-Method")))
	if method == "" {
		return false
	}
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		// ok
	default:
		return false
	}

	reqHeaders := r.Header.Values("Access-Control-Request-Headers")
	allowedHeaders := map[string]struct{}{
		"authorization": {},
		"content-type":  {},
	}
	for _, h := range reqHeaders {
		if _, ok := allowedHeaders[strings.ToLower(strings.TrimSpace(h))]; !ok {
			return false
		}
	}
	return true
}
