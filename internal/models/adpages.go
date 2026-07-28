package models

// AdPage is one public page a banner can be targeted at. Route is the chi route
// pattern the page is registered under (see newRouter in cmd/server/main.go); Key
// is what gets stored in ad_banner_pages.page.
type AdPage struct {
	Key   string
	Label string
	Route string
}

// AdPages is the ordered catalog of targetable pages, and the single source of
// truth for two things that must never drift apart: the public side's route ->
// key lookup, and the admin form's checkbox list.
//
// List and detail pages are separate entries on purpose, so an article page can
// be sold on its own. Adding a page here also needs the key added to the
// ad_banner_pages.page ENUM (see migration 0020).
var AdPages = []AdPage{
	{Key: "home", Label: "Home", Route: "/"},
	{Key: "about", Label: "About", Route: "/about"},
	{Key: "program", Label: "Programs", Route: "/program"},
	{Key: "program_detail", Label: "Program detail", Route: "/program/{slug}"},
	{Key: "live", Label: "Live", Route: "/live"},
	{Key: "news", Label: "News", Route: "/news"},
	{Key: "news_detail", Label: "News article", Route: "/news/{slug}"},
	{Key: "broadcasters", Label: "Broadcasters", Route: "/broadcasters"},
	{Key: "broadcaster_detail", Label: "Broadcaster profile", Route: "/broadcasters/{slug}"},
}

// AdPageKeyForRoute maps a chi route pattern to its targeting key, or "" if the
// route isn't targetable. "" matches no target row, so a page that was never
// registered here shows only banners targeted at every page - the safe default.
func AdPageKeyForRoute(route string) string {
	for _, p := range AdPages {
		if p.Route == route {
			return p.Key
		}
	}
	return ""
}

// ValidAdPageKey reports whether key names a targetable page. Used to reject
// hand-crafted form values before they reach the ENUM column.
func ValidAdPageKey(key string) bool {
	for _, p := range AdPages {
		if p.Key == key {
			return true
		}
	}
	return false
}

// AdPageLabel returns the human-readable label for a targeting key, falling back
// to the key itself so an unknown value (e.g. a row left behind by a removed
// page) still renders as something.
func AdPageLabel(key string) string {
	for _, p := range AdPages {
		if p.Key == key {
			return p.Label
		}
	}
	return key
}
