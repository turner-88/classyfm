package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

type forgotPasswordData struct {
	Base  baseData
	Sent  bool
	Error string
	Email string
}

// ForgotPasswordPage renders the "request a reset link" form.
func (h *Handler) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	h.r.Page(w, http.StatusOK, "admin/forgot_password", forgotPasswordData{Base: h.guestBase(r, "Forgot Password")})
}

// ForgotPasswordSubmit issues a reset token and emails a reset link, if the
// address belongs to a real user. Always shows the same generic confirmation
// regardless of whether the address was found, to avoid leaking which emails
// have accounts.
func (h *Handler) ForgotPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))

	if h.q == nil {
		h.r.Page(w, http.StatusServiceUnavailable, "admin/forgot_password", forgotPasswordData{
			Base: h.guestBase(r, "Forgot Password"), Error: "Database unavailable.",
		})
		return
	}
	if email == "" {
		h.r.Page(w, http.StatusBadRequest, "admin/forgot_password", forgotPasswordData{
			Base: h.guestBase(r, "Forgot Password"), Error: "Email is required.",
		})
		return
	}

	if user, err := h.q.GetUserByEmail(r.Context(), email); err == nil {
		if token, err := randomToken(); err == nil {
			hash := hashResetToken(token)
			expires := time.Now().Add(h.resetTokenTTL)
			if err := h.q.CreatePasswordResetToken(r.Context(), sqlc.CreatePasswordResetTokenParams{
				TokenHash: hash,
				UserID:    user.ID,
				ExpiresAt: expires,
			}); err != nil {
				slog.Error("create password reset token failed", "err", err, "user_id", user.ID)
			} else {
				link := h.siteURL + "/admin/reset-password?token=" + token
				body := "Hi " + user.Name + ",\n\n" +
					"There was a request to reset your admin account password. Click the link below to set a new password (valid until " +
					expires.Format("02 Jan 2006 15:04") + "):\n\n" + link +
					"\n\nIf you didn't request this, just ignore this email.\n"
				if err := h.mailer.Send(user.Email, "Reset Password — "+h.station, body); err != nil {
					slog.Error("send password reset email failed", "err", err, "user_id", user.ID)
				}
			}
		}
	}

	h.r.Page(w, http.StatusOK, "admin/forgot_password", forgotPasswordData{
		Base: h.guestBase(r, "Forgot Password"), Sent: true, Email: email,
	})
}

type resetPasswordData struct {
	Base  baseData
	Token string
	Error string
}

// ResetPasswordPage renders the "set a new password" form for a token from a
// reset-link email.
func (h *Handler) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	h.renderResetPassword(w, r, token, "")
}

func (h *Handler) renderResetPassword(w http.ResponseWriter, r *http.Request, token, errMsg string) {
	status := http.StatusOK
	if errMsg != "" {
		status = http.StatusBadRequest
	}
	h.r.Page(w, status, "admin/reset_password", resetPasswordData{
		Base: h.guestBase(r, "Reset Password"), Token: token, Error: errMsg,
	})
}

// ResetPasswordSubmit validates the token and sets a new password, then
// invalidates every existing session for that user (forces re-login
// everywhere, including wherever the account may have been compromised).
func (h *Handler) ResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	newPassword := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if h.q == nil {
		h.renderResetPassword(w, r, token, "Database unavailable.")
		return
	}
	if token == "" {
		h.renderResetPassword(w, r, token, "Invalid link.")
		return
	}
	if len(newPassword) < 8 {
		h.renderResetPassword(w, r, token, "New password must be at least 8 characters.")
		return
	}
	if newPassword != confirm {
		h.renderResetPassword(w, r, token, "Password confirmation does not match.")
		return
	}

	rec, err := h.q.GetValidPasswordResetToken(r.Context(), hashResetToken(token))
	if err != nil {
		h.renderResetPassword(w, r, token, "Invalid or expired link. Request a new one.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		h.renderResetPassword(w, r, token, "Failed to process password.")
		return
	}
	if err := h.q.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{PasswordHash: string(hash), ID: rec.UserID}); err != nil {
		h.renderResetPassword(w, r, token, "Failed to save password.")
		return
	}
	_ = h.q.MarkPasswordResetTokenUsed(r.Context(), hashResetToken(token))
	_ = h.q.DeleteSessionsByUserID(r.Context(), rec.UserID)
	h.auditAs(r, rec.UserID, "", "password_reset", "user", &rec.UserID, "Reset password via email link")

	http.Redirect(w, r, "/admin/login?reset=1", http.StatusSeeOther)
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
