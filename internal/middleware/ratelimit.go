package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	count   int
	resetAt time.Time
}

// RateLimit throttles each client IP to max requests per window, answering a block with a
// plaintext (Indonesian) message — the shape the admin/HTML forms expect. It's a simple
// in-memory fixed-window limiter — enough to slow down brute-force attempts against a
// single-instance deployment (e.g. the admin login form) without pulling in an external
// dependency.
func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimit(max, window, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Terlalu banyak percobaan, coba lagi nanti.", http.StatusTooManyRequests)
	})
}

// RateLimitJSON is RateLimit for the JSON API: a block answers with the API's uniform
// {"error": ...} body (English, matching the other API errors) instead of plaintext, so
// /api/v1 clients never see a non-JSON response.
func RateLimitJSON(max int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimit(max, window, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests, try again later"})
	})
}

// rateLimit is the shared fixed-window throttling core; onBlocked writes the response for
// a throttled request, letting callers vary the body format (plaintext vs JSON).
func rateLimit(max int, window time.Duration, onBlocked http.HandlerFunc) func(http.Handler) http.Handler {
	var (
		mu           sync.Mutex
		visitors     = map[string]*visitor{}
		lastCleanup  = time.Now()
		cleanupEvery = 10 * time.Minute
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()

			mu.Lock()
			if now.Sub(lastCleanup) > cleanupEvery {
				for k, v := range visitors {
					if now.After(v.resetAt) {
						delete(visitors, k)
					}
				}
				lastCleanup = now
			}
			v, ok := visitors[ip]
			if !ok || now.After(v.resetAt) {
				v = &visitor{resetAt: now.Add(window)}
				visitors[ip] = v
			}
			v.count++
			blocked := v.count > max
			mu.Unlock()

			if blocked {
				onBlocked(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
