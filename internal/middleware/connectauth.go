package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// ConnectSessionCookieName holds the Connect chatroom's session token. It is entirely
// separate from the admin SessionCookieName so the public chat identity and the admin
// session can coexist in one browser without interfering.
const ConnectSessionCookieName = "classyfm_connect_session"

// connectMarker is the first payload field of a Connect session token, mirroring the
// impersonateMarker scheme in virtual.go so the shared signing helpers can carry it.
const connectMarker = "connect"

type connectCtxKey int

const chatUserCtxKey connectCtxKey = iota

// ChatUser is the Google-authenticated Connect chatroom user attached to the request
// context. IsAdmin drives the special badge; IsBanned gates posting. It is re-read from
// the database on every identity-bearing request (see ConnectAuth), so a ban takes
// effect immediately rather than waiting for the token to expire.
type ChatUser struct {
	ID        uint64
	Name      string
	AvatarURL string
	IsAdmin   bool
	IsBanned  bool
}

// NewConnectSessionToken mints a self-contained, HMAC-signed cookie value carrying only
// the chat user's id, mirroring NewImpersonationToken: the rest of the identity is
// re-read from the database each request.
func NewConnectSessionToken(secret string, chatUserID uint64, ttl time.Duration) string {
	expiry := time.Now().Add(ttl).Unix()
	payload := connectMarker + "|" + strconv.FormatUint(chatUserID, 10) + "|" + strconv.FormatInt(expiry, 10)
	return signVirtualPayload(payload, secret)
}

// verifyConnectSessionToken checks the signature and expiry of a value produced by
// NewConnectSessionToken and returns the chat user id. It reports failure (not an error)
// for anything unrecognizable.
func verifyConnectSessionToken(token, secret string) (uint64, bool) {
	payload, ok := parseSignedPayload(token, secret)
	if !ok {
		return 0, false
	}
	fields := strings.Split(payload, "|")
	if len(fields) != 3 || fields[0] != connectMarker {
		return 0, false
	}
	id, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	if !unexpired(fields[2]) {
		return 0, false
	}
	return id, true
}

// ConnectAuth loads the Connect chat user referenced by the connect session cookie (if
// any) and attaches it to the request context. Like Auth it never blocks the request and
// only touches the database when a connect cookie is present, so it is cheap to apply
// across every public page (needed for the site-wide floating chat widget). Pair with
// RequireChatAuth on routes that must be authenticated.
func ConnectAuth(q *sqlc.Queries, sessionSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(ConnectSessionCookieName)
			if err != nil || cookie.Value == "" || q == nil {
				next.ServeHTTP(w, r)
				return
			}
			id, ok := verifyConnectSessionToken(cookie.Value, sessionSecret)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			u, err := q.GetChatUserByID(r.Context(), id)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			cu := &ChatUser{ID: u.ID, Name: u.Name, AvatarURL: u.AvatarUrl.String, IsAdmin: u.IsAdmin, IsBanned: u.IsBanned}
			ctx := context.WithValue(r.Context(), chatUserCtxKey, cu)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CurrentChatUser returns the authenticated Connect chat user from the request context,
// or nil.
func CurrentChatUser(r *http.Request) *ChatUser {
	u, _ := r.Context().Value(chatUserCtxKey).(*ChatUser)
	return u
}

// RequireChatAuth 401s any request not made by a signed-in Connect chat user. Must run
// after ConnectAuth.
func RequireChatAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CurrentChatUser(r) == nil {
			http.Error(w, "Silakan masuk dengan Google untuk mengirim pesan.", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
