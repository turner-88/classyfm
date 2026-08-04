// Package radio provides the stream configuration and a cached proxy for
// now-playing metadata, sourced directly from the Shoutcast server's own native
// endpoints. Caching means all site visitors share a single upstream fetch rather
// than each browser hammering the Shoutcast server.
package radio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// coverCacheTTL bounds how long a resolved cover art URL is trusted for a given
// "artist - song" key, so a steady song across repeated 15s polls doesn't re-hit
// iTunes's undocumented rate limit.
const coverCacheTTL = time.Hour

// NowPlaying is the now-playing snapshot served to the frontend player and the
// /live page.
type NowPlaying struct {
	Artist   string `json:"artist"`
	Song     string `json:"song"`
	HasSong  bool   `json:"has_song"`  // false = no current metadata ("Empty Title")
	CoverURL string `json:"cover_url"` // best-effort iTunes artwork; "" if none
	Live     bool   `json:"live"`
	// Listeners is the Shoutcast server's current audience. Deliberately not in
	// the JSON: /api/nowplaying is public, and the count is only surfaced in the
	// admin panel. Publishing it later is a one-word change to this tag.
	Listeners int `json:"-"`
}

// Service caches now-playing metadata fetched from the Shoutcast server (avoids
// CORS and shields the Shoutcast server from load).
type Service struct {
	streamURL     string
	shoutcastBase string
	ttl           time.Duration
	client        *http.Client

	mu       sync.RWMutex
	cached   NowPlaying
	fetched  time.Time
	hasValue bool

	// Separate cache for iTunes cover art lookups, keyed by "artist - song" so a
	// held song doesn't trigger a fresh iTunes request on every 15s poll.
	coverMu      sync.Mutex
	coverKey     string
	coverURL     string
	coverFetched time.Time
}

// NewService builds a now-playing service. shoutcastBase is the Shoutcast server's
// base URL, e.g. "https://c4.siar.us:10340" (no trailing slash required).
func NewService(streamURL, shoutcastBase string) *Service {
	return &Service{
		streamURL:     streamURL,
		shoutcastBase: strings.TrimRight(shoutcastBase, "/"),
		ttl:           12 * time.Second,
		client:        &http.Client{Timeout: 6 * time.Second},
	}
}

// StreamURL returns the configured audio stream URL.
func (s *Service) StreamURL() string { return s.streamURL }

// Current returns the cached now-playing value, refreshing if older than the TTL.
func (s *Service) Current(ctx context.Context) NowPlaying {
	s.mu.RLock()
	fresh := s.hasValue && time.Since(s.fetched) < s.ttl
	val := s.cached
	s.mu.RUnlock()
	if fresh {
		return val
	}
	return s.refresh(ctx)
}

// refresh fetches played.html and stats?json=1 independently - one failing must not
// blank the other, so each field is only overwritten on a successful fetch, starting
// from the last cached value.
func (s *Service) refresh(ctx context.Context) NowPlaying {
	s.mu.RLock()
	np := s.cached
	s.mu.RUnlock()

	if artist, song, hasSong, ok := s.fetchPlayedHTML(ctx); ok {
		np.Artist, np.Song, np.HasSong = artist, song, hasSong
	}
	if listeners, live, ok := s.fetchStats(ctx); ok {
		np.Live, np.Listeners = live, listeners
	}
	if np.HasSong {
		np.CoverURL = s.coverArt(ctx, np.Artist, np.Song)
	} else {
		np.CoverURL = ""
	}

	s.mu.Lock()
	s.cached, s.fetched, s.hasValue = np, time.Now(), true
	s.mu.Unlock()
	return np
}

// fetchPlayedHTML scrapes {shoutcastBase}/played.html's "Stream History" table for
// the current song. The current-song row is marked with a 3rd <td> ("Current
// Song"); its 2nd cell holds "Artist - Title", or literally "Empty Title" when no
// metadata is available. ok reports whether the fetch/parse itself succeeded (not
// whether a song was found - see hasSong for that).
func (s *Service) fetchPlayedHTML(ctx context.Context) (artist, song string, hasSong, ok bool) {
	body, err := s.get(ctx, s.shoutcastBase+"/played.html")
	if err != nil {
		slog.Warn("played.html fetch failed", "err", err)
		return "", "", false, false
	}
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		slog.Warn("played.html parse failed", "err", err)
		return "", "", false, false
	}

	var title string
	found := false
	doc.Find("tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		cells := tr.Find("td")
		if cells.Length() < 3 {
			return true // keep looking
		}
		title = strings.TrimSpace(cells.Eq(1).Text())
		found = true
		return false // stop
	})
	if !found {
		return "", "", false, false
	}
	if title == "" || title == "Empty Title" {
		return "", "", false, true
	}
	if before, after, hasSep := strings.Cut(title, " - "); hasSep {
		return before, after, true, true
	}
	return "", title, true, true
}

// shoutcastStats is the subset of {shoutcastBase}/stats?json=1 that's reliable.
// songtitle is deliberately not decoded here - verified empty across repeated
// checks against the live server, so played.html is the metadata source instead.
//
// peaklisteners is likewise not decoded: the server resets it on restart, so it
// answers "peak since this process started", not "peak today". The station's own
// daily peak is built from currentlisteners samples instead - see
// internal/listeners.
type shoutcastStats struct {
	StreamStatus     int `json:"streamstatus"`
	CurrentListeners int `json:"currentlisteners"`
}

// StreamStats fetches the Shoutcast server's stats directly, bypassing the
// now-playing cache. The listener sampler uses this rather than Current: it must
// not drag played.html and the iTunes cover lookup along on every poll, and a
// value up to the cache TTL old would quietly duplicate samples.
func (s *Service) StreamStats(ctx context.Context) (listeners int, live, ok bool) {
	return s.fetchStats(ctx)
}

func (s *Service) fetchStats(ctx context.Context) (listeners int, live, ok bool) {
	body, err := s.get(ctx, s.shoutcastBase+"/stats?json=1")
	if err != nil {
		slog.Warn("stats fetch failed", "err", err)
		return 0, false, false
	}
	defer body.Close()

	var st shoutcastStats
	if err := json.NewDecoder(io.LimitReader(body, 1<<16)).Decode(&st); err != nil {
		slog.Warn("stats decode failed", "err", err)
		return 0, false, false
	}
	return st.CurrentListeners, st.StreamStatus == 1, true
}

type itunesSearchResponse struct {
	Results []struct {
		ArtworkURL100 string `json:"artworkUrl100"`
	} `json:"results"`
}

// coverArt returns best-effort cover art for the given artist/song from the free,
// keyless iTunes Search API. It only enriches cover art - it never overrides the
// Shoutcast-sourced artist/song text, since local/Indonesian content won't reliably
// match iTunes's catalog. Callers should skip this entirely when there's no current
// song (hasSong == false).
func (s *Service) coverArt(ctx context.Context, artist, song string) string {
	key := artist + " - " + song

	s.coverMu.Lock()
	if s.coverKey == key && time.Since(s.coverFetched) < coverCacheTTL {
		cached := s.coverURL
		s.coverMu.Unlock()
		return cached
	}
	s.coverMu.Unlock()

	endpoint := "https://itunes.apple.com/search?term=" + url.QueryEscape(key) + "&media=music&entity=song&limit=1"
	art := ""
	if body, err := s.get(ctx, endpoint); err != nil {
		slog.Warn("itunes cover art fetch failed", "err", err)
	} else {
		var resp itunesSearchResponse
		if decErr := json.NewDecoder(io.LimitReader(body, 1<<16)).Decode(&resp); decErr == nil && len(resp.Results) > 0 {
			art = strings.Replace(resp.Results[0].ArtworkURL100, "100x100bb", "600x600bb", 1)
		}
		body.Close()
	}

	s.coverMu.Lock()
	s.coverKey, s.coverURL, s.coverFetched = key, art, time.Now()
	s.coverMu.Unlock()
	return art
}

// get issues a GET request with the package's conventional User-Agent and returns
// the response body for the caller to read and close, erroring on non-200 status.
func (s *Service) get(ctx context.Context, u string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, u)
	}
	return resp.Body, nil
}
