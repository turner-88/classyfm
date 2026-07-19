package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

// CSRFCookieName is the cookie holding the per-browser CSRF token (double-submit).
const CSRFCookieName = "csrf_token"

const csrfFieldName = "csrf_token"

type csrfCtxKey int

const csrfTokenCtxKey csrfCtxKey = iota

// CSRF ensures every request carries a CSRF token cookie, and rejects state-changing
// requests (POST/PUT/PATCH/DELETE) whose "csrf_token" form field doesn't match it.
func CSRF(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if cookie, err := r.Cookie(CSRFCookieName); err == nil {
				token = cookie.Value
			}
			if token == "" {
				token = generateCSRFToken()
				http.SetCookie(w, &http.Cookie{
					Name:     CSRFCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: true,
					Secure:   secure,
					SameSite: http.SameSiteLaxMode,
				})
			}

			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				submitted := r.FormValue(csrfFieldName)
				if submitted == "" || subtle.ConstantTimeCompare([]byte(submitted), []byte(token)) != 1 {
					http.Error(w, "invalid or missing CSRF token", http.StatusForbidden)
					return
				}
			}

			ctx := context.WithValue(r.Context(), csrfTokenCtxKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CSRFToken returns the current request's CSRF token, for embedding in forms.
func CSRFToken(r *http.Request) string {
	tok, _ := r.Context().Value(csrfTokenCtxKey).(string)
	return tok
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand failure is unrecoverable
	}
	return hex.EncodeToString(b)
}
