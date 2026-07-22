package admin

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

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

// AboutBannerUpdate saves the About Us banner: either an uploaded image or a
// video URL (e.g. a YouTube link), chosen via media_type. Both sub-fields are
// always submitted; only the one matching media_type is persisted.
func (h *Handler) AboutBannerUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	_ = r.ParseMultipartForm(maxUploadBytes + 1<<20)

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
	videoURL := strings.TrimSpace(r.FormValue("video_url"))

	err := h.q.UpdateAboutBanner(r.Context(), sqlc.UpdateAboutBannerParams{
		MediaType: sqlc.AboutPageBannerMediaType(mediaType),
		ImageUrl:  toNullString(imageURL),
		VideoUrl:  toNullString(videoURL),
	})
	if err != nil {
		http.Error(w, "failed to save banner", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "about_banner", nil, "Updated About Us banner")
	http.Redirect(w, r, "/admin/about", http.StatusSeeOther)
}

// AboutSegmentUpdate saves the title/body for one of the 3 fixed segments.
func (h *Handler) AboutSegmentUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	segment := chi.URLParam(r, "segment")
	switch segment {
	case "profile", "music", "audience":
	default:
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	err := h.q.UpdateAboutSegment(r.Context(), sqlc.UpdateAboutSegmentParams{
		Title:   strings.TrimSpace(r.FormValue("title")),
		Body:    strings.TrimSpace(r.FormValue("body")),
		Segment: sqlc.AboutPageSegmentsSegment(segment),
	})
	if err != nil {
		http.Error(w, "failed to save segment", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "about_segment", nil, "Updated About Us segment "+segment)
	http.Redirect(w, r, "/admin/about", http.StatusSeeOther)
}
