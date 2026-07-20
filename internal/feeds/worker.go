package feeds

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// Worker periodically fetches every configured Source and upserts results into
// news_items, isolating failures per source and recording status in feed_sources so
// one bad source never blocks the others or fails the whole run.
type Worker struct {
	q        *sqlc.Queries
	sources  []Source
	interval time.Duration
}

// NewWorker builds a feed aggregation worker.
func NewWorker(q *sqlc.Queries, interval time.Duration, sources ...Source) *Worker {
	return &Worker{q: q, sources: sources, interval: interval}
}

// Run fetches immediately, then again on every tick, until ctx is canceled.
func (w *Worker) Run(ctx context.Context) {
	w.RunOnce(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.RunOnce(ctx)
		}
	}
}

// RunOnce runs a single fetch pass over every source. Exported so the admin
// "Refresh now" action can trigger the same logic on demand.
func (w *Worker) RunOnce(ctx context.Context) {
	for _, src := range w.sources {
		w.runSource(ctx, src)
	}
}

func (w *Worker) runSource(ctx context.Context, src Source) {
	key := sqlc.FeedSourcesSource(src.Key())

	var endpoint string
	fs, err := w.q.GetFeedSource(ctx, key)
	switch {
	case err == nil:
		if !fs.IsEnabled {
			return
		}
		endpoint = fs.Endpoint.String
	case err == sql.ErrNoRows:
		// No config row yet; fetch with the source's built-in default endpoint.
	default:
		slog.Error("feed source lookup failed", "source", key, "err", err)
		return
	}

	items, fetchErr := src.Fetch(ctx, endpoint)
	status := "ok"
	if fetchErr != nil {
		status = fetchErr.Error()
		slog.Warn("feed fetch failed", "source", key, "err", fetchErr)
	}

	count := 0
	for _, item := range items {
		imageURL, thumbURL := w.resolveImagesForUpsert(ctx, key, item)

		err := w.q.UpsertNewsItem(ctx, sqlc.UpsertNewsItemParams{
			Source:      sqlc.NewsItemsSource(src.Key()),
			ExternalID:  sql.NullString{String: item.ExternalID, Valid: item.ExternalID != ""},
			Title:       item.Title,
			Excerpt:     sql.NullString{String: item.Excerpt, Valid: item.Excerpt != ""},
			Url:         sql.NullString{String: item.URL, Valid: item.URL != ""},
			ImageUrl:    sql.NullString{String: imageURL, Valid: imageURL != ""},
			ThumbUrl:    sql.NullString{String: thumbURL, Valid: thumbURL != ""},
			PublishedAt: item.PublishedAt,
		})
		if err != nil {
			slog.Error("upsert news item failed", "source", key, "external_id", item.ExternalID, "err", err)
			continue
		}
		count++
	}

	if err := w.q.UpdateFeedSourceStatus(ctx, sqlc.UpdateFeedSourceStatusParams{
		LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
		LastStatus:    sql.NullString{String: status, Valid: true},
		ItemCount:     int32(count),
		Source:        key,
	}); err != nil {
		slog.Error("update feed source status failed", "source", key, "err", err)
	}
}

// resolveImagesForUpsert decides the image_url/thumb_url to write for item,
// preferring whatever's already stored over a freshly re-resolved value
// that's actually worse (see PreferImage) - a source re-derives both images
// from scratch on every refresh, and that resolution involves a live network
// probe that can fail transiently, so this is what keeps a routine refresh
// from silently undoing an already-upgraded image.
func (w *Worker) resolveImagesForUpsert(ctx context.Context, source sqlc.FeedSourcesSource, item NewsItem) (imageURL, thumbURL string) {
	if item.ExternalID == "" {
		return item.ImageURL, item.ThumbURL
	}

	existing, err := w.q.GetNewsItemImages(ctx, sqlc.GetNewsItemImagesParams{
		Source:     sqlc.NewsItemsSource(source),
		ExternalID: sql.NullString{String: item.ExternalID, Valid: true},
	})
	switch {
	case err == nil:
		return PreferImage(existing.ImageUrl.String, item.ImageURL), PreferImage(existing.ThumbUrl.String, item.ThumbURL)
	case err == sql.ErrNoRows:
		return item.ImageURL, item.ThumbURL
	default:
		slog.Warn("lookup existing images failed", "source", source, "external_id", item.ExternalID, "err", err)
		return item.ImageURL, item.ThumbURL
	}
}
