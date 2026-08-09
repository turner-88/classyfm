package admin

import (
	"context"
	"net/http"
	"net/url"

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
		http.Error(w, "failed to load feed sources", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/feed_sources_list", feedSourcesListData{
		Base:    h.base(r, "Feed Sources", "feed-sources"),
		Sources: sources,
	})
}

// feedSourceKeys are the sources the worker knows how to fetch. Fixed rows, so this
// is an allowlist rather than a lookup.
var feedSourceKeys = []string{"youtube", "klikpositif", "katasumbar"}

// FeedSourcesUpdate saves the enabled flag + endpoint override for every source in
// one submission; each source's fields are suffixed with its key.
func (h *Handler) FeedSourcesUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	_ = r.ParseForm()
	for _, source := range feedSourceKeys {
		if err := h.q.UpdateFeedSourceConfig(r.Context(), sqlc.UpdateFeedSourceConfigParams{
			IsEnabled: r.FormValue("enabled_"+source) == "on",
			Endpoint:  toNullString(r.FormValue("endpoint_" + source)),
			Source:    sqlc.FeedSourcesSource(source),
		}); err != nil {
			http.Error(w, "failed to save feed source", http.StatusInternalServerError)
			return
		}
	}
	h.audit(r, "update", "feed_source", nil, "Updated feed sources")
	h.flash(w, "Feed sources saved.")
	http.Redirect(w, r, "/admin/feed-sources", http.StatusSeeOther)
}

// FeedSourcesRefresh triggers an immediate fetch pass across every source. Both
// this page and the dashboard offer the button, so the caller says where to land
// afterwards - matched against an allowlist rather than trusted, since a redirect
// target taken from a form is otherwise an open redirect.
func (h *Handler) FeedSourcesRefresh(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	if h.worker != nil {
		h.worker.RunOnce(r.Context())
	}
	h.audit(r, "refresh", "feed_source", nil, "Triggered refresh of all feed sources")
	dest := "/admin/feed-sources"
	switch r.FormValue("return") {
	case "/admin":
		dest = "/admin"
		h.flash(w, "Feed sources refreshed.")
	case "/admin/newsfeed":
		dest = "/admin/newsfeed"
		h.flash(w, "Feed sources refreshed.")
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

type newsfeedListData struct {
	Base         baseData
	Items        []sqlc.NewsItem
	SourceFilter string
	Pagination   pagination
}

// NewsfeedList renders the read-only aggregated newsfeed (YouTube/KlikPositif/
// KataSumbar), optionally filtered by ?source=.
func (h *Handler) NewsfeedList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	filter := r.URL.Query().Get("source")
	switch filter {
	case "youtube", "klikpositif", "katasumbar":
	default:
		filter = ""
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "published_at", "desc", "title", "source", "published_at")

	total, err := h.q.CountAggregatedNews(r.Context(), sqlc.CountAggregatedNewsParams{
		Source: sqlc.NewsItemsSource(filter), Search: pattern,
	})
	if err != nil {
		http.Error(w, "failed to load newsfeed", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/newsfeed", url.Values{"source": {filter}, "q": {search}, "sort": {sort}, "dir": {dir}})
	items, err := h.q.ListAggregatedNews(r.Context(), sqlc.ListAggregatedNewsParams{
		Source: sqlc.NewsItemsSource(filter), Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load newsfeed", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/newsfeed_list", newsfeedListData{
		Base:         h.base(r, "Newsfeed", "newsfeed"),
		Items:        items,
		SourceFilter: filter,
		Pagination:   pg,
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
		http.Error(w, "failed to save changes", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "newsfeed_item", &id, "Updated newsfeed item status")
	http.Redirect(w, r, "/admin/newsfeed", http.StatusSeeOther)
}
