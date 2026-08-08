// Package spotify resolves display data from a public Spotify URL: the artwork
// thumbnail (via Spotify's keyless oEmbed endpoint) and the /embed player URL used
// in an iframe. Both work off an open.spotify.com share link (episode, show, track,
// album, playlist) and need no API credentials.
package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
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
