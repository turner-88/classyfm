// Package sanitize strips HTML markup from admin-authored Hot Release content
// before it is stored, so it renders safely as plain text.
package sanitize

import (
	"html"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var (
	stripPolicy = bluemonday.StrictPolicy()
	paraBreak   = regexp.MustCompile(`(?i)</p\s*>|</div\s*>|</li\s*>|</h[1-6]\s*>|</blockquote\s*>|</pre\s*>`)
	lineBreak   = regexp.MustCompile(`(?i)<br\s*/?>`)
	blankLines  = regexp.MustCompile(`\n{3,}`)
)

// PlainText converts HTML input into plain text: paragraph/block-level closing
// tags become a blank line (two newlines) so paragraphs stay visually separated,
// <br> becomes a single newline, then all remaining tags and attributes are
// stripped, entities are decoded to literal characters, and any run of 3+
// newlines (from adjacent block tags or pre-existing whitespace in the source)
// is collapsed back down to exactly one blank line.
func PlainText(s string) string {
	s = paraBreak.ReplaceAllString(s, "\n\n")
	s = lineBreak.ReplaceAllString(s, "\n")
	s = stripPolicy.Sanitize(s)
	s = html.UnescapeString(s)
	s = blankLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
