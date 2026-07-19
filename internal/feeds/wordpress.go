package feeds

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// WordPressSource parses a standard WordPress RSS 2.0 feed (KlikPositif, KataSumbar
// both expose one at /feed/) — no API key needed.
type WordPressSource struct {
	client      *http.Client
	key         string
	defaultFeed string
}

// NewWordPressSource builds a WordPress RSS source. key must match a
// feed_sources.source / news_items.source enum value. defaultFeedURL is used when no
// per-source endpoint override is configured.
func NewWordPressSource(client *http.Client, key, defaultFeedURL string) *WordPressSource {
	return &WordPressSource{client: client, key: key, defaultFeed: defaultFeedURL}
}

func (s *WordPressSource) Key() string { return s.key }

type wpRSS struct {
	Channel struct {
		Items []wpItem `xml:"item"`
	} `xml:"channel"`
}

type wpItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

// rfc822z matches WordPress's <pubDate> format, e.g. "Sat, 18 Jul 2026 09:05:32 +0000".
const rfc822z = time.RFC1123Z

func (s *WordPressSource) Fetch(ctx context.Context, endpoint string) ([]NewsItem, error) {
	feedURL := endpoint
	if feedURL == "" {
		feedURL = s.defaultFeed
	}
	if feedURL == "" {
		return nil, fmt.Errorf("%s: no feed url configured", s.key)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: fetch: %w", s.key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: unexpected status %d", s.key, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%s: read body: %w", s.key, err)
	}

	var feed wpRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("%s: parse rss: %w", s.key, err)
	}

	items := make([]NewsItem, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		published, err := time.Parse(rfc822z, strings.TrimSpace(it.PubDate))
		if err != nil {
			continue // skip items with an unparseable date rather than failing the whole fetch
		}
		externalID := it.GUID
		if externalID == "" {
			externalID = it.Link
		}
		items = append(items, NewsItem{
			ExternalID:  externalID,
			Title:       plainText(it.Title),
			Excerpt:     truncateText(excerptFromDescription(it.Description), 300),
			URL:         it.Link,
			ImageURL:    firstImageSrc(it.Description),
			PublishedAt: published,
		})
	}
	return items, nil
}

// excerptFromDescription strips markup and the "The post ... appeared first on ..."
// boilerplate WordPress appends to feed descriptions.
func excerptFromDescription(rawHTML string) string {
	text := plainText(rawHTML)
	if idx := strings.Index(text, "The post "); idx != -1 {
		text = text[:idx]
	}
	return strings.TrimSpace(text)
}
