// Package feeds aggregates external newsfeeds (YouTube, KlikPositif, KataSumbar) into
// the shared news_items table via RSS/Atom — no external API keys required.
package feeds

import (
	"context"
	"time"
)

// NewsItem is the normalized shape a Source produces for upserting into news_items.
type NewsItem struct {
	ExternalID  string
	Title       string
	Excerpt     string
	URL         string
	ImageURL    string
	PublishedAt time.Time
}

// Source fetches recent items from one external feed. endpoint overrides the
// source's built-in default (channel id / feed URL) when non-empty, so it can be
// edited from the admin Feed Sources screen.
type Source interface {
	// Key identifies the source; it must match a feed_sources.source /
	// news_items.source enum value ("youtube", "klikpositif", "katasumbar").
	Key() string
	Fetch(ctx context.Context, endpoint string) ([]NewsItem, error)
}
