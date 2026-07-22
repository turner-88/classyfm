package admin

import (
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	appmw "github.com/classyfm/classyfm/internal/middleware"
)

type profileData struct {
	Base      baseData
	User      sqlc.User
	Error     string
	PassError string
	Success   string
}

// ProfilePage renders the logged-in user's own details and change-password form.
func (h *Handler) ProfilePage(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	h.renderProfile(w, r, "", "", "")
}

func (h *Handler) renderProfile(w http.ResponseWriter, r *http.Request, errMsg, passErr, success string) {
	u := appmw.CurrentUser(r)
	if u == nil {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}
	// The break-glass root login has no users row to load a profile from.
	if u.Virtual {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	user, err := h.q.GetUserByID(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "failed to load profile", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/profile", profileData{
		Base:      h.base(r, "Profile", "profile"),
		User:      user,
		Error:     errMsg,
		PassError: passErr,
		Success:   success,
	})
}

// ProfileUpdate saves the logged-in user's own name/email.
func (h *Handler) ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	u := appmw.CurrentUser(r)
	if u == nil {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}
	if u.Virtual {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if name == "" || email == "" {
		h.renderProfile(w, r, "Name and email are required.", "", "")
		return
	}
	if err := h.q.UpdateUser(r.Context(), sqlc.UpdateUserParams{Name: name, Email: email, Role: sqlc.UsersRole(u.Role), IsActive: true, ID: u.ID}); err != nil {
		h.renderProfile(w, r, friendlyDBError(err, "Email is already used by another user."), "", "")
		return
	}
	h.audit(r, "update", "user", &u.ID, "Updated own profile")
	http.Redirect(w, r, "/admin/profile", http.StatusSeeOther)
}

// ProfilePassword changes the logged-in user's own password.
func (h *Handler) ProfilePassword(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	u := appmw.CurrentUser(r)
	if u == nil {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}
	if u.Virtual {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	newPassword := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if len(newPassword) < 8 {
		h.renderProfile(w, r, "", "New password must be at least 8 characters.", "")
		return
	}
	if newPassword != confirm {
		h.renderProfile(w, r, "", "Password confirmation does not match.", "")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		h.renderProfile(w, r, "", "Failed to process password.", "")
		return
	}
	if err := h.q.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{PasswordHash: string(hash), ID: u.ID}); err != nil {
		h.renderProfile(w, r, "", "Failed to save password.", "")
		return
	}
	h.audit(r, "password_change", "user", &u.ID, "Changed own password")
	h.renderProfile(w, r, "", "", "Password changed successfully.")
}
