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
	ID           uint64
	Email        string
	Name         string
	Role         string
	Virtual      bool // true for the break-glass root login: not backed by a users row
	Impersonated bool // true when the break-glass root is acting as this user (see virtual.go)
}

// Auth loads the session referenced by the session cookie (if any) and attaches the
// user to the request context. It never blocks the request; pair with RequireAuth on
// routes that must be authenticated. sessionSecret is used only to verify the
// self-contained virtual-session cookie minted by the break-glass root login (see
// VirtualSessionCookie); it never touches the DB-backed session flow below.
func Auth(q *sqlc.Queries, sessionSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			if user := verifyVirtualSessionToken(cookie.Value, sessionSecret); user != nil {
				ctx := context.WithValue(r.Context(), userCtxKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if q == nil {
				next.ServeHTTP(w, r)
				return
			}
			if userID, ok := verifyImpersonationToken(cookie.Value, sessionSecret); ok {
				// Re-read the user every request rather than trusting the token's
				// contents, so a rename, role change or deactivation takes effect
				// immediately - the same guarantees GetSession bakes into its query
				// for normal sessions.
				u, err := q.GetUserByID(r.Context(), userID)
				if err != nil || !u.IsActive {
					next.ServeHTTP(w, r)
					return
				}
				user := &AuthUser{ID: u.ID, Email: u.Email, Name: u.Name, Role: string(u.Role), Impersonated: true}
				ctx := context.WithValue(r.Context(), userCtxKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
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

// RequireVirtualRoot 403s any request not made by the break-glass root login.
// Must run after RequireAuth. Note an impersonated user has Virtual false, so
// impersonation can never be chained from within an impersonated session.
func RequireVirtualRoot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := CurrentUser(r); u == nil || !u.Virtual {
			http.Error(w, "Anda tidak memiliki akses ke halaman ini.", http.StatusForbidden)
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
