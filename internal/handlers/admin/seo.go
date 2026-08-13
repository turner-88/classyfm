package admin

import (
	"net/http"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// seoFormData is the view-model for the single SEO settings form: the keyword
// list, a default meta description + social-share image used site-wide when a
// page sets none, and search-engine verification codes. All feed the public
// site's <head> (see internal/handlers/public.base and layouts/base.html).
type seoFormData struct {
	Base               baseData
	Keywords           string
	DefaultDescription string
	OgImageURL         string
	GoogleVerification string
	BingVerification   string
	Error              string
}

// SeoSettings renders the SEO settings form seeded with the stored values.
func (h *Handler) SeoSettings(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	s, err := h.q.GetSeoSettings(r.Context())
	if err != nil {
		http.Error(w, "failed to load SEO settings", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/seo_form", seoFormData{
		Base:               h.base(r, "SEO", "seo"),
		Keywords:           s.Keywords,
		DefaultDescription: s.DefaultDescription,
		OgImageURL:         s.OgImageUrl,
		GoogleVerification: s.GoogleVerification,
		BingVerification:   s.BingVerification,
	})
}

// SeoSettingsUpdate saves the single SEO settings row. The share image is a real
// file upload, so the body is a multipart form; the current image is carried
// forward via the hidden current_image_url field and overwritten only when a new
// file is submitted (mirroring broadcasterFromForm).
func (h *Handler) SeoSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	if err := parseUploadForm(w, r); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	data := seoFormData{
		Base:               h.base(r, "SEO", "seo"),
		Keywords:           strings.TrimSpace(r.FormValue("keywords")),
		DefaultDescription: strings.TrimSpace(r.FormValue("default_description")),
		OgImageURL:         strings.TrimSpace(r.FormValue("current_image_url")),
		GoogleVerification: strings.TrimSpace(r.FormValue("google_verification")),
		BingVerification:   strings.TrimSpace(r.FormValue("bing_verification")),
	}
	if url, err := h.saveUploadedImage(r, "og_image", uploadSubdirSeo); err != nil {
		data.Error = "The share image could not be uploaded. Use a JPG, PNG, WEBP, or GIF within the size limit."
		h.r.Page(w, http.StatusBadRequest, "admin/seo_form", data)
		return
	} else if url != "" {
		data.OgImageURL = url
	}

	if err := h.q.UpdateSeoSettings(r.Context(), sqlc.UpdateSeoSettingsParams{
		Keywords:           data.Keywords,
		DefaultDescription: data.DefaultDescription,
		OgImageUrl:         data.OgImageURL,
		GoogleVerification: data.GoogleVerification,
		BingVerification:   data.BingVerification,
	}); err != nil {
		http.Error(w, "failed to save SEO settings", http.StatusInternalServerError)
		return
	}

	h.audit(r, "update", "seo_settings", nil, "Updated SEO settings")
	h.flash(w, "SEO settings saved.")
	http.Redirect(w, r, "/admin/seo", http.StatusSeeOther)
}
