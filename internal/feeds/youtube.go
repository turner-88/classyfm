package feeds

import (
	"context"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// youtubePlaceholderWidth is the width of YouTube's fixed gray placeholder
// image, served (with HTTP 200) in place of maxresdefault.jpg/sddefault.jpg
// when a video has no thumbnail at that size.
const youtubePlaceholderWidth = 120

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
		videoID := strings.TrimPrefix(e.ID, "yt:video:")
		items = append(items, NewsItem{
			ExternalID:  videoID,
			Title:       e.Title,
			Excerpt:     truncateText(e.Group.Description, 300),
			URL:         e.Link.Href,
			ImageURL:    BestYouTubeThumbnail(ctx, s.client, videoID, e.Group.Thumbnail.URL),
			ThumbURL:    e.Group.Thumbnail.URL,
			PublishedAt: e.Published,
		})
	}
	return items, nil
}

// BestYouTubeThumbnail tries YouTube's higher-resolution thumbnail sizes,
// which live at a predictable URL per video ID but aren't listed in the Atom
// feed itself (it only ever gives hqdefault.jpg, 480x360). Ordered by how much
// usable image the hero section actually gets: maxresdefault and hq720 are both
// 1280x720 widescreen, while sddefault is 640x480 4:3 - cropped to the hero's
// aspect that leaves only ~640x360, so hq720 must be tried first even though
// it's the same nominal height. Falls back to fallback (the feed's own
// thumbnail) when none of the larger sizes exist.
func BestYouTubeThumbnail(ctx context.Context, client *http.Client, videoID, fallback string) string {
	for _, name := range []string{"maxresdefault.jpg", "hq720.jpg", "sddefault.jpg"} {
		u := fmt.Sprintf("https://i.ytimg.com/vi/%s/%s", videoID, name)
		if youtubeThumbnailIsReal(ctx, client, u) {
			return u
		}
	}
	return fallback
}

// youtubeThumbnailIsReal reports whether url is a genuine thumbnail rather
// than YouTube's fixed 120x90 gray placeholder, which it serves with HTTP 200
// for any size that doesn't exist for a given video.
func youtubeThumbnailIsReal(ctx context.Context, client *http.Client, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	cfg, _, err := image.DecodeConfig(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false
	}
	return cfg.Width > youtubePlaceholderWidth
}
