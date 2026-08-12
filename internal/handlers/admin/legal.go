package admin

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// legalPages is the fixed set of rows in legal_pages - seeded by migration and
// never grows at runtime, so this drives both the list page and the slug
// allowlist. PublicURL lets the templates offer a "View public page" link.
var legalPages = []struct {
	Slug      string
	Label     string
	PublicURL string
}{
	{"privacy", "Kebijakan Privasi", "/privacy-policy"},
	{"terms", "Syarat dan Ketentuan", "/terms-and-conditions"},
}

// legalPageMeta returns the fixed metadata for a slug, or ok=false if the slug is
// not one of the known legal pages.
func legalPageMeta(slug string) (label, publicURL string, ok bool) {
	for _, p := range legalPages {
		if p.Slug == slug {
			return p.Label, p.PublicURL, true
		}
	}
	return "", "", false
}

type legalListRow struct {
	Slug      string
	Label     string
	PublicURL string
	Title     string
}

type legalListData struct {
	Base baseData
	Rows []legalListRow
}

type legalFormData struct {
	Base      baseData
	Slug      string
	Label     string
	PublicURL string
	Title     string
	Intro     string
	Body      string
	Error     string
}

// LegalPagesList renders the two legal pages (Privacy Policy, Terms & Conditions),
// each linking to its own edit form.
func (h *Handler) LegalPagesList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	pages, err := h.q.ListLegalPages(r.Context())
	if err != nil {
		http.Error(w, "failed to load legal pages", http.StatusInternalServerError)
		return
	}
	titles := make(map[string]string, len(pages))
	for _, p := range pages {
		titles[string(p.Slug)] = p.Title
	}
	rows := make([]legalListRow, 0, len(legalPages))
	for _, p := range legalPages {
		rows = append(rows, legalListRow{
			Slug:      p.Slug,
			Label:     p.Label,
			PublicURL: p.PublicURL,
			Title:     titles[p.Slug],
		})
	}
	h.r.Page(w, http.StatusOK, "admin/legal_list", legalListData{
		Base: h.base(r, "Legal Pages", "legal"),
		Rows: rows,
	})
}

// LegalPageEdit renders the edit form for one legal page. The slug is validated
// against the fixed allowlist so an unknown /admin/legal/{slug} 404s rather than
// querying for a row that can't exist.
func (h *Handler) LegalPageEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	slug := chi.URLParam(r, "slug")
	label, publicURL, ok := legalPageMeta(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	page, err := h.q.GetLegalPage(r.Context(), sqlc.LegalPagesSlug(slug))
	if err != nil {
		http.Error(w, "failed to load legal page", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/legal_form", legalFormData{
		Base:      h.base(r, label, "legal"),
		Slug:      slug,
		Label:     label,
		PublicURL: publicURL,
		Title:     page.Title,
		Intro:     page.Intro,
		Body:      page.Body,
	})
}

// LegalPageUpdate saves one legal page's title, hero subtitle, and Markdown body.
func (h *Handler) LegalPageUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	slug := chi.URLParam(r, "slug")
	label, publicURL, ok := legalPageMeta(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()

	title := strings.TrimSpace(r.FormValue("title"))
	intro := strings.TrimSpace(r.FormValue("intro"))
	// Body is Markdown; trim only the surrounding whitespace, keep internal
	// newlines that structure the document.
	body := strings.TrimSpace(r.FormValue("body"))

	if title == "" {
		h.r.Page(w, http.StatusBadRequest, "admin/legal_form", legalFormData{
			Base:      h.base(r, label, "legal"),
			Slug:      slug,
			Label:     label,
			PublicURL: publicURL,
			Title:     title,
			Intro:     intro,
			Body:      body,
			Error:     "Title is required.",
		})
		return
	}

	if err := h.q.UpdateLegalPage(r.Context(), sqlc.UpdateLegalPageParams{
		Title: title,
		Intro: intro,
		Body:  body,
		Slug:  sqlc.LegalPagesSlug(slug),
	}); err != nil {
		http.Error(w, "failed to save legal page", http.StatusInternalServerError)
		return
	}

	h.audit(r, "update", "legal_page", nil, "Updated legal page: "+label)
	h.flash(w, label+" saved.")
	http.Redirect(w, r, "/admin/legal/"+slug, http.StatusSeeOther)
}
