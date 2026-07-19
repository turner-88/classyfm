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
	h.r.Page(w, http.StatusOK, "admin/forgot_password", forgotPasswordData{Base: h.guestBase(r, "Lupa Kata Sandi")})
}

// ForgotPasswordSubmit issues a reset token and emails a reset link, if the
// address belongs to a real user. Always shows the same generic confirmation
// regardless of whether the address was found, to avoid leaking which emails
// have accounts.
func (h *Handler) ForgotPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))

	if h.q == nil {
		h.r.Page(w, http.StatusServiceUnavailable, "admin/forgot_password", forgotPasswordData{
			Base: h.guestBase(r, "Lupa Kata Sandi"), Error: "Database tidak tersedia.",
		})
		return
	}
	if email == "" {
		h.r.Page(w, http.StatusBadRequest, "admin/forgot_password", forgotPasswordData{
			Base: h.guestBase(r, "Lupa Kata Sandi"), Error: "Email wajib diisi.",
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
				body := "Halo " + user.Name + ",\n\n" +
					"Ada permintaan untuk mengatur ulang kata sandi akun admin Anda. Klik tautan berikut untuk membuat kata sandi baru (berlaku sampai " +
					expires.Format("02 Jan 2006 15:04") + "):\n\n" + link +
					"\n\nJika Anda tidak meminta ini, abaikan saja email ini.\n"
				if err := h.mailer.Send(user.Email, "Atur Ulang Kata Sandi — "+h.station, body); err != nil {
					slog.Error("send password reset email failed", "err", err, "user_id", user.ID)
				}
			}
		}
	}

	h.r.Page(w, http.StatusOK, "admin/forgot_password", forgotPasswordData{
		Base: h.guestBase(r, "Lupa Kata Sandi"), Sent: true, Email: email,
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
		Base: h.guestBase(r, "Atur Ulang Kata Sandi"), Token: token, Error: errMsg,
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
		h.renderResetPassword(w, r, token, "Database tidak tersedia.")
		return
	}
	if token == "" {
		h.renderResetPassword(w, r, token, "Tautan tidak valid.")
		return
	}
	if len(newPassword) < 8 {
		h.renderResetPassword(w, r, token, "Kata sandi baru minimal 8 karakter.")
		return
	}
	if newPassword != confirm {
		h.renderResetPassword(w, r, token, "Konfirmasi kata sandi tidak cocok.")
		return
	}

	rec, err := h.q.GetValidPasswordResetToken(r.Context(), hashResetToken(token))
	if err != nil {
		h.renderResetPassword(w, r, token, "Tautan tidak valid atau sudah kedaluwarsa. Minta tautan baru.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		h.renderResetPassword(w, r, token, "Gagal memproses kata sandi.")
		return
	}
	if err := h.q.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{PasswordHash: string(hash), ID: rec.UserID}); err != nil {
		h.renderResetPassword(w, r, token, "Gagal menyimpan kata sandi.")
		return
	}
	_ = h.q.MarkPasswordResetTokenUsed(r.Context(), hashResetToken(token))
	_ = h.q.DeleteSessionsByUserID(r.Context(), rec.UserID)
	h.auditAs(r, rec.UserID, "", "password_reset", "user", &rec.UserID, "Mengatur ulang kata sandi lewat tautan email")

	http.Redirect(w, r, "/admin/login?reset=1", http.StatusSeeOther)
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
