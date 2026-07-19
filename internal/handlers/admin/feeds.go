package admin

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// feedSourceUpdater is the subset of *feeds.Worker the admin handlers need. Declared
// here (rather than importing internal/feeds) to keep this package's dependency graph
// one-directional; *feeds.Worker satisfies it.
type feedSourceUpdater interface {
	RunOnce(ctx context.Context)
}

type feedSourcesListData struct {
	Base    baseData
	Sources []sqlc.FeedSource
}

// FeedSourcesList renders every feed source's config + last-run status.
func (h *Handler) FeedSourcesList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	sources, err := h.q.ListFeedSources(r.Context())
	if err != nil {
		http.Error(w, "gagal memuat sumber feed", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/feed_sources_list", feedSourcesListData{
		Base:    h.base(r, "Sumber Feed", "feed-sources"),
		Sources: sources,
	})
}

// FeedSourceUpdate saves the enabled flag + endpoint override for one source.
func (h *Handler) FeedSourceUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	source := chi.URLParam(r, "source")
	switch source {
	case "youtube", "klikpositif", "katasumbar":
	default:
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	err := h.q.UpdateFeedSourceConfig(r.Context(), sqlc.UpdateFeedSourceConfigParams{
		IsEnabled: r.FormValue("is_enabled") == "on",
		Endpoint:  toNullString(r.FormValue("endpoint")),
		Source:    sqlc.FeedSourcesSource(source),
	})
	if err != nil {
		http.Error(w, "gagal menyimpan sumber feed", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "feed_source", nil, "Mengubah sumber feed "+source)
	http.Redirect(w, r, "/admin/feed-sources", http.StatusSeeOther)
}

// FeedSourcesRefresh triggers an immediate fetch pass across every source.
func (h *Handler) FeedSourcesRefresh(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	if h.worker != nil {
		h.worker.RunOnce(r.Context())
	}
	h.audit(r, "refresh", "feed_source", nil, "Memicu refresh semua sumber feed")
	http.Redirect(w, r, "/admin/feed-sources", http.StatusSeeOther)
}

type newsfeedListData struct {
	Base         baseData
	Items        []sqlc.NewsItem
	SourceFilter string
}

// NewsfeedList renders the read-only aggregated newsfeed (YouTube/KlikPositif/
// KataSumbar), optionally filtered by ?source=.
func (h *Handler) NewsfeedList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	items, err := h.q.ListAggregatedNews(r.Context())
	if err != nil {
		http.Error(w, "gagal memuat newsfeed", http.StatusInternalServerError)
		return
	}
	filter := r.URL.Query().Get("source")
	if filter != "" {
		filtered := items[:0]
		for _, it := range items {
			if string(it.Source) == filter {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	h.r.Page(w, http.StatusOK, "admin/newsfeed_list", newsfeedListData{
		Base:         h.base(r, "Newsfeed", "newsfeed"),
		Items:        items,
		SourceFilter: filter,
	})
}

// NewsfeedTogglePublish flips is_published on one aggregated item.
func (h *Handler) NewsfeedTogglePublish(w http.ResponseWriter, r *http.Request) {
	h.newsfeedToggle(w, r, func(ctx context.Context, id uint64, item sqlc.NewsItem) error {
		return h.q.SetNewsItemPublished(ctx, sqlc.SetNewsItemPublishedParams{IsPublished: !item.IsPublished, ID: id})
	})
}

// NewsfeedToggleFeature flips is_featured on one aggregated item.
func (h *Handler) NewsfeedToggleFeature(w http.ResponseWriter, r *http.Request) {
	h.newsfeedToggle(w, r, func(ctx context.Context, id uint64, item sqlc.NewsItem) error {
		return h.q.SetNewsItemFeatured(ctx, sqlc.SetNewsItemFeaturedParams{IsFeatured: !item.IsFeatured, ID: id})
	})
}

func (h *Handler) newsfeedToggle(w http.ResponseWriter, r *http.Request, apply func(context.Context, uint64, sqlc.NewsItem) error) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	item, err := h.q.GetNewsItem(r.Context(), id)
	if err != nil || item.Source == sqlc.NewsItemsSourceHotRelease {
		http.NotFound(w, r)
		return
	}
	if err := apply(r.Context(), id, item); err != nil {
		http.Error(w, "gagal menyimpan perubahan", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "newsfeed_item", &id, "Mengubah status newsfeed item")
	http.Redirect(w, r, "/admin/newsfeed", http.StatusSeeOther)
}
