package feeds

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// YouTubeSource pulls the channel's free, keyless Atom feed instead of calling the
// Data API v3 — avoids provisioning a Google Cloud project / API key entirely.
type YouTubeSource struct {
	client         *http.Client
	defaultChannel string
}

// NewYouTubeSource builds a YouTube RSS source. defaultChannel is used when no
// per-source endpoint override is configured (e.g. via the admin Feed Sources screen).
func NewYouTubeSource(client *http.Client, defaultChannel string) *YouTubeSource {
	return &YouTubeSource{client: client, defaultChannel: defaultChannel}
}

func (s *YouTubeSource) Key() string { return "youtube" }

type ytAtomFeed struct {
	Entries []ytAtomEntry `xml:"entry"`
}

type ytAtomEntry struct {
	ID        string    `xml:"id"`
	Title     string    `xml:"title"`
	Published time.Time `xml:"published"`
	Link      struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Group struct {
		Description string `xml:"description"`
		Thumbnail   struct {
			URL string `xml:"url,attr"`
		} `xml:"thumbnail"`
	} `xml:"group"`
}

// Fetch parses https://www.youtube.com/feeds/videos.xml?channel_id=... (the channel's
// most recent ~15 uploads).
func (s *YouTubeSource) Fetch(ctx context.Context, endpoint string) ([]NewsItem, error) {
	channelID := endpoint
	if channelID == "" {
		channelID = s.defaultChannel
	}
	if channelID == "" {
		return nil, fmt.Errorf("youtube: no channel id configured")
	}

	feedURL := "https://www.youtube.com/feeds/videos.xml?channel_id=" + url.QueryEscape(channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("youtube: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("youtube: read body: %w", err)
	}

	var feed ytAtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("youtube: parse atom: %w", err)
	}

	items := make([]NewsItem, 0, len(feed.Entries))
	for _, e := range feed.Entries {
		items = append(items, NewsItem{
			ExternalID:  strings.TrimPrefix(e.ID, "yt:video:"),
			Title:       e.Title,
			Excerpt:     truncateText(e.Group.Description, 300),
			URL:         e.Link.Href,
			ImageURL:    e.Group.Thumbnail.URL,
			PublishedAt: e.Published,
		})
	}
	return items, nil
}
