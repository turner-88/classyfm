package admin

import (
	"net/http"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// mediaPlatforms is the fixed set of rows in media_links - the table is seeded by
// migration and never grows at runtime, so the form renders one field per entry
// here and anything else posted is ignored.
var mediaPlatforms = []string{"instagram", "facebook", "x", "youtube", "spotify", "tiktok"}

type mediaLinksListData struct {
	Base  baseData
	Links []sqlc.MediaLink
}

// MediaLinksList renders the 6 fixed platform links (Instagram/Facebook/X/YouTube/Spotify/TikTok)
// as a single form.
func (h *Handler) MediaLinksList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	links, err := h.q.ListMediaLinks(r.Context())
	if err != nil {
		http.Error(w, "failed to load media links", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/media_links_list", mediaLinksListData{
		Base:  h.base(r, "Media", "media"),
		Links: links,
	})
}

// MediaLinksUpdate saves every platform's account URL in one submission. Only rows
// whose value actually changed are written, so re-saving an untouched form is a
// no-op rather than six UPDATEs and six audit entries.
func (h *Handler) MediaLinksUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	_ = r.ParseForm()

	current, err := h.q.ListMediaLinks(r.Context())
	if err != nil {
		http.Error(w, "failed to load media links", http.StatusInternalServerError)
		return
	}
	stored := make(map[string]string, len(current))
	for _, l := range current {
		stored[string(l.Platform)] = l.Url
	}

	var changed []string
	for _, platform := range mediaPlatforms {
		url := strings.TrimSpace(r.FormValue("url_" + platform))
		if prev, ok := stored[platform]; !ok || prev == url {
			continue
		}
		if err := h.q.UpdateMediaLinkURL(r.Context(), sqlc.UpdateMediaLinkURLParams{
			Url:      url,
			Platform: sqlc.MediaLinksPlatform(platform),
		}); err != nil {
			http.Error(w, "failed to save media link", http.StatusInternalServerError)
			return
		}
		changed = append(changed, platform)
	}

	if len(changed) > 0 {
		h.audit(r, "update", "media_link", nil, "Updated media links: "+strings.Join(changed, ", "))
	}
	h.flash(w, "Media links saved.")
	http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
}
