// Package middleware provides admin session authentication and CSRF protection.
package middleware

import (
	"context"
	"net/http"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// SessionCookieName is the cookie holding the admin session token.
const SessionCookieName = "classyfm_admin_session"

type ctxKey int

const userCtxKey ctxKey = iota

// AuthUser is the authenticated admin user attached to the request context.
type AuthUser struct {
	ID    uint64
	Email string
	Name  string
	Role  string
}

// Auth loads the session referenced by the session cookie (if any) and attaches the
// user to the request context. It never blocks the request; pair with RequireAuth on
// routes that must be authenticated.
func Auth(q *sqlc.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" || q == nil {
				next.ServeHTTP(w, r)
				return
			}
			sess, err := q.GetSession(r.Context(), cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			user := &AuthUser{ID: sess.UserID, Email: sess.Email, Name: sess.Name, Role: string(sess.Role)}
			ctx := context.WithValue(r.Context(), userCtxKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CurrentUser returns the authenticated user from the request context, or nil.
func CurrentUser(r *http.Request) *AuthUser {
	u, _ := r.Context().Value(userCtxKey).(*AuthUser)
	return u
}

// RequireAuth redirects unauthenticated requests to the admin login page.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CurrentUser(r) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole 403s any request whose authenticated user's role doesn't match
// role. Must run after RequireAuth (which guarantees CurrentUser is non-nil).
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := CurrentUser(r)
			if u == nil || u.Role != role {
				http.Error(w, "Anda tidak memiliki akses ke halaman ini.", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
