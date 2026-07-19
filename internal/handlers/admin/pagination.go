package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// adminPageSize is the row count per page for every paginated admin list.
const adminPageSize = 50

// pagination is the view-model consumed by the "pagination" template partial
// and by sortable column headers / filter pills on every admin list page.
type pagination struct {
	Page       int
	TotalPages int
	BaseURL    string     // "<path>?...&page=" - page number appended by the template
	Path       string     // "<path>", no query string
	Params     url.Values // current non-empty q/sort/dir/source params (never "page")
	Search     string     // convenience: Params.Get("q")
	Sort       string     // convenience: Params.Get("sort")
	Dir        string     // convenience: Params.Get("dir"), "asc"|"desc"
}

// paginate reads ?page= from the request and clamps it against the total row
// count, returning the view-model for the "pagination" partial. params carries
// whatever filter/search/sort state the caller wants preserved across page
// links (e.g. {"q": {search}, "sort": {sort}, "dir": {dir}}, and for newsfeed
// also "source"); empty values are dropped automatically. Callers derive the
// LIMIT/OFFSET for their query from the returned Page via Offset.
func paginate(r *http.Request, total int64, path string, params url.Values) pagination {
	clean := url.Values{}
	for k, vals := range params {
		if len(vals) > 0 && vals[0] != "" {
			clean.Set(k, vals[0])
		}
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	totalPages := int((total + adminPageSize - 1) / adminPageSize)
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	baseURL := path + "?page="
	if enc := clean.Encode(); enc != "" {
		baseURL = path + "?" + enc + "&page="
	}
	return pagination{
		Page: page, TotalPages: totalPages,
		BaseURL: baseURL, Path: path, Params: clean,
		Search: clean.Get("q"), Sort: clean.Get("sort"), Dir: clean.Get("dir"),
	}
}

// Offset returns the SQL OFFSET for this page, given adminPageSize as the LIMIT.
func (p pagination) Offset() int32 {
	return int32((p.Page - 1) * adminPageSize)
}

// WithParam returns a link to Path with key set to val (or removed, if val is
// empty) on top of the current params, and "page" always dropped (so changing
// any filter naturally resets to page 1). Used for sort links and, on
// newsfeed, filter pills - both need to change one param while preserving the
// rest of the current search/sort/filter state.
func (p pagination) WithParam(key, val string) string {
	v := url.Values{}
	for k, vals := range p.Params {
		if len(vals) > 0 {
			v.Set(k, vals[0])
		}
	}
	if val == "" {
		v.Del(key)
	} else {
		v.Set(key, val)
	}
	if enc := v.Encode(); enc != "" {
		return p.Path + "?" + enc
	}
	return p.Path
}

// SortURL returns the link for a sortable column header: toggles direction if
// col is already the active sort column, otherwise starts at defaultDir (e.g.
// "asc" for text columns, "desc" for date columns so newest-first is the
// natural first click).
func (p pagination) SortURL(col, defaultDir string) string {
	dir := defaultDir
	if p.Sort == col {
		if p.Dir == "asc" {
			dir = "desc"
		} else {
			dir = "asc"
		}
	}
	v := url.Values{}
	for k, vals := range p.Params {
		if len(vals) > 0 {
			v.Set(k, vals[0])
		}
	}
	v.Set("sort", col)
	v.Set("dir", dir)
	if enc := v.Encode(); enc != "" {
		return p.Path + "?" + enc
	}
	return p.Path
}

// parseSort validates ?sort= against allowed columns (falling back to
// defaultSort) and normalizes ?dir= to "asc"/"desc" (falling back to
// defaultDir). The allow-list is a UX/correctness measure only - sort/dir are
// always bound as SQL params via sqlc.arg(), never string-interpolated, so
// there's no injection risk either way.
func parseSort(r *http.Request, defaultSort, defaultDir string, allowed ...string) (sort, dir string) {
	sort = r.URL.Query().Get("sort")
	ok := false
	for _, a := range allowed {
		if sort == a {
			ok = true
			break
		}
	}
	if !ok {
		sort = defaultSort
	}
	dir = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("dir")))
	if dir != "asc" && dir != "desc" {
		dir = defaultDir
	}
	return sort, dir
}

// searchPattern reads ?q=, trims it, and returns both the raw value (for
// redisplay in the search box) and a MySQL LIKE pattern ("" becomes "%",
// matching every row).
func searchPattern(r *http.Request) (raw, pattern string) {
	raw = strings.TrimSpace(r.URL.Query().Get("q"))
	return raw, "%" + raw + "%"
}
