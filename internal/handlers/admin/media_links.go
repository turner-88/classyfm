package admin

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

type mediaLinksListData struct {
	Base  baseData
	Links []sqlc.MediaLink
}

// MediaLinksList renders the 4 fixed platform badges (Instagram/Facebook/X/YouTube),
// each editable in place.
func (h *Handler) MediaLinksList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	links, err := h.q.ListMediaLinks(r.Context())
	if err != nil {
		http.Error(w, "gagal memuat tautan media", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/media_links_list", mediaLinksListData{
		Base:  h.base(r, "Media", "media"),
		Links: links,
	})
}

// MediaLinkUpdate saves the account URL for one platform.
func (h *Handler) MediaLinkUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	platform := chi.URLParam(r, "platform")
	switch platform {
	case "instagram", "facebook", "x", "youtube":
	default:
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	err := h.q.UpdateMediaLinkURL(r.Context(), sqlc.UpdateMediaLinkURLParams{
		Url:      strings.TrimSpace(r.FormValue("url")),
		Platform: sqlc.MediaLinksPlatform(platform),
	})
	if err != nil {
		http.Error(w, "gagal menyimpan tautan media", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "media_link", nil, "Mengubah tautan media "+platform)
	http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
}
