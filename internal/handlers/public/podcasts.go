package public

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// Podcast renders the public podcast list at /podcast, newest first.
func (h *Handler) Podcast(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := int32((page - 1) * newsPageSize)

	var podcasts []sqlc.ListPublishedPodcastsRow
	var total int64
	if h.q != nil {
		podcasts, _ = h.q.ListPublishedPodcasts(r.Context(), sqlc.ListPublishedPodcastsParams{
			Limit: newsPageSize, Offset: offset,
		})
		total, _ = h.q.CountPublishedPodcasts(r.Context())
	}
	totalPages := int((total + newsPageSize - 1) / newsPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	h.r.Page(w, http.StatusOK, "public/podcast", struct {
		Base       baseData
		Podcasts   []sqlc.ListPublishedPodcastsRow
		Page       int
		TotalPages int
	}{h.base(r, "Podcast", "podcast", "Podcasts from "+h.station+"."), podcasts, page, totalPages})
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

	base := h.base(r, item.Title, "podcast", item.Description)
	base.OGImage = item.ThumbUrl.String
	h.r.Page(w, http.StatusOK, "public/podcast_detail", struct {
		Base         baseData
		Item         sqlc.Podcast
		Broadcasters []sqlc.Broadcaster
	}{base, item, broadcasters})
}
