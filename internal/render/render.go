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
		"sourceLabel": func(s any) string { return models.SourceLabel(fmt.Sprint(s)) },
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
		"displayUrl": func(raw string) string {
			u, err := url.Parse(raw)
			if err != nil {
				return raw
			}
			host := strings.TrimPrefix(u.Host, "www.")
			return host + strings.TrimSuffix(u.Path, "/")
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
	}
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
