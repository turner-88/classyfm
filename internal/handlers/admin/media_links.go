package admin

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// mediaPlatforms is the fixed set of rows in media_links - the table is seeded by
// migration and never grows at runtime, so the form renders one field per entry
// here and anything else posted is ignored. The order is the display order: the
// query sorts by the enum column, which is alphabetical and meaningless to the
// editor, so the page follows this slice instead.
var mediaPlatforms = []struct {
	Key         string
	Label       string
	Placeholder string
}{
	{"instagram", "Instagram", "https://www.instagram.com/classyfm"},
	{"facebook", "Facebook", "https://www.facebook.com/classyfm"},
	{"x", "X", "https://x.com/classyfm"},
	{"youtube", "YouTube", "https://www.youtube.com/@classyfm"},
	{"spotify", "Spotify", "https://open.spotify.com/user/classyfm"},
	{"tiktok", "TikTok", "https://www.tiktok.com/@classyfm"},
}

// mediaLinkRow is one platform as the form renders it: the stored (or just-posted)
// URL plus the labelling the template would otherwise have to pick out of an
// if-chain over the platform key.
type mediaLinkRow struct {
	Platform    string
	Label       string
	Placeholder string
	URL         string
	Invalid     bool
}

type mediaLinksListData struct {
	Base  baseData
	Rows  []mediaLinkRow
	Error string
}

// mediaLinkRows builds the six rows in display order. value returns the URL to
// show for a platform, which is the stored one on a plain page load and the
// just-posted one when re-rendering after a validation failure - retyping the
// other five fields because one was wrong is not a thing anyone should do.
func mediaLinkRows(value func(platform string) string) []mediaLinkRow {
	rows := make([]mediaLinkRow, 0, len(mediaPlatforms))
	for _, p := range mediaPlatforms {
		rows = append(rows, mediaLinkRow{
			Platform:    p.Key,
			Label:       p.Label,
			Placeholder: p.Placeholder,
			URL:         value(p.Key),
		})
	}
	return rows
}

// storedMediaLinks reads the current rows as a platform -> URL map.
func (h *Handler) storedMediaLinks(r *http.Request) (map[string]string, error) {
	links, err := h.q.ListMediaLinks(r.Context())
	if err != nil {
		return nil, err
	}
	stored := make(map[string]string, len(links))
	for _, l := range links {
		stored[string(l.Platform)] = l.Url
	}
	return stored, nil
}

// validMediaURL accepts an absolute http(s) URL, or nothing at all - an empty
// field is how a footer icon is hidden. Anything else (ftp:, javascript:, a bare
// handle) would end up in an href on the public site, and type="url" alone does
// not stop it.
func validMediaURL(raw string) bool {
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// MediaLinksList renders the 6 fixed platform links (Instagram/Facebook/X/YouTube/Spotify/TikTok)
// as a single form.
func (h *Handler) MediaLinksList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	stored, err := h.storedMediaLinks(r)
	if err != nil {
		http.Error(w, "failed to load media links", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/media_links_list", mediaLinksListData{
		Base: h.base(r, "Media", "media"),
		Rows: mediaLinkRows(func(platform string) string { return stored[platform] }),
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

	stored, err := h.storedMediaLinks(r)
	if err != nil {
		http.Error(w, "failed to load media links", http.StatusInternalServerError)
		return
	}

	posted := make(map[string]string, len(mediaPlatforms))
	for _, p := range mediaPlatforms {
		posted[p.Key] = strings.TrimSpace(r.FormValue("url_" + p.Key))
	}

	// Every field is checked before anything is written, so one bad URL cannot
	// leave half the platforms saved and half not.
	rows := mediaLinkRows(func(platform string) string { return posted[platform] })
	var bad []string
	for i := range rows {
		if !validMediaURL(rows[i].URL) {
			rows[i].Invalid = true
			bad = append(bad, rows[i].Label)
		}
	}
	if len(bad) > 0 {
		h.r.Page(w, http.StatusBadRequest, "admin/media_links_list", mediaLinksListData{
			Base:  h.base(r, "Media", "media"),
			Rows:  rows,
			Error: "Enter a full web address starting with http:// or https:// for: " + strings.Join(bad, ", ") + ".",
		})
		return
	}

	var changed []string
	for _, p := range mediaPlatforms {
		if prev, ok := stored[p.Key]; !ok || prev == posted[p.Key] {
			continue
		}
		if err := h.q.UpdateMediaLinkURL(r.Context(), sqlc.UpdateMediaLinkURLParams{
			Url:      posted[p.Key],
			Platform: sqlc.MediaLinksPlatform(p.Key),
		}); err != nil {
			http.Error(w, "failed to save media link", http.StatusInternalServerError)
			return
		}
		changed = append(changed, p.Key)
	}

	if len(changed) > 0 {
		h.audit(r, "update", "media_link", nil, "Updated media links: "+strings.Join(changed, ", "))
	}
	h.flash(w, "Media links saved.")
	http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
}
