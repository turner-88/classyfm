// Package markdown renders trusted admin-authored Markdown (currently the legal
// pages) into sanitized HTML for embedding in a template.
package markdown

import (
	"bytes"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// converter is safe for concurrent use, so build it once.
var converter = goldmark.New(goldmark.WithExtensions(extension.GFM))

// policy sanitizes the rendered HTML. The Markdown is admin-authored (trusted),
// but running it through bluemonday keeps the output to a known-good tag set and
// is cheap insurance. UGCPolicy allows the tags Markdown produces (headings,
// paragraphs, lists, emphasis, links); links are restricted to safe schemes.
var policy = bluemonday.UGCPolicy()

// ToHTML converts a Markdown source string to sanitized HTML. It is registered as
// the "markdown" template func; on any conversion error it returns empty output
// rather than leaking a partial/unsafe render.
func ToHTML(md string) template.HTML {
	var buf bytes.Buffer
	if err := converter.Convert([]byte(md), &buf); err != nil {
		return ""
	}
	return template.HTML(policy.SanitizeBytes(buf.Bytes()))
}
