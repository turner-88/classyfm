package middleware

import "net/http"

// CORS adds read-oriented CORS headers to every response and short-circuits the
// preflight OPTIONS request with 204. origin is set verbatim as
// Access-Control-Allow-Origin ("*" when blank).
//
// It is meant for the public /api/v1 surface, which is read-only and cookie-free:
// browsers get GET data they could already fetch server-side, and the write
// endpoints authenticate with a Bearer token (not the ambient session cookie), so
// a wildcard origin exposes nothing extra. If you ever need credentialed requests,
// pass a single concrete origin instead of "*".
func CORS(origin string) func(http.Handler) http.Handler {
	if origin == "" {
		origin = "*"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			if origin != "*" {
				// The response now varies by request Origin, so caches must key on it.
				h.Add("Vary", "Origin")
			}
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			h.Set("Access-Control-Max-Age", "86400")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
