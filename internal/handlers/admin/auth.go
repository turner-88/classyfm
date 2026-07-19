package admin

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	appmw "github.com/classyfm/classyfm/internal/middleware"
	"github.com/classyfm/classyfm/internal/radio"
)

const sessionTTL = 7 * 24 * time.Hour

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
		Base:  h.guestBase(r, "Masuk"),
		Reset: r.URL.Query().Get("reset") == "1",
	})
}

// Login verifies credentials and starts a session.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	fail := func(status int, msg string) {
		h.r.Page(w, status, "admin/login", loginData{Base: h.guestBase(r, "Masuk"), Error: msg, Email: email})
	}

	if h.q == nil {
		fail(http.StatusServiceUnavailable, "Database tidak tersedia.")
		return
	}
	if email == "" || password == "" {
		fail(http.StatusBadRequest, "Email dan kata sandi wajib diisi.")
		return
	}

	user, err := h.q.GetUserByEmail(r.Context(), email)
	if err != nil {
		fail(http.StatusUnauthorized, "Email atau kata sandi salah.")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		fail(http.StatusUnauthorized, "Email atau kata sandi salah.")
		return
	}

	token, err := randomToken()
	if err != nil {
		fail(http.StatusInternalServerError, "Terjadi kesalahan, coba lagi.")
		return
	}
	expires := time.Now().Add(sessionTTL)
	if err := h.q.CreateSession(r.Context(), sqlc.CreateSessionParams{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: expires,
	}); err != nil {
		fail(http.StatusInternalServerError, "Terjadi kesalahan, coba lagi.")
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
	h.auditAs(r, user.ID, user.Email, "login", "user", &user.ID, "Masuk ke admin panel")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// Logout destroys the current session.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if u := appmw.CurrentUser(r); u != nil {
		h.audit(r, "logout", "user", &u.ID, "Keluar dari admin panel")
	}
	if cookie, err := r.Cookie(appmw.SessionCookieName); err == nil && h.q != nil {
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
		if programs, err := h.q.ListPrograms(r.Context()); err == nil {
			programCount = len(programs)
		}
		if items, err := h.q.ListAllHotRelease(r.Context()); err == nil {
			hotReleaseCount = len(items)
		}
		if items, err := h.q.ListAggregatedNews(r.Context()); err == nil {
			newsfeedCount = len(items)
		}
		if sources, err := h.q.ListFeedSources(r.Context()); err == nil {
			feedSources = sources
		}
		if u := appmw.CurrentUser(r); u != nil && u.Role == "superadmin" {
			if logs, err := h.q.ListAuditLogs(r.Context(), sqlc.ListAuditLogsParams{Limit: 5, Offset: 0}); err == nil {
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
