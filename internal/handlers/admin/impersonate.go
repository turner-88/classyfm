package admin

import (
	"log/slog"
	"net/http"
	"time"

	appmw "github.com/classyfm/classyfm/internal/middleware"
)

// impersonateTTL is deliberately short. Like the break-glass root cookie, an
// impersonation token is self-contained and verified without a sessions row, so
// there is no server-side revocation - only the expiry, deactivating the user, or
// rotating SESSION_SECRET ends it early.
const impersonateTTL = time.Hour

// Impersonate swaps the current session for one acting as the given user. Gated
// to the break-glass root by appmw.RequireVirtualRoot on the route, so a normal
// superadmin can never reach it. Never audited: root-initiated sessions stay out
// of audit_logs entirely (see Handler.audit), leaving only the slog trail below.
func (h *Handler) Impersonate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	user, err := h.q.GetUserByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !user.IsActive {
		http.Error(w, "Cannot impersonate a deactivated user.", http.StatusBadRequest)
		return
	}

	expires := time.Now().Add(impersonateTTL)
	h.setSessionCookie(w, appmw.NewImpersonationToken(h.sessionSecret, user.ID, impersonateTTL), expires)
	slog.Warn("impersonation started", "user_id", user.ID, "email", user.Email, "ip", requestIP(r))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// StopImpersonating ends an impersonation session by re-minting the root token
// the session came from. Lives outside the superadmin route group because the
// impersonated user may be a plain admin, hence the explicit Impersonated check.
func (h *Handler) StopImpersonating(w http.ResponseWriter, r *http.Request) {
	u := appmw.CurrentUser(r)
	if u == nil || !u.Impersonated {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	expires := time.Now().Add(virtualSessionTTL)
	h.setSessionCookie(w, appmw.NewVirtualSessionToken(h.sessionSecret, virtualSessionTTL), expires)
	slog.Warn("impersonation ended", "user_id", u.ID, "email", u.Email, "ip", requestIP(r))
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}
