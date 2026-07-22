package admin

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	appmw "github.com/classyfm/classyfm/internal/middleware"
	"github.com/classyfm/classyfm/internal/radio"
)

const sessionTTL = 7 * 24 * time.Hour

// virtualSessionTTL is deliberately much shorter than sessionTTL: the break-glass
// root login is an emergency credential, not a day-to-day account.
const virtualSessionTTL = 1 * time.Hour

type loginData struct {
	Base  baseData
	Error string
	Email string
	Reset bool
}

// LoginPage renders the admin login form.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if appmw.CurrentUser(r) != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/login", loginData{
		Base:  h.guestBase(r, "Sign In"),
		Reset: r.URL.Query().Get("reset") == "1",
	})
}

// Login verifies credentials and starts a session.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	fail := func(status int, msg string) {
		h.r.Page(w, status, "admin/login", loginData{Base: h.guestBase(r, "Sign In"), Error: msg, Email: email})
	}

	if email == "" || password == "" {
		fail(http.StatusBadRequest, "Email and password are required.")
		return
	}

	// Checked before the DB-availability guard below: the break-glass root login
	// exists precisely for cases where the database or the users table itself is
	// unusable, so it must not depend on h.q.
	if email == appmw.VirtualRootEmail {
		h.virtualRootLogin(w, r, password, fail)
		return
	}

	if h.q == nil {
		fail(http.StatusServiceUnavailable, "Database unavailable.")
		return
	}

	user, err := h.q.GetUserByEmail(r.Context(), email)
	if err != nil {
		fail(http.StatusUnauthorized, "Incorrect email or password.")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		fail(http.StatusUnauthorized, "Incorrect email or password.")
		return
	}
	if !user.IsActive {
		fail(http.StatusUnauthorized, "Incorrect email or password.")
		return
	}

	token, err := randomToken()
	if err != nil {
		fail(http.StatusInternalServerError, "Something went wrong, try again.")
		return
	}
	expires := time.Now().Add(sessionTTL)
	if err := h.q.CreateSession(r.Context(), sqlc.CreateSessionParams{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: expires,
	}); err != nil {
		fail(http.StatusInternalServerError, "Something went wrong, try again.")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     appmw.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})
	h.auditAs(r, user.ID, user.Email, "login", "user", &user.ID, "Logged into admin panel")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// virtualRootLogin handles the break-glass superadmin login: email
// appmw.VirtualRootEmail authenticates with the SESSION_SECRET env var as the
// password instead of a bcrypt hash from the users table. On success it issues a
// self-contained signed session cookie (see appmw.NewVirtualSessionToken) rather
// than a DB-backed session row, since the virtual user has no users row to attach
// one to. Deliberately never calls h.auditAs: this identity is excluded from
// audit_logs entirely. Both outcomes are still recorded via slog for ops
// visibility (server logs only, never the admin-visible audit trail).
func (h *Handler) virtualRootLogin(w http.ResponseWriter, r *http.Request, password string, fail func(int, string)) {
	ip := requestIP(r)
	ok := h.sessionSecret != "" && subtle.ConstantTimeCompare([]byte(password), []byte(h.sessionSecret)) == 1
	if !ok {
		slog.Warn("virtual root login failed", "ip", ip)
		fail(http.StatusUnauthorized, "Incorrect email or password.")
		return
	}

	token := appmw.NewVirtualSessionToken(h.sessionSecret, virtualSessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     appmw.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(virtualSessionTTL),
	})
	slog.Warn("virtual root login succeeded", "ip", ip)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func requestIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// Logout destroys the current session.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	u := appmw.CurrentUser(r)
	if u != nil {
		h.audit(r, "logout", "user", &u.ID, "Logged out of admin panel")
	}
	if cookie, err := r.Cookie(appmw.SessionCookieName); err == nil && h.q != nil && (u == nil || !u.Virtual) {
		_ = h.q.DeleteSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     appmw.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// Dashboard renders the admin landing page.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	var programCount, hotReleaseCount, newsfeedCount int
	var feedSources []sqlc.FeedSource
	var recentActivity []sqlc.AuditLog
	if h.q != nil {
		if n, err := h.q.CountPrograms(r.Context(), sqlc.CountProgramsParams{Search: "%"}); err == nil {
			programCount = int(n)
		}
		if n, err := h.q.CountAllHotRelease(r.Context(), "%"); err == nil {
			hotReleaseCount = int(n)
		}
		if n, err := h.q.CountAggregatedNews(r.Context(), sqlc.CountAggregatedNewsParams{Search: "%"}); err == nil {
			newsfeedCount = int(n)
		}
		if sources, err := h.q.ListFeedSources(r.Context()); err == nil {
			feedSources = sources
		}
		if u := appmw.CurrentUser(r); u != nil && u.Role == "superadmin" {
			if logs, err := h.q.ListAuditLogs(r.Context(), sqlc.ListAuditLogsParams{Search: "%", Limit: 5, Offset: 0}); err == nil {
				recentActivity = logs
			}
		}
	}
	h.r.Page(w, http.StatusOK, "admin/dashboard", struct {
		Base            baseData
		ProgramCount    int
		HotReleaseCount int
		NewsfeedCount   int
		FeedSources     []sqlc.FeedSource
		RecentActivity  []sqlc.AuditLog
		NowPlaying      radio.NowPlaying
	}{
		h.base(r, "Dashboard", "dashboard"), programCount, hotReleaseCount, newsfeedCount,
		feedSources, recentActivity, h.radio.Current(r.Context()),
	})
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
