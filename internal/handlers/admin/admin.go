// Package admin holds HTTP handlers for the admin panel: authentication and CRUD
// screens that manage exactly the data the public site consumes.
package admin

import (
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/mail"
	appmw "github.com/classyfm/classyfm/internal/middleware"
	"github.com/classyfm/classyfm/internal/radio"
	"github.com/classyfm/classyfm/internal/render"
)

// Handler renders the admin pages and handles admin form submissions.
type Handler struct {
	r             *render.Renderer
	q             *sqlc.Queries
	worker        feedSourceUpdater // nil if no database is configured
	radio         *radio.Service
	station       string
	secure        bool   // whether to mark the session cookie Secure (true in production/HTTPS)
	uploadDir     string // on-disk directory for user-uploaded files (e.g. program banners)
	mailer        *mail.Mailer
	siteURL       string
	resetTokenTTL time.Duration
}

// New constructs the admin handler. q and worker may be nil if no database is
// configured, in which case admin handlers report the panel as unavailable rather
// than panicking. mailer may be unconfigured (see mail.Mailer.Configured), in
// which case password-reset requests are accepted but no email is actually sent.
func New(r *render.Renderer, q *sqlc.Queries, worker feedSourceUpdater, radioSvc *radio.Service, station string, secure bool, uploadDir string, mailer *mail.Mailer, siteURL string, resetTokenTTL time.Duration) *Handler {
	return &Handler{
		r: r, q: q, worker: worker, radio: radioSvc, station: station, secure: secure, uploadDir: uploadDir,
		mailer: mailer, siteURL: siteURL, resetTokenTTL: resetTokenTTL,
	}
}

// baseData is the common view-model every admin page embeds (used by the admin layout).
type baseData struct {
	Title       string
	Nav         string // active nav key: dashboard|programs
	StationName string
	UserName    string
	UserRole    string
	CSRFToken   string
	Guest       bool // true on pre-authentication pages (login, forgot/reset password): layout hides the sidebar/nav chrome
}

func (h *Handler) base(r *http.Request, title, nav string) baseData {
	name, role := "", ""
	if u := appmw.CurrentUser(r); u != nil {
		name = u.Name
		role = u.Role
	}
	return baseData{
		Title:       title,
		Nav:         nav,
		StationName: h.station,
		UserName:    name,
		UserRole:    role,
		CSRFToken:   appmw.CSRFToken(r),
	}
}

// guestBase is like base but marks the page as a guest (pre-authentication) page,
// so the admin layout hides the sidebar/nav chrome that assumes a logged-in session.
func (h *Handler) guestBase(r *http.Request, title string) baseData {
	b := h.base(r, title, "")
	b.Guest = true
	return b
}

// audit records one row in the audit_logs table for a successful admin-panel
// mutation, attributed to the request's current session user (via
// appmw.CurrentUser). entityID may be nil (e.g. logout has no single entity).
func (h *Handler) audit(r *http.Request, action, entityType string, entityID *uint64, detail string) {
	var userID uint64
	email := ""
	if u := appmw.CurrentUser(r); u != nil {
		userID = u.ID
		email = u.Email
	}
	h.auditAs(r, userID, email, action, entityType, entityID, detail)
}

// auditAs is the low-level form of audit for the one case where the acting
// user isn't yet attached to the request context: a successful Login, which
// authenticates and creates the session within the same request that must be
// audited. userID 0 records an anonymous/unattributed entry.
// Errors are logged and swallowed - auditing must never break the primary
// action it's attached to.
func (h *Handler) auditAs(r *http.Request, userID uint64, email, action, entityType string, entityID *uint64, detail string) {
	if h.q == nil {
		return
	}
	var uid sql.NullInt64
	if userID != 0 {
		uid = sql.NullInt64{Int64: int64(userID), Valid: true}
	}
	var entity sql.NullInt64
	if entityID != nil {
		entity = sql.NullInt64{Int64: int64(*entityID), Valid: true}
	}
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if err := h.q.CreateAuditLog(r.Context(), sqlc.CreateAuditLogParams{
		UserID:            uid,
		UserEmailSnapshot: email,
		Action:            action,
		EntityType:        entityType,
		EntityID:          entity,
		Detail:            detail,
		IpAddress:         ip,
	}); err != nil {
		slog.Error("audit log write failed", "err", err, "action", action, "entity_type", entityType)
	}
}

// unavailable renders a friendly message when the database isn't configured.
func (h *Handler) unavailable(w http.ResponseWriter, r *http.Request) bool {
	if h.q != nil {
		return false
	}
	http.Error(w, "Admin panel is unavailable: no database configured.", http.StatusServiceUnavailable)
	return true
}
