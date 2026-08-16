// Package middleware also provides baseline security headers for every response.
package middleware

import "net/http"

// SecurityHeaders sets baseline defensive headers on every response. CSP is strict
// (no inline scripts, and the only inline style allow-listed is Turbo Drive's
// progress-bar <style>, pinned by hash in contentSecurityPolicy) but allows https:
// images since news items/covers link to external hosts (YouTube, RSS sources,
// the now-playing API), allows Google Fonts (Baloo 2) via its two fixed hosts,
// and allows Cloudflare's Web Analytics beacon script (static.cloudflareinsights.com)
// since it's served from a fixed, known origin. Cloudflare's separate Bot
// Management/JS-challenge snippet is injected inline with a per-request token
// and can't be allowlisted this way (see project memory on CSP/Cloudflare).
// Google Analytics 4 needs googletagmanager.com in script-src for gtag.js plus its
// collection endpoints in connect-src, since GA4 sends hits via fetch/sendBeacon
// rather than the legacy pixel img-src already covers. Those are fixed, known origins
// and are allowlisted unconditionally (like Cloudflare's above) - nothing contacts them
// unless GA_MEASUREMENT_ID is set and the layout actually emits the tag. Note this
// allows gtag.js only, not a GTM container, which could inject arbitrary third-party
// scripts and defeat the point of the policy.
// frame-src covers the two embeds on the site: the About page banner's YouTube
// player (admin-chosen video mode) and the podcast detail page's Spotify player.
// Without it the iframes fall back to default-src 'self' and the browser blocks
// them outright, so the allowlist stays limited to YouTube's two hosts and Spotify.
// hsts is only set when the app is served over TLS in production (locally we
// run plain HTTP behind no proxy).
func SecurityHeaders(hsts, chat bool) func(http.Handler) http.Handler {
	policy := contentSecurityPolicy(false, chat)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("Content-Security-Policy", policy)
			if hsts {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AdminContentSecurityPolicy overrides the global CSP for the (auth-gated) admin
// panel with one that additionally allows inline styles. The legal-page editor
// (Toast UI Editor, web/static/js/vendor) applies element style attributes at
// runtime that 'style-src self' blocks — verified in headless Chrome, only
// style-src is affected (no inline scripts, no eval). Relaxing style-src just for
// /admin keeps the public site's strict policy intact. Mount it on the admin route
// group after the global SecurityHeaders, whose header it replaces.
func AdminContentSecurityPolicy(chat bool) func(http.Handler) http.Handler {
	policy := contentSecurityPolicy(true, chat)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", policy)
			next.ServeHTTP(w, r)
		})
	}
}

// contentSecurityPolicy builds the CSP string. When allowInlineStyle is true,
// style-src uses 'unsafe-inline' instead of the Turbo progress-bar hash (admin panel
// only — see AdminContentSecurityPolicy); the two are mutually exclusive because a hash
// in style-src makes the browser ignore 'unsafe-inline'. When allowFirebase is true (FEATURE_CHAT on), the
// Firebase-specific script/connect/frame hosts the Connect chat needs are added;
// with the flag off nothing loads or contacts Firebase, so they are omitted.
func contentSecurityPolicy(allowInlineStyle, allowFirebase bool) string {
	// style-src: the hash and 'unsafe-inline' are MUTUALLY EXCLUSIVE and must never
	// share the list — per the CSP spec, a hash (or nonce) in style-src makes the
	// browser ignore 'unsafe-inline'. Combining them is exactly what silently broke
	// the admin dashboard's overlay labels and the legal-page editor.
	styleSrc := "style-src 'self' https://fonts.googleapis.com "
	if allowInlineStyle {
		// Admin panel only: 'unsafe-inline' covers both inline style="" attributes and
		// Turbo Drive's injected progress-bar <style>, so no hash is needed here.
		styleSrc += "'unsafe-inline'"
	} else {
		// Public site stays strict: allow-list only Turbo Drive's fixed progress-bar
		// <style> element (injected into <head> on first navigation by
		// web/static/js/turbo.js) by its exact hash; every other inline style stays
		// blocked. The hash is stable until turbo.js is upgraded — recompute it (the
		// browser console reports the needed value) if the progress-bar CSS changes.
		styleSrc += "'sha256-WAyOw4V+FqDc35lQPyRADLBWbuNK8ahvYEaQIYF1+Ps='"
	}

	// apis.google.com serves gapi (apis.google.com/js/api.js), which the Firebase
	// Auth popup/redirect flow loads to relay the sign-in result — chat only.
	scriptSrc := "script-src 'self' https://static.cloudflareinsights.com https://www.googletagmanager.com"
	// The Connect chat uses the (locally vendored) Firebase SDK against the shared
	// Realtime Database: RTDB streams over WebSocket (wss://*.firebaseio.com), and
	// Firebase Auth (Google) + installations call the identitytoolkit/securetoken/
	// firebaseinstallations Google APIs. script-src stays 'self' (the SDK is served
	// from /static, not a CDN).
	connectSrc := "connect-src 'self' https://*.google-analytics.com https://*.analytics.google.com https://www.googletagmanager.com"
	// frame-src adds the Firebase auth domain + accounts.google.com for the Google
	// sign-in popup/iframe helper, alongside the YouTube/Spotify embeds.
	frameSrc := "frame-src https://www.youtube.com https://www.youtube-nocookie.com https://open.spotify.com"
	if allowFirebase {
		scriptSrc += " https://apis.google.com"
		connectSrc += " https://*.googleapis.com https://*.firebaseio.com wss://*.firebaseio.com " +
			"https://securetoken.googleapis.com https://identitytoolkit.googleapis.com https://firebaseinstallations.googleapis.com"
		frameSrc += " https://classyfm-dd873.firebaseapp.com https://accounts.google.com https://apis.google.com"
	}

	return "default-src 'self'; " +
		"img-src 'self' https: data:; " +
		scriptSrc + "; " +
		styleSrc + "; " +
		"font-src 'self' https://fonts.gstatic.com; " +
		connectSrc + "; " +
		"media-src 'self' https:; " +
		frameSrc + "; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"
}
