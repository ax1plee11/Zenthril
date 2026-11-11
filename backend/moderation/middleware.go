package moderation

import (
	"context"
	"net/http"
)

func (t *ConnectionMonitor) ConnectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		go func() {
			userID, ok := r.Context().Value("userID").(string)
			if !ok || userID == "" {
				return
			}

			headers := make(map[string]string)
			for key, values := range r.Header {
				if len(values) > 0 {
					headers[key] = values[0]
				}
			}

			ips := ExtractRealIP(headers, r.RemoteAddr)
			userAgent := r.Header.Get("User-Agent")

			ctx := context.Background()
			_ = t.RecordConnection(ctx, userID, ips, userAgent)
		}()
	})
}
