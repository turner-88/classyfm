package admin

import (
	"net/http"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

type aboutPageData struct {
	Base     baseData
	Banner   sqlc.AboutPageBanner
	Segments []sqlc.AboutPageSegment
	Error    string
}

// AboutPage renders the About Us banner and the 3 fixed text segments
// (profile/music/audience), each editable in place.
func (h *Handler) AboutPage(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	banner, err := h.q.GetAboutBanner(r.Context())
	if err != nil {
		http.Error(w, "failed to load about page", http.StatusInternalServerError)
		return
	}
	segments, err := h.q.ListAboutSegments(r.Context())
	if err != nil {
		http.Error(w, "failed to load about page", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/about", aboutPageData{
		Base:     h.base(r, "About Us", "about"),
		Banner:   banner,
		Segments: segments,
	})
}

// aboutSegments are the fixed segment keys, in the order the public page shows
// them. They are rows that always exist, so this is an allowlist rather than a
// lookup.
var aboutSegments = []string{"profile", "music", "audience"}

// AboutUpdate saves the whole About Us page in one submission: the banner (either
// an uploaded image or a video URL, chosen via media_type - both sub-fields are
// always submitted and only the one matching media_type is persisted) plus the
// title/body of each fixed segment, whose fields are suffixed with the segment key.
func (h *Handler) AboutUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	renderErr := func(msg string) {
		banner, _ := h.q.GetAboutBanner(r.Context())
		segments, _ := h.q.ListAboutSegments(r.Context())
		h.r.Page(w, http.StatusBadRequest, "admin/about", aboutPageData{
			Base:     h.base(r, "About Us", "about"),
			Banner:   banner,
			Segments: segments,
			Error:    msg,
		})
	}

	if err := parseUploadForm(w, r); err != nil {
		renderErr(err.Error())
		return
	}

	mediaType := strings.TrimSpace(r.FormValue("media_type"))
	switch mediaType {
	case "image", "video":
	default:
		renderErr("Invalid banner type.")
		return
	}

	imageURL := strings.TrimSpace(r.FormValue("current_image_url"))
	if uploaded, err := h.saveUploadedImage(r, "image", uploadSubdirAbout); err != nil {
		renderErr("Failed to upload banner image: " + err.Error())
		return
	} else if uploaded != "" {
		imageURL = uploaded
	}

	err := h.q.UpdateAboutBanner(r.Context(), sqlc.UpdateAboutBannerParams{
		MediaType: sqlc.AboutPageBannerMediaType(mediaType),
		ImageUrl:  toNullString(imageURL),
		VideoUrl:  toNullString(strings.TrimSpace(r.FormValue("video_url"))),
	})
	if err != nil {
		http.Error(w, "failed to save banner", http.StatusInternalServerError)
		return
	}

	for _, segment := range aboutSegments {
		if err := h.q.UpdateAboutSegment(r.Context(), sqlc.UpdateAboutSegmentParams{
			Title:   strings.TrimSpace(r.FormValue("seg_title_" + segment)),
			Body:    strings.TrimSpace(r.FormValue("seg_body_" + segment)),
			Segment: sqlc.AboutPageSegmentsSegment(segment),
		}); err != nil {
			http.Error(w, "failed to save segment", http.StatusInternalServerError)
			return
		}
	}

	h.audit(r, "update", "about_page", nil, "Updated About Us page")
	h.flash(w, "About Us saved.")
	http.Redirect(w, r, "/admin/about", http.StatusSeeOther)
}
