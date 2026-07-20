package feeds

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// wpResizeSuffix matches WordPress's auto-generated intermediate-size suffix,
// e.g. "-300x200" in "IMG-20260720-WA0049-300x200.webp". WordPress always keeps
// the original file alongside every generated size, so stripping this recovers
// the full-resolution upload.
var wpResizeSuffix = regexp.MustCompile(`-\d+x\d+(\.\w+)$`)

// WPOriginalURL returns the likely full-resolution original for a WordPress
// media URL that carries an auto-generated size suffix, and whether a suffix
// was found at all (callers should only trust the result when ok is true).
func WPOriginalURL(src string) (original string, ok bool) {
	loc := wpResizeSuffix.FindStringSubmatchIndex(src)
	if loc == nil {
		return "", false
	}
	return src[:loc[0]] + src[loc[2]:loc[3]], true
}

// ProbeImageExists reports whether url resolves to a real image via HEAD,
// falling back to a GET (aborted after headers) for servers that reject HEAD.
func ProbeImageExists(ctx context.Context, client *http.Client, url string) bool {
	if ok, checked := probeImageRequest(ctx, client, http.MethodHead, url); checked {
		return ok
	}
	ok, _ := probeImageRequest(ctx, client, http.MethodGet, url)
	return ok
}

func probeImageRequest(ctx context.Context, client *http.Client, method, url string) (ok bool, checked bool) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return false, false
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		return false, false
	}
	return resp.StatusCode == http.StatusOK && strings.HasPrefix(resp.Header.Get("Content-Type"), "image/"), true
}

// FetchOGImage fetches pageURL and returns its <meta property="og:image">
// content, or "" if the page has none.
func FetchOGImage(ctx context.Context, client *http.Client, pageURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	content, _ := doc.Find(`meta[property="og:image"]`).First().Attr("content")
	return strings.TrimSpace(content), nil
}

// ytThumbnailURL matches YouTube's predictable per-video thumbnail URLs, e.g.
// "https://i2.ytimg.com/vi/<id>/hqdefault.jpg" or the maxres/sddefault
// variants BestYouTubeThumbnail constructs itself (no host number prefix).
var ytThumbnailURL = regexp.MustCompile(`^https://i\d*\.ytimg\.com/vi/([^/]+)/(\w+)\.jpg$`)

// ytThumbRank orders YouTube's predefined thumbnail sizes from lowest to
// highest resolution.
var ytThumbRank = map[string]int{
	"default":       0,
	"mqdefault":     1,
	"hqdefault":     2,
	"sddefault":     3,
	"maxresdefault": 4,
}

// PreferImage decides which image URL to keep when a feed refresh re-resolves
// an item that's already in the database: existing is what's currently
// stored, candidate is what this fetch just resolved. Every feed source
// re-derives its image from scratch on every refresh (there's no "only
// process new items" step), and resolution involves a live network probe
// (WPOriginalURL/ProbeImageExists, YouTube's maxres/sddefault check) that can
// fail transiently - without this check, a single flaky request during a
// routine refresh would silently downgrade an already-upgraded image. This
// makes refreshes upgrade-or-keep, never downgrade.
func PreferImage(existing, candidate string) string {
	if candidate == "" {
		return existing
	}
	if existing == "" || existing == candidate {
		return candidate
	}

	// WordPress: candidate is a resized thumbnail of the exact original
	// that's already stored in full resolution.
	if base, ok := WPOriginalURL(candidate); ok && base == existing {
		return existing
	}

	// YouTube: candidate is a lower predefined size of the same video whose
	// thumbnail is already stored at a higher one.
	me, mc := ytThumbnailURL.FindStringSubmatch(existing), ytThumbnailURL.FindStringSubmatch(candidate)
	if me != nil && mc != nil && me[1] == mc[1] && ytThumbRank[me[2]] > ytThumbRank[mc[2]] {
		return existing
	}

	return candidate
}
