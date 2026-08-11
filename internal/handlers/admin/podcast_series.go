package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

type podcastSeriesListData struct {
	Base   baseData
	Series []sqlc.PodcastSeries
}

// PodcastSeriesList renders every podcast series (a small, unpaginated set).
func (h *Handler) PodcastSeriesList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	series, err := h.q.ListPodcastSeries(r.Context())
	if err != nil {
		http.Error(w, "failed to load podcast series", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/podcast_series_list", podcastSeriesListData{
		Base:   h.base(r, "Podcast Series", "podcast-series"),
		Series: series,
	})
}

type podcastSeriesFormData struct {
	Base   baseData
	IsNew  bool
	Series sqlc.PodcastSeries
	Error  string
}

// PodcastSeriesNew renders the create form.
func (h *Handler) PodcastSeriesNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	h.r.Page(w, http.StatusOK, "admin/podcast_series_form", podcastSeriesFormData{
		Base:   h.base(r, "New Podcast Series", "podcast-series"),
		IsNew:  true,
		Series: sqlc.PodcastSeries{IsActive: true},
	})
}

// PodcastSeriesCreate inserts a new series.
func (h *Handler) PodcastSeriesCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	s, formErr := podcastSeriesFromForm(r)

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/podcast_series_form", podcastSeriesFormData{
			Base:   h.base(r, "New Podcast Series", "podcast-series"),
			IsNew:  true,
			Series: s,
			Error:  msg,
		})
	}

	if formErr != "" {
		renderErr(formErr)
		return
	}

	s.Slug = h.uniquePodcastSeriesSlug(r.Context(), slugify(s.Name), 0)
	res, err := h.q.CreatePodcastSeries(r.Context(), sqlc.CreatePodcastSeriesParams{
		Name: s.Name, Slug: s.Slug, SortOrder: s.SortOrder, IsActive: s.IsActive,
	})
	if err != nil {
		renderErr("Failed to save series: " + err.Error())
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	h.audit(r, "create", "podcast_series", &uid, "Created podcast series "+s.Name)
	h.flash(w, "Series created.")
	http.Redirect(w, r, "/admin/podcast-series", http.StatusSeeOther)
}

// PodcastSeriesEdit renders the edit form.
func (h *Handler) PodcastSeriesEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s, err := h.q.GetPodcastSeries(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/podcast_series_form", podcastSeriesFormData{
		Base:   h.base(r, "Edit Podcast Series", "podcast-series"),
		IsNew:  false,
		Series: s,
	})
}

// PodcastSeriesUpdate saves edits to an existing series.
func (h *Handler) PodcastSeriesUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	existing, err := h.q.GetPodcastSeries(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s, formErr := podcastSeriesFromForm(r)
	s.ID = id

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/podcast_series_form", podcastSeriesFormData{
			Base:   h.base(r, "Edit Podcast Series", "podcast-series"),
			IsNew:  false,
			Series: s,
			Error:  msg,
		})
	}

	if formErr != "" {
		renderErr(formErr)
		return
	}

	// The slug follows the name but only changes when the name does, so an existing
	// public ?series= filter URL stays stable across sort-order edits.
	s.Slug = existing.Slug
	if slugify(s.Name) != slugify(existing.Name) {
		s.Slug = h.uniquePodcastSeriesSlug(r.Context(), slugify(s.Name), id)
	}

	if err := h.q.UpdatePodcastSeries(r.Context(), sqlc.UpdatePodcastSeriesParams{
		Name: s.Name, Slug: s.Slug, SortOrder: s.SortOrder, IsActive: s.IsActive, ID: id,
	}); err != nil {
		renderErr("Failed to save series: " + err.Error())
		return
	}
	h.audit(r, "update", "podcast_series", &id, "Updated podcast series "+s.Name)
	h.flash(w, "Series saved.")
	http.Redirect(w, r, "/admin/podcast-series", http.StatusSeeOther)
}

// PodcastSeriesDelete removes a series. The podcasts.series_id FK is RESTRICT, so a
// series still assigned to a podcast can't be deleted - that surfaces as a friendly
// flash telling the admin to reassign first, rather than a 500.
func (h *Handler) PodcastSeriesDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeletePodcastSeries(r.Context(), id); err != nil {
		if isForeignKeyErr(err) {
			h.flash(w, "That series is still assigned to one or more podcasts. Reassign them to another series first.")
			http.Redirect(w, r, "/admin/podcast-series", http.StatusSeeOther)
			return
		}
		http.Error(w, "failed to delete series", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "podcast_series", &id, "Deleted podcast series")
	h.flash(w, "Series deleted.")
	http.Redirect(w, r, "/admin/podcast-series", http.StatusSeeOther)
}

// podcastSeriesFromForm reads the series fields from the request. The slug is derived
// from the name by the caller, not here.
func podcastSeriesFromForm(r *http.Request) (s sqlc.PodcastSeries, formErr string) {
	if err := r.ParseForm(); err != nil {
		return s, "Invalid form submission."
	}
	s.Name = strings.TrimSpace(r.FormValue("name"))
	if n, err := strconv.Atoi(r.FormValue("sort_order")); err == nil {
		s.SortOrder = int32(n)
	}
	s.IsActive = r.FormValue("is_active") == "on"
	if s.Name == "" {
		return s, "Name is required."
	}
	return s, ""
}

// uniquePodcastSeriesSlug returns base, or base-2, base-3, ... if a different series
// already holds it. excludeID is the series being updated (0 on create).
func (h *Handler) uniquePodcastSeriesSlug(ctx context.Context, base string, excludeID uint64) string {
	if base == "" {
		base = "series"
	}
	candidate := base
	for n := 2; ; n++ {
		existing, err := h.q.GetPodcastSeriesBySlug(ctx, candidate)
		if err != nil || existing.ID == excludeID {
			return candidate
		}
		candidate = base + "-" + strconv.Itoa(n)
	}
}

// isForeignKeyErr reports whether err is a MySQL foreign-key constraint violation
// (a child row still references the row being deleted).
func isForeignKeyErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "foreign key constraint fails") ||
		strings.Contains(msg, "Cannot delete or update a parent row")
}
