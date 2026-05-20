package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related HTTP headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy - restrict resources that can be loaded
		// Default-src 'self' - only allow resources from same origin
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none';")

		// Strict-Transport-Security - enforce HTTPS for 1 year
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// X-Content-Type-Options - prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// X-Frame-Options - prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// X-XSS-Protection - enable XSS filter
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy - control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy - restrict browser features
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}
