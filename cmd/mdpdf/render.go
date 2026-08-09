package main

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"
	"unicode"

	chroma "github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// brandStyle colors code tokens with the ClassyFM palette (navy keys, green
// strings, red literals, grey-italic comments). Parent token types cascade to
// their children, so JSON's NameTag/LiteralStringDouble/etc. all pick these up.
var brandStyle = chroma.MustNewStyle("classyfm", chroma.StyleEntries{
	chroma.Background:      "#24283b",
	chroma.Text:            "#24283b",
	chroma.NameTag:         "#292e62",
	chroma.Name:            "#292e62",
	chroma.String:          "#0a7d54",
	chroma.Number:          "#e32229",
	chroma.KeywordConstant: "#e32229",
	chroma.Keyword:         "#e32229",
	chroma.Comment:         "italic #8a90a2",
	chroma.Punctuation:     "#6b7180",
	chroma.Error:           "#24283b",
})

// pageData feeds the HTML shell template. FontFaceCSS is typed template.CSS
// because it is injected inside a <style> element — html/template would replace
// a plain string (or template.HTML) there with its ZgotmplZ safe placeholder.
type pageData struct {
	Subtitle      string
	GeneratedDate string
	FontFaceCSS   template.CSS
	BodyHTML      template.HTML
}

// renderHTML converts Markdown source into the full self-contained HTML document
// (fonts embedded, brand CSS) that Chrome will print to PDF.
func renderHTML(md []byte, subtitle string) (string, error) {
	converter := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithCustomStyle(brandStyle),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(false),
					// Emit only the token spans; codeWrapper supplies the <pre><code>.
					chromahtml.PreventSurroundingPre(true),
				),
				highlighting.WithWrapperRenderer(codeWrapper),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)

	// A per-conversion context carries the GitHub-compatible id generator that
	// WithAutoHeadingID reads when assigning heading ids.
	pc := parser.NewContext(parser.WithIDs(newGithubIDs()))
	var body bytes.Buffer
	if err := converter.Convert(md, &body, parser.WithContext(pc)); err != nil {
		return "", fmt.Errorf("markdown: %w", err)
	}
	// The document's first heading sits right under the cover block — let it stay
	// on the first page instead of triggering the h2 page-break rule.
	bodyHTML := strings.Replace(body.String(), "<h2", `<h2 class="no-break-before"`, 1)
	bodyHTML = badgeHeadings(bodyHTML)
	bodyHTML = styleEndpointTables(bodyHTML)
	bodyHTML = styleMethods(bodyHTML)

	fontCSS, err := buildFontFaceCSS()
	if err != nil {
		return "", err
	}

	tmpl, err := template.ParseFS(templateFS, "assets/template.html.tmpl")
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, pageData{
		Subtitle:      subtitle,
		GeneratedDate: time.Now().Format("2 January 2006"),
		FontFaceCSS:   template.CSS(fontCSS),
		BodyHTML:      template.HTML(bodyHTML),
	})
	if err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return out.String(), nil
}

// headingAnno matches an endpoint heading's " — <auth> (rate-limited N/min)"
// annotation so it can be re-rendered as badges. Group 2 is the auth phrase
// (anything up to an optional "(" — future-proof against new wording); group 3
// is the optional rate limit. Anchoring on </code>…</h3> keeps the rewrite
// scoped to endpoint headings (the annotation always follows the inline-code
// endpoint path) and leaves prose em dashes alone.
var headingAnno = regexp.MustCompile(
	`(</code>)\s*—\s*([^(<]+?)\s*(?:\(rate-limited ([0-9]+/min)\))?\s*(</h3>)`)

// badgeHeadings rewrites the auth/rate-limit annotation trailing an endpoint
// heading into pill badges. It runs on the rendered HTML, after goldmark has
// already assigned heading ids from the full text, so in-document cross-links
// to those ids keep resolving.
func badgeHeadings(html string) string {
	return headingAnno.ReplaceAllStringFunc(html, func(m string) string {
		s := headingAnno.FindStringSubmatch(m)
		auth := s[2]
		cls := "badge badge-open"
		if strings.Contains(auth, "Bearer") {
			cls = "badge badge-bearer"
		}
		b := s[1] + ` <span class="` + cls + `">` + auth + `</span>`
		if s[3] != "" {
			b += ` <span class="badge badge-rate">rate-limited ` + s[3] + `</span>`
		}
		return b + s[4]
	})
}

// endpointTable matches a whole rendered <table>…</table> block so each can be
// inspected in isolation; only those whose header carries both a Method and an
// Auth column (the endpoint-index tables) get their cells restyled.
var endpointTable = regexp.MustCompile(`(?s)<table>.*?</table>`)

// authCell matches an Auth column cell: "open" or "Bearer", optionally trailed by
// a "(N/min)" rate note.
var authCell = regexp.MustCompile(`<td>(open|Bearer)\s*(?:\((\d+/min)\))?</td>`)

// styleEndpointTables restyles the Auth column of the endpoint-index tables into
// pill badges, reusing the auth/rate styles used by badgeHeadings. It runs on the
// rendered HTML and leaves every other table alone. The Method column is handled
// document-wide by styleMethods, not here.
func styleEndpointTables(html string) string {
	return endpointTable.ReplaceAllStringFunc(html, func(tbl string) string {
		if !strings.Contains(tbl, ">Method<") || !strings.Contains(tbl, ">Auth<") {
			return tbl
		}
		return authCell.ReplaceAllStringFunc(tbl, func(cell string) string {
			s := authCell.FindStringSubmatch(cell)
			cls := "badge badge-open"
			if s[1] == "Bearer" {
				cls = "badge badge-bearer"
			}
			out := `<td><span class="` + cls + `">` + s[1] + `</span>`
			if s[2] != "" {
				out += ` <span class="badge badge-rate">` + s[2] + `</span>`
			}
			return out + `</td>`
		})
	})
}

// inlineMethod matches an inline-code HTTP verb anywhere in the document — bare
// (`GET`) or leading an endpoint path (`GET /api/v1/config`). The [^<\n] path
// guard keeps each match on a single line, which also keeps it out of highlighted
// <pre> code blocks (whose content is token <span>s, not bare text).
var inlineMethod = regexp.MustCompile(`<code>(GET|POST|PUT|PATCH|DELETE)( [^<\n]*)?</code>`)

// styleMethods turns every inline-code HTTP verb across the document — endpoint
// headings, the index tables, and prose — into a colored method chip, keeping any
// trailing endpoint path as inline code beside it.
func styleMethods(html string) string {
	return inlineMethod.ReplaceAllStringFunc(html, func(m string) string {
		s := inlineMethod.FindStringSubmatch(m)
		chip := `<span class="method method-` + strings.ToLower(s[1]) + `">` + s[1] + `</span>`
		if path := strings.TrimSpace(s[2]); path != "" {
			chip += ` <code>` + path + `</code>`
		}
		return chip
	})
}

// codeWrapper wraps highlighted code in our own <pre class="hljs"><code> so the
// brand code-block CSS (navy left border, tinted background) applies; the inner
// spans already carry inline token colors from the chroma formatter.
func codeWrapper(w util.BufWriter, _ highlighting.CodeBlockContext, entering bool) {
	if entering {
		w.WriteString(`<pre class="hljs"><code>`)
	} else {
		w.WriteString("</code></pre>")
	}
}

// githubIDs generates heading ids matching GitHub's algorithm so the document's
// existing "#anchor" cross-links resolve in the PDF: lowercase; keep letters,
// digits, "_" and "-"; drop other punctuation; map each space to "-" (no
// collapsing); de-duplicate repeats with "-1", "-2", ...
type githubIDs struct {
	seen map[string]int
}

func newGithubIDs() *githubIDs { return &githubIDs{seen: map[string]int{}} }

func (g *githubIDs) Generate(value []byte, _ ast.NodeKind) []byte {
	var b strings.Builder
	for _, r := range string(value) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(unicode.ToLower(r))
		case r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	slug := b.String()
	if slug == "" {
		slug = "section"
	}
	if n, ok := g.seen[slug]; ok {
		g.seen[slug] = n + 1
		slug = fmt.Sprintf("%s-%d", slug, n)
	} else {
		g.seen[slug] = 1
	}
	return []byte(slug)
}

func (g *githubIDs) Put(value []byte) {
	g.seen[string(value)] = 1
}
