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
		err := w.q.UpsertNewsItem(ctx, sqlc.UpsertNewsItemParams{
			Source:      sqlc.NewsItemsSource(src.Key()),
			ExternalID:  sql.NullString{String: item.ExternalID, Valid: item.ExternalID != ""},
			Title:       item.Title,
			Excerpt:     sql.NullString{String: item.Excerpt, Valid: item.Excerpt != ""},
			Url:         sql.NullString{String: item.URL, Valid: item.URL != ""},
			ImageUrl:    sql.NullString{String: item.ImageURL, Valid: item.ImageURL != ""},
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
