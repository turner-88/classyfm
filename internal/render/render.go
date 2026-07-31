// Package render provides an HTML template cache and rendering helpers.
//
// Each public/admin page template is parsed together with the shared layouts and
// partials into its own *template.Template, keyed by page name. In development the
// cache can be rebuilt per request (see New with reload=true) so template edits show
// up without a restart.
package render

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/models"
)

// Renderer renders named page templates into HTTP responses.
type Renderer struct {
	fsys   fs.FS
	reload bool
	cache  map[string]*template.Template
	funcs  template.FuncMap
}

// New builds a Renderer from the given template filesystem (rooted so that
// "layouts/base.html", "partials/*.html", "public/*.html", "admin/*.html" resolve).
// When reload is true, templates are re-parsed on every render (development).
func New(fsys fs.FS, reload bool) (*Renderer, error) {
	// A fresh value per process start (i.e. per deploy) so cache-busted asset URLs
	// (see staticVer below) naturally invalidate any stale browser/CDN cache without
	// requiring a manual purge.
	version := strconv.FormatInt(time.Now().Unix(), 10)
	funcs := defaultFuncs()
	funcs["staticVer"] = func() string { return version }

	r := &Renderer{
		fsys:   fsys,
		reload: reload,
		funcs:  funcs,
	}
	cache, err := r.build()
	if err != nil {
		return nil, err
	}
	r.cache = cache
	return r, nil
}

func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		"now":     func() time.Time { return time.Now() },
		"year":    func() int { return time.Now().Year() },
		"fmtDate": func(t time.Time) string { return t.Format("02 Jan 2006") },
		"fmtTime": func(s string) string { return models.ClockLabel(s) },
		"weekday": func(d any) string {
			switch v := d.(type) {
			case int:
				return models.Weekday(v)
			case int8:
				return models.Weekday(int(v))
			default:
				return ""
			}
		},
		// day_of_week reaches templates as an int8 from sqlc but as a plain int
		// from anything hand-built, so accept both - same as weekday above.
		"weekdayShort": func(d any) string {
			switch v := d.(type) {
			case int:
				return models.WeekdayShort(v)
			case int8:
				return models.WeekdayShort(int(v))
			default:
				return ""
			}
		},
		"sourceLabel": func(s any) string { return models.SourceLabel(fmt.Sprint(s)) },
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
		// hasID marks the selected <option>s of a <select multiple>: the current
		// selection arrives as a []uint64 of ids and each option renders one row.
		"hasID": func(ids []uint64, id uint64) bool { return slices.Contains(ids, id) },
		// eqNum marks the selected <option> of a numeric <select>. day_of_week
		// reaches templates as an int8 from sqlc but as a plain int from a range
		// index, and template `eq` refuses to compare across those types.
		"eqNum": func(a, b any) bool {
			toInt := func(v any) (int64, bool) {
				switch n := v.(type) {
				case int:
					return int64(n), true
				case int8:
					return int64(n), true
				case int32:
					return int64(n), true
				case int64:
					return n, true
				}
				return 0, false
			}
			x, okA := toInt(a)
			y, okB := toInt(b)
			return okA && okB && x == y
		},
		// mod drives row parity in alternating layouts (About's segment rows).
		// A zero divisor returns 0 rather than panicking mid-render, since a
		// template panic would take down the whole page.
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"displayUrl": func(raw string) string {
			u, err := url.Parse(raw)
			if err != nil {
				return raw
			}
			host := strings.TrimPrefix(u.Host, "www.")
			return host + strings.TrimSuffix(u.Path, "/")
		},
		// socialURL turns a broadcaster's stored social value into a profile URL,
		// or "" when it can't safely be linked - templates render the value as
		// plain text in that case, which is what every one of these did before.
		//
		// The columns hold whatever an admin typed, and in practice that is not
		// always a handle: alongside "edo_pruss17" there is instagram
		// "Jasmine Andrea" and facebook "Vivi d' Chressya", which are display
		// names. Anything with whitespace or a slash in it would produce a
		// broken profile link, so it stays unlinked rather than guessing.
		"socialURL": func(platform, handle string) string {
			h := strings.TrimSpace(handle)
			h = strings.TrimPrefix(h, "@")
			if h == "" {
				return ""
			}
			if strings.HasPrefix(h, "http://") || strings.HasPrefix(h, "https://") {
				return h
			}
			if strings.ContainsAny(h, " \t/?&#") {
				return ""
			}
			switch platform {
			case "instagram":
				return "https://instagram.com/" + h
			// The column is still named twitter; the destination is not.
			case "twitter", "x":
				return "https://x.com/" + h
			case "facebook":
				return "https://facebook.com/" + h
			default:
				return ""
			}
		},
		// listImage picks the list-sized image for anywhere other than the
		// hero section (news cards, article detail, admin previews):
		// thumb_url when set, falling back to image_url (the hi-res one)
		// for rows that never got a distinct thumbnail (e.g. hot_release
		// before an upgrade, or admin-authored articles).
		"listImage": func(thumb, image sql.NullString) string {
			if thumb.Valid && thumb.String != "" {
				return thumb.String
			}
			return image.String
		},
		// heroImage is HeroImage; it lives outside defaultFuncs because the home
		// page hero builds its slides in Go and must resolve images by exactly
		// the same rule.
		"heroImage": HeroImage,
		// readingTime labels an article body, at the usual 200 wpm. Returns ""
		// for empty content so the template can drop the whole meta item rather
		// than print "0 min read" on an aggregated item with no body.
		"readingTime": func(s string) string {
			words := len(strings.Fields(s))
			if words == 0 {
				return ""
			}
			return strconv.Itoa(max(words/200, 1)) + " min read"
		},
		// splitParagraphs splits admin-entered body text on blank lines, so
		// multi-paragraph editorial copy (e.g. About Us segments) renders as
		// separate <p> tags instead of collapsing into one block.
		"splitParagraphs": func(s string) []string {
			return strings.Split(strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n")), "\n\n")
		},
	}
}

// HeroImage is listImage's counterpart for a full-width hero (the home page
// slideshow, the news article poster): image_url first, since the thumbnail is
// sized for a card and visibly soft blown up to 16:9, falling back to thumb_url
// for rows that only ever had one. The empty-string check matters as much as the
// Valid one - a Valid-but-blank column would otherwise emit <img src="">, which
// browsers resolve against the page URL and actually fetch.
//
// Exported, unlike the other helpers, because the home hero assembles its slides
// in Go (news items and admin-uploaded slides share one view-model) and picking
// the image there by hand would let the two rules drift apart.
func HeroImage(image, thumb sql.NullString) string {
	if image.Valid && image.String != "" {
		return image.String
	}
	return thumb.String
}

// build parses every page template ("public/*.html", "admin/*.html") together with
// the base layout and all partials, keyed by "public/<name>" / "admin/<name>".
func (r *Renderer) build() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	partials, _ := fs.Glob(r.fsys, "partials/*.html")
	layouts, _ := fs.Glob(r.fsys, "layouts/*.html")

	for _, group := range []string{"public", "admin"} {
		pages, err := fs.Glob(r.fsys, group+"/*.html")
		if err != nil {
			return nil, err
		}
		for _, page := range pages {
			base := strings.TrimSuffix(filepath.Base(page), filepath.Ext(page))
			name := group + "/" + base
			files := append([]string{}, layouts...)
			files = append(files, partials...)
			files = append(files, page)

			tmpl, err := template.New(filepath.Base(page)).
				Funcs(r.funcs).
				ParseFS(r.fsys, files...)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", name, err)
			}
			cache[name] = tmpl
		}
	}
	return cache, nil
}

// Page renders the named template (e.g. "public/home") with data and status code.
func (r *Renderer) Page(w http.ResponseWriter, status int, name string, data any) {
	cache := r.cache
	if r.reload {
		if c, err := r.build(); err == nil {
			cache = c
		} else {
			http.Error(w, "template reload error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	tmpl, ok := cache[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}

	baseTemplate := "base"
	if strings.HasPrefix(name, "admin/") {
		baseTemplate = "admin-base"
	}

	// Render to a buffer first so a template error doesn't write a half page.
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, baseTemplate, data); err != nil {
		http.Error(w, "template render error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.Copy(w, &buf)
}
