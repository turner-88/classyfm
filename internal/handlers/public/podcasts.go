package public

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// podcastListItem is the card view-model for the podcast list, normalized so the "all"
// and "by series" queries (which sqlc types as distinct row structs with identical
// fields) render through one template path.
type podcastListItem struct {
	Slug            string
	Title           string
	Description     string
	ThumbUrl        sql.NullString
	SeriesName      string
	BroadcasterName sql.NullString
}

// Podcast renders the public podcast list at /podcast, newest first, optionally
// filtered to one series via ?series=<slug>.
func (h *Handler) Podcast(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := int32((page - 1) * newsPageSize)

	var series []sqlc.PodcastSeries
	var items []podcastListItem
	var total int64
	selectedSeries := ""
	if h.q != nil {
		series, _ = h.q.ListPodcastSeries(r.Context())
		// An unknown ?series= slug falls back to the unfiltered list rather than 404,
		// mirroring the news source-filter behavior.
		if sel := r.URL.Query().Get("series"); sel != "" {
			if _, err := h.q.GetPodcastSeriesBySlug(r.Context(), sel); err == nil {
				selectedSeries = sel
			}
		}
		if selectedSeries != "" {
			rows, _ := h.q.ListPublishedPodcastsBySeriesSlug(r.Context(), sqlc.ListPublishedPodcastsBySeriesSlugParams{
				Slug: selectedSeries, Limit: newsPageSize, Offset: offset,
			})
			for _, p := range rows {
				items = append(items, podcastListItem{p.Slug, p.Title, p.Description, p.ThumbUrl, p.SeriesName, p.BroadcasterName})
			}
			total, _ = h.q.CountPublishedPodcastsBySeriesSlug(r.Context(), selectedSeries)
		} else {
			rows, _ := h.q.ListPublishedPodcasts(r.Context(), sqlc.ListPublishedPodcastsParams{
				Limit: newsPageSize, Offset: offset,
			})
			for _, p := range rows {
				items = append(items, podcastListItem{p.Slug, p.Title, p.Description, p.ThumbUrl, p.SeriesName, p.BroadcasterName})
			}
			total, _ = h.q.CountPublishedPodcasts(r.Context())
		}
	}
	totalPages := int((total + newsPageSize - 1) / newsPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	h.r.Page(w, http.StatusOK, "public/podcast", struct {
		Base           baseData
		Series         []sqlc.PodcastSeries
		SelectedSeries string
		Podcasts       []podcastListItem
		Page           int
		TotalPages     int
	}{h.base(r, "Podcast", "podcast", "Podcasts from "+h.station+"."), series, selectedSeries, items, page, totalPages})
}

// PodcastDetail renders a single podcast at /podcast/{slug}.
func (h *Handler) PodcastDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		h.NotFound(w, r)
		return
	}
	slug := chi.URLParam(r, "slug")
	item, err := h.q.GetPublishedPodcastBySlug(r.Context(), slug)
	if err != nil {
		h.NotFound(w, r)
		return
	}
	broadcasters, _ := h.q.ListPodcastBroadcasters(r.Context(), item.ID)
	// Best-effort: the series name heads the page as its eyebrow. A missing series
	// (should not happen given the NOT NULL FK) just leaves it blank.
	seriesName := ""
	if s, err := h.q.GetPodcastSeries(r.Context(), item.SeriesID); err == nil {
		seriesName = s.Name
	}

	base := h.base(r, item.Title, "podcast", item.Description)
	base.OGImage = item.ThumbUrl.String
	h.r.Page(w, http.StatusOK, "public/podcast_detail", struct {
		Base         baseData
		Item         sqlc.Podcast
		SeriesName   string
		Broadcasters []sqlc.Broadcaster
	}{base, item, seriesName, broadcasters})
}
