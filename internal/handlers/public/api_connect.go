package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	appmw "github.com/classyfm/classyfm/internal/middleware"
)

// googleTokenInfoClient verifies Google ID tokens over HTTP (see verifyGoogleIDToken).
// A short timeout keeps a slow Google response from stalling a sign-in request.
var googleTokenInfoClient = &http.Client{Timeout: 10 * time.Second}

// googleTokenInfo is the subset of Google's tokeninfo response we consume. All fields
// come back as strings from that endpoint (including the numeric exp).
type googleTokenInfo struct {
	Iss     string `json:"iss"`
	Sub     string `json:"sub"`
	Aud     string `json:"aud"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// verifyGoogleIDToken validates a Google-issued ID token via Google's public tokeninfo
// endpoint — no third-party dependency, at the cost of one network round-trip (acceptable
// here: it runs once per sign-in, not per message). Google verifies the signature and
// expiry; the audience and issuer are NOT checked by tokeninfo, so we enforce them
// ourselves — a token minted for another app must never authenticate here.
func verifyGoogleIDToken(ctx context.Context, idToken, audience string) (googleTokenInfo, error) {
	var info googleTokenInfo
	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return info, err
	}
	resp, err := googleTokenInfoClient.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// tokeninfo answers 400 for a bad, malformed, or expired token.
		return info, errInvalidGoogleToken
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return info, err
	}
	if info.Sub == "" || info.Aud != audience {
		return info, errInvalidGoogleToken
	}
	if info.Iss != "accounts.google.com" && info.Iss != "https://accounts.google.com" {
		return info, errInvalidGoogleToken
	}
	return info, nil
}

// errInvalidGoogleToken is returned by verifyGoogleIDToken for any token that fails
// verification (bad signature/expiry per Google, or a wrong audience/issuer per us).
var errInvalidGoogleToken = errors.New("invalid Google ID token")

// This file implements the Connect chatroom's JSON API at /api/v1/connect. Reads are open
// (reuse ConnectMessagesJSON directly at the route). Writes authenticate with a Bearer
// Connect session token (ConnectAuthBearer middleware) rather than the web's session
// cookie. A native app obtains that token from APIConnectSession by exchanging a Google
// ID token it acquired via the platform's own sign-in — so Google OAuth never runs inside
// a WebView.

// connectUserDTO is the chat identity returned by the session-exchange and /me endpoints.
type connectUserDTO struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar,omitempty"`
	IsAdmin bool   `json:"is_admin"`
}

// APIConnectSession exchanges a Google ID token (acquired natively by the app) for a
// Connect session token. It verifies the ID token against the configured Google client ID
// — the same client the website's web OAuth uses, which the apps pass as their
// serverClientId — then upserts the chat user (recomputing the admin badge) exactly like
// the web ConnectCallback, and returns the signed session token for the app to send as a
// Bearer header.
func (h *Handler) APIConnectSession(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil || h.q == nil {
		writeJSONError(w, http.StatusNotFound, "chat login is not configured")
		return
	}
	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil || strings.TrimSpace(req.IDToken) == "" {
		writeJSONError(w, http.StatusBadRequest, "id_token is required")
		return
	}

	info, err := verifyGoogleIDToken(r.Context(), req.IDToken, h.oauth.ClientID)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid Google token")
		return
	}
	name := strings.TrimSpace(info.Name)
	if name == "" {
		name = info.Email
	}

	if _, err := h.q.UpsertChatUser(r.Context(), sqlc.UpsertChatUserParams{
		GoogleSub: info.Sub,
		Email:     info.Email,
		Name:      name,
		AvatarUrl: nullString(info.Picture),
		IsAdmin:   h.emailIsAdmin(r, info.Email),
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not sign you in")
		return
	}
	cu, err := h.q.GetChatUserByGoogleSub(r.Context(), info.Sub)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not sign you in")
		return
	}

	token := appmw.NewConnectSessionToken(h.sessionSecret, cu.ID, connectSessionTTL)
	writeJSON(w, http.StatusOK, cacheNone, map[string]any{
		"token": token,
		"user":  connectUserDTO{ID: cu.ID, Name: cu.Name, Avatar: cu.AvatarUrl.String, IsAdmin: cu.IsAdmin},
	})
}

// APIConnectMe returns the chat identity behind the Bearer token.
func (h *Handler) APIConnectMe(w http.ResponseWriter, r *http.Request) {
	u := appmw.CurrentChatUser(r)
	if u == nil {
		writeJSONError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	writeJSON(w, http.StatusOK, cacheNone, map[string]any{
		"user": connectUserDTO{ID: u.ID, Name: u.Name, Avatar: u.AvatarURL, IsAdmin: u.IsAdmin},
	})
}

// APIConnectPost stores a message from the Bearer-authenticated chat user. It applies the
// same sanitization, length cap, and ban check as the web ConnectPost, and returns the
// stored message so the app can render it optimistically.
func (h *Handler) APIConnectPost(w http.ResponseWriter, r *http.Request) {
	u := appmw.CurrentChatUser(r)
	if u == nil {
		writeJSONError(w, http.StatusUnauthorized, "sign in required")
		return
	}
	if u.IsBanned {
		writeJSONError(w, http.StatusForbidden, "your account is blocked from chat")
		return
	}
	if h.q == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "chat unavailable")
		return
	}
	if !h.chatEnabled(r) {
		writeJSONError(w, http.StatusForbidden, "chat is currently disabled")
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body := chatSanitizer.Sanitize(strings.TrimSpace(req.Body))
	if body == "" {
		writeJSONError(w, http.StatusBadRequest, "message is empty")
		return
	}
	if len(body) > chatMaxBody {
		body = body[:chatMaxBody]
	}

	res, err := h.q.CreateChatMessage(r.Context(), sqlc.CreateChatMessageParams{ChatUserID: u.ID, Body: body})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not send message")
		return
	}
	id, _ := res.LastInsertId()
	vm := chatMessageVM{
		ID: uint64(id), UserID: u.ID, Name: u.Name, Avatar: u.AvatarURL, IsAdmin: u.IsAdmin,
		Body: body, Time: formatChatTime(time.Now()),
	}
	h.chat.Append(vm)
	writeJSON(w, http.StatusOK, cacheNone, map[string]any{"message": vm})
}
