package public

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	appmw "github.com/classyfm/classyfm/internal/middleware"
)

// Connect chatroom tunables.
const (
	// chatRecentLimit caps how many recent messages the page and the "no since"
	// poll return, oldest-to-newest after reversing.
	chatRecentLimit = 50
	// chatPollLimit caps a single polling delta so a long-idle tab reconnecting
	// can't pull an unbounded backlog in one request.
	chatPollLimit = 200
	// chatMaxBody is the max stored message length (matches the VARCHAR(1000)).
	chatMaxBody = 1000
	// connectSessionTTL is how long a Connect login lasts.
	connectSessionTTL = 30 * 24 * time.Hour
	// oauthStateTTL bounds the OAuth round-trip.
	oauthStateTTL = 10 * time.Minute
)

const connectStateCookieName = "classyfm_connect_oauth_state"

// chatSanitizer strips all HTML from posted message bodies. Template auto-escaping is
// the second layer; this keeps the stored value clean too.
var chatSanitizer = bluemonday.StrictPolicy()

// chatEnabledKey is the settings row that gates public chat posting site-wide (toggled from
// the admin panel). Absent (or anything other than "0") means chat is on; posting is
// disabled only when it is explicitly "0".
const chatEnabledKey = "chat_enabled"

// chatEnabled reports whether public chat posting is currently allowed. A missing row or a
// read error is treated as enabled (fail-open: a settings glitch must not silently mute the
// whole chat). Reads stay open regardless — this gates writes only.
func (h *Handler) chatEnabled(r *http.Request) bool {
	if h.q == nil {
		return true
	}
	s, err := h.q.GetSetting(r.Context(), chatEnabledKey)
	if err != nil {
		return true
	}
	return s.V != "0"
}

// chatMessageVM is one rendered chat message, shared by the initial page render and the
// polling JSON so a single formatter drives both surfaces.
type chatMessageVM struct {
	ID      uint64 `json:"id"`
	UserID  uint64 `json:"user_id"` // author's chat_users.id, target of the ban action
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
	IsAdmin bool   `json:"is_admin"`
	Body    string `json:"body"`
	Time    string `json:"time"`
}

// formatChatTime renders a message timestamp as clock time in the station's timezone -
// stable (no client-side ticking) and unambiguous for a live chat.
func formatChatTime(t time.Time) string { return t.In(stationLoc).Format("15:04") }

// Connect renders the dedicated /connect chatroom page. The feed is hydrated client-side
// by connect.js from ConnectMessagesJSON - the same path the site-wide floating widget
// uses - so there is a single rendering path for messages (no template/JS divergence).
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	h.r.Page(w, http.StatusOK, "public/connect", struct {
		Base baseData
	}{h.base(r, "Connect", "connect", "Join the "+h.station+" public chatroom.")})
}

// ConnectMessagesJSON is the open (no-auth) polling endpoint. With ?since=<id> it returns
// only newer messages; without it, the most recent page. Anyone can read the chat.
func (h *Handler) ConnectMessagesJSON(w http.ResponseWriter, r *http.Request) {
	out := []chatMessageVM{}
	if h.q != nil {
		if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
			since, err := strconv.ParseUint(sinceStr, 10, 64)
			if err == nil {
				// The in-process cache serves the steady-state case (an active poller whose
				// cursor is within the recent window); it misses only for a long-idle
				// reconnect, which falls back to the DB.
				if cached, ok := h.chat.Since(r.Context(), since); ok {
					out = cached
				} else {
					rows, _ := h.q.ListChatMessagesSince(r.Context(), sqlc.ListChatMessagesSinceParams{ID: since, Limit: chatPollLimit})
					for _, m := range rows {
						out = append(out, chatMessageVM{
							ID: m.ID, UserID: m.ChatUserID, Name: m.AuthorName, Avatar: m.AuthorAvatar.String,
							IsAdmin: m.AuthorIsAdmin, Body: m.Body, Time: formatChatTime(m.CreatedAt),
						})
					}
				}
			}
		} else if cached, ok := h.chat.Recent(r.Context(), chatRecentLimit); ok {
			out = cached
		} else {
			rows, _ := h.q.ListRecentChatMessages(r.Context(), chatRecentLimit)
			for i := len(rows) - 1; i >= 0; i-- {
				m := rows[i]
				out = append(out, chatMessageVM{
					ID: m.ID, UserID: m.ChatUserID, Name: m.AuthorName, Avatar: m.AuthorAvatar.String,
					IsAdmin: m.AuthorIsAdmin, Body: m.Body, Time: formatChatTime(m.CreatedAt),
				})
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// hidden carries recently-moderated ids (independent of the ?since cursor, since a hidden
	// message's id is <= the client's cursor) so connect.js can remove them from a live feed.
	_ = json.NewEncoder(w).Encode(map[string]any{"messages": out, "hidden": h.chat.Hidden()})
}

// ConnectPost stores a message from the signed-in chat user. Requires chat auth (wired
// via the route), rejects banned users, and sanitizes + length-caps the body. Answers
// JSON to a fetch (Accept: application/json) or redirects back for the no-JS fallback.
func (h *Handler) ConnectPost(w http.ResponseWriter, r *http.Request) {
	u := appmw.CurrentChatUser(r)
	if u == nil {
		http.Error(w, "Silakan masuk dengan Google untuk mengirim pesan.", http.StatusUnauthorized)
		return
	}
	if u.IsBanned {
		http.Error(w, "Akun Anda diblokir dari chat.", http.StatusForbidden)
		return
	}
	if !h.chatEnabled(r) {
		http.Error(w, "Chat sedang dinonaktifkan.", http.StatusForbidden)
		return
	}
	body := chatSanitizer.Sanitize(strings.TrimSpace(r.FormValue("body")))
	if body == "" {
		h.connectRespond(w, r, http.StatusBadRequest, nil)
		return
	}
	if len(body) > chatMaxBody {
		body = body[:chatMaxBody]
	}
	if h.q == nil {
		http.Error(w, "chat unavailable", http.StatusServiceUnavailable)
		return
	}
	res, err := h.q.CreateChatMessage(r.Context(), sqlc.CreateChatMessageParams{ChatUserID: u.ID, Body: body})
	if err != nil {
		http.Error(w, "could not send message", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	msg := &chatMessageVM{
		ID: uint64(id), UserID: u.ID, Name: u.Name, Avatar: u.AvatarURL, IsAdmin: u.IsAdmin,
		Body: body, Time: formatChatTime(time.Now()),
	}
	h.chat.Append(*msg)
	h.connectRespond(w, r, http.StatusOK, msg)
}

// connectRespond answers a chat POST: JSON for a fetch, otherwise a redirect back to the
// page the message was posted from (so the widget works from any page).
func (h *Handler) connectRespond(w http.ResponseWriter, r *http.Request, status int, msg *chatMessageVM) {
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": msg})
		return
	}
	dest := r.Referer()
	if dest == "" {
		dest = "/connect"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// ConnectLogin starts the Google OAuth flow: mint a random state, stash it in a
// short-lived cookie, and redirect to Google's consent screen.
func (h *Handler) ConnectLogin(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		http.Error(w, "Chat login is not configured.", http.StatusNotFound)
		return
	}
	state, err := randomHex(16)
	if err != nil {
		http.Error(w, "could not start login", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     connectStateCookieName,
		Value:    state,
		Path:     "/connect",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(oauthStateTTL),
	})
	http.Redirect(w, r, h.oauth.AuthCodeURL(state), http.StatusSeeOther)
}

// googleUserInfo is the subset of Google's userinfo response we consume. "id" is the
// stable Google subject id used as chat_users.google_sub.
type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// ConnectCallback completes the OAuth flow: verify state, exchange the code, read the
// Google profile, upsert the chat user (recomputing the admin badge), and set the
// session cookie.
func (h *Handler) ConnectCallback(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil || h.q == nil {
		http.Error(w, "Chat login is not configured.", http.StatusNotFound)
		return
	}
	stateCookie, err := r.Cookie(connectStateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid login state", http.StatusBadRequest)
		return
	}
	// Clear the one-shot state cookie.
	http.SetCookie(w, &http.Cookie{Name: connectStateCookieName, Path: "/connect", MaxAge: -1})

	tok, err := h.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "login failed", http.StatusBadGateway)
		return
	}
	client := h.oauth.Client(r.Context(), tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "could not read Google profile", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil || info.ID == "" {
		http.Error(w, "could not read Google profile", http.StatusBadGateway)
		return
	}

	name := strings.TrimSpace(info.Name)
	if name == "" {
		name = info.Email
	}
	isAdmin := h.emailIsAdmin(r, info.Email)

	if _, err := h.q.UpsertChatUser(r.Context(), sqlc.UpsertChatUserParams{
		GoogleSub: info.ID,
		Email:     info.Email,
		Name:      name,
		AvatarUrl: nullString(info.Picture),
		IsAdmin:   isAdmin,
	}); err != nil {
		http.Error(w, "could not sign you in", http.StatusInternalServerError)
		return
	}
	cu, err := h.q.GetChatUserByGoogleSub(r.Context(), info.ID)
	if err != nil {
		http.Error(w, "could not sign you in", http.StatusInternalServerError)
		return
	}

	token := appmw.NewConnectSessionToken(h.sessionSecret, cu.ID, connectSessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     appmw.ConnectSessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(connectSessionTTL),
	})
	http.Redirect(w, r, "/connect", http.StatusSeeOther)
}

// emailIsAdmin reports whether a Google email belongs to an active admin/superadmin in
// the CMS users table - the source of the Connect "special badge".
func (h *Handler) emailIsAdmin(r *http.Request, email string) bool {
	if h.q == nil || email == "" {
		return false
	}
	u, err := h.q.GetUserByEmail(r.Context(), strings.ToLower(email))
	if err != nil || !u.IsActive {
		return false
	}
	return u.Role == sqlc.UsersRoleAdmin || u.Role == sqlc.UsersRoleSuperadmin
}

// ConnectLogout clears the Connect session cookie.
func (h *Handler) ConnectLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     appmw.ConnectSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	// A fetch from connect.js (see bindLogout) gets the freshly-rendered signed-out
	// composer back to swap in place, so sign-out reflects without a page reload. The
	// cookie is cleared above but ctx still holds the user for this request, so force
	// ChatUser nil to render the signed-out branch. Plain form posts fall through to the
	// redirect below (the no-JS / degraded path).
	if r.Header.Get("X-Requested-With") == "fetch" {
		b := h.base(r, "Connect", "connect", "")
		b.ChatUser = nil
		h.r.Partial(w, http.StatusOK, "public/connect", "connect-composer", b)
		return
	}
	dest := r.Referer()
	if dest == "" {
		dest = "/connect"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// nullString wraps a possibly-empty string as a sql.NullString (blank -> NULL).
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
