// Package middleware also provides baseline security headers for every response.
package middleware

import "net/http"

// SecurityHeaders sets baseline defensive headers on every response. CSP is strict
// (no inline scripts/styles are used anywhere in the templates) but allows https:
// images since news items/covers link to external hosts (YouTube, RSS sources,
// the now-playing API), allows Google Fonts (Baloo 2) via its two fixed hosts,
// and allows Cloudflare's Web Analytics beacon script (static.cloudflareinsights.com)
// since it's served from a fixed, known origin. Cloudflare's separate Bot
// Management/JS-challenge snippet is injected inline with a per-request token
// and can't be allowlisted this way (see project memory on CSP/Cloudflare).
// frame-src exists solely for the About page's banner, whose admin-chosen video
// mode embeds a YouTube player; without it the iframe falls back to default-src
// 'self' and the browser blocks it outright. Nothing else on the site frames
// third-party content, so that allowlist stays limited to YouTube's two hosts.
// hsts is only set when the app is served over TLS in production (locally we
// run plain HTTP behind no proxy).
func SecurityHeaders(hsts bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("Content-Security-Policy",
				"default-src 'self'; "+
					"img-src 'self' https: data:; "+
					"script-src 'self' https://static.cloudflareinsights.com; "+
					"style-src 'self' https://fonts.googleapis.com; "+
					"font-src 'self' https://fonts.gstatic.com; "+
					"connect-src 'self'; "+
					"media-src 'self' https:; "+
					"frame-src https://www.youtube.com https://www.youtube-nocookie.com; "+
					"frame-ancestors 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'")
			if hsts {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
