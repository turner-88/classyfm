package admin

import (
	"net/http"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// menuFormData is the view-model for the single Menu settings form: which of the
// public nav's News / Podcast / Event top-level items are shown. Hiding an item
// only removes it from the header menu; the underlying page route still works.
type menuFormData struct {
	Base        baseData
	ShowNews    bool
	ShowPodcast bool
	ShowEvent   bool
}

// MenuSettings renders the Menu settings form seeded with the stored toggles.
func (h *Handler) MenuSettings(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	m, err := h.q.GetMenuSettings(r.Context())
	if err != nil {
		http.Error(w, "failed to load menu settings", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/menu_form", menuFormData{
		Base:        h.base(r, "Menu", "menu"),
		ShowNews:    m.ShowNews,
		ShowPodcast: m.ShowPodcast,
		ShowEvent:   m.ShowEvent,
	})
}

// MenuSettingsUpdate saves the single Menu settings row. Unchecked checkboxes
// submit no value, so a missing field reads as off.
func (h *Handler) MenuSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	_ = r.ParseForm()

	arg := sqlc.UpdateMenuSettingsParams{
		ShowNews:    r.FormValue("show_news") == "on",
		ShowPodcast: r.FormValue("show_podcast") == "on",
		ShowEvent:   r.FormValue("show_event") == "on",
	}
	if err := h.q.UpdateMenuSettings(r.Context(), arg); err != nil {
		http.Error(w, "failed to save menu settings", http.StatusInternalServerError)
		return
	}

	h.audit(r, "update", "menu_settings", nil, "Updated menu settings")
	h.flash(w, "Menu settings saved.")
	http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
}
