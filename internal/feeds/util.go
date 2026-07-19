package feeds

import (
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

var (
	stripTags   = bluemonday.StrictPolicy()
	imgSrcRegex = regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
)

// firstImageSrc returns the src of the first <img> tag found in HTML, if any.
func firstImageSrc(rawHTML string) string {
	m := imgSrcRegex.FindStringSubmatch(rawHTML)
	if m == nil {
		return ""
	}
	return html.UnescapeString(m[1])
}

// plainText strips HTML tags and collapses whitespace, decoding entities.
func plainText(rawHTML string) string {
	text := stripTags.Sanitize(rawHTML)
	text = html.UnescapeString(text)
	return strings.Join(strings.Fields(text), " ")
}

// truncateText trims s to at most maxRunes runes, breaking on a rune boundary and
// appending an ellipsis when truncated.
func truncateText(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxRunes])) + "…"
}
