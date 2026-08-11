// Package spotify resolves display data from a public Spotify URL: the artwork
// thumbnail (via Spotify's keyless oEmbed endpoint), the episode description (scraped
// from the share page's og:description meta tag) and the /embed player URL used in an
// iframe. All work off an open.spotify.com share link (episode, show, track, album,
// playlist) and need no API credentials.
package spotify

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// EmbedURL rewrites a public Spotify link into its iframe player URL:
// https://open.spotify.com/episode/<id>?si=... -> https://open.spotify.com/embed/episode/<id>
// It returns "" for anything that is not an open.spotify.com URL, so a template can
// decide whether to render the player at all. A link already under /embed is returned
// unchanged (minus its query string).
func EmbedURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host != "open.spotify.com" {
		return ""
	}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	// Drop a locale prefix such as /intl-id/episode/<id>.
	if len(segs) > 0 && strings.HasPrefix(segs[0], "intl-") {
		segs = segs[1:]
	}
	if len(segs) < 2 || segs[0] == "" || segs[1] == "" {
		return ""
	}
	if segs[0] == "embed" {
		return "https://open.spotify.com/" + strings.Join(segs, "/")
	}
	return "https://open.spotify.com/embed/" + strings.Join(segs, "/")
}

// FetchThumbnail returns the artwork URL for a Spotify link via its oEmbed endpoint,
// or "" (with an error) when the link is invalid or the fetch fails. It is best-effort
// by design: callers save the row regardless and simply store no thumbnail on failure.
func FetchThumbnail(ctx context.Context, client *http.Client, spotifyURL string) (string, error) {
	spotifyURL = strings.TrimSpace(spotifyURL)
	if spotifyURL == "" {
		return "", nil
	}
	endpoint := "https://open.spotify.com/oembed?url=" + url.QueryEscape(spotifyURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body struct {
		ThumbnailURL string `json:"thumbnail_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return strings.TrimSpace(body.ThumbnailURL), nil
}

// metaDescription matches the content of the <meta name="description"> tag on a Spotify
// share page. Spotify emits it server-side (for search/link previews), which is why it
// is readable without JavaScript or API credentials. This tag - not og:description,
// which only carries "<show> · Episode" - holds the full episode blurb. Spotify uses
// double quotes around the content, so [^"]* safely spans apostrophes inside it.
var metaDescription = regexp.MustCompile(`(?is)<meta[^>]+name=["']description["'][^>]*content="([^"]*)"`)

// spotifyDescPrefix strips Spotify's boilerplate lead-in, e.g.
// "Listen to this episode from <show> on Spotify. " or "Listen to <show> on Spotify. ",
// leaving just the episode's own description text.
var spotifyDescPrefix = regexp.MustCompile(`(?is)^Listen to .*? on Spotify\.\s*`)

// FetchDescription returns the episode/show description for a Spotify link by scraping
// the <meta name="description"> tag from its share page, or "" (with an error) when the
// link is invalid or the fetch fails. Like FetchThumbnail it is best-effort by design:
// callers save the row regardless and simply store no description on failure.
func FetchDescription(ctx context.Context, client *http.Client, spotifyURL string) (string, error) {
	spotifyURL = strings.TrimSpace(spotifyURL)
	if spotifyURL == "" {
		return "", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spotifyURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// The meta tag lives in <head>, so cap the read rather than pull the whole page.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	m := metaDescription.FindSubmatch(body)
	if m == nil {
		return "", nil
	}
	desc := strings.TrimSpace(html.UnescapeString(string(m[1])))
	desc = spotifyDescPrefix.ReplaceAllString(desc, "")
	return strings.TrimSpace(desc), nil
}
