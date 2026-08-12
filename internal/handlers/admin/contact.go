package admin

import (
	"net/http"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// contactFormData is the view-model for the single Contact settings form: the
// WhatsApp number + prefilled message (used by the floating WhatsApp widget) and
// the phone/email shown in the footer's "Get in touch" block. Error is set when a
// submission fails validation, in which case the submitted values are echoed back.
type contactFormData struct {
	Base            baseData
	WhatsAppNumber  string
	WhatsAppMessage string
	Phone           string
	Email           string
	Error           string
}

// ContactSettings renders the Contact settings form seeded with the stored values.
func (h *Handler) ContactSettings(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	c, err := h.q.GetContactSettings(r.Context())
	if err != nil {
		http.Error(w, "failed to load contact settings", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/contact_form", contactFormData{
		Base:            h.base(r, "Contact", "contact"),
		WhatsAppNumber:  c.WhatsappNumber,
		WhatsAppMessage: c.WhatsappMessage,
		Phone:           c.Phone,
		Email:           c.Email,
	})
}

// ContactSettingsUpdate validates and saves the single Contact settings row.
func (h *Handler) ContactSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	_ = r.ParseForm()

	data := contactFormData{
		Base:            h.base(r, "Contact", "contact"),
		WhatsAppNumber:  strings.TrimSpace(r.FormValue("whatsapp_number")),
		WhatsAppMessage: strings.TrimSpace(r.FormValue("whatsapp_message")),
		Phone:           strings.TrimSpace(r.FormValue("phone")),
		Email:           strings.TrimSpace(r.FormValue("email")),
	}

	// Each field is optional (an empty value simply hides its footer line / the
	// WhatsApp widget), but a value that is present must be usable: a WhatsApp
	// number needs at least a few digits to form a wa.me link, and an email must
	// look like an address since it lands in a mailto: href on the public site.
	if data.WhatsAppNumber != "" && countDigits(data.WhatsAppNumber) < 6 {
		data.Error = "Enter a valid WhatsApp number including the country code, or leave it blank."
	} else if data.Email != "" && !strings.Contains(data.Email, "@") {
		data.Error = "Enter a valid email address, or leave it blank."
	}
	if data.Error != "" {
		h.r.Page(w, http.StatusBadRequest, "admin/contact_form", data)
		return
	}

	if err := h.q.UpdateContactSettings(r.Context(), sqlc.UpdateContactSettingsParams{
		WhatsappNumber:  data.WhatsAppNumber,
		WhatsappMessage: data.WhatsAppMessage,
		Phone:           data.Phone,
		Email:           data.Email,
	}); err != nil {
		http.Error(w, "failed to save contact settings", http.StatusInternalServerError)
		return
	}

	h.audit(r, "update", "contact_settings", nil, "Updated contact settings")
	h.flash(w, "Contact settings saved.")
	http.Redirect(w, r, "/admin/contact", http.StatusSeeOther)
}

// countDigits returns how many ASCII digits are in s, used to sanity-check a
// human-typed phone/WhatsApp number before it becomes a wa.me link.
func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}
