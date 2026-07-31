package public

import (
	"context"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/models"
	"github.com/classyfm/classyfm/internal/render"
)

// Hero defaults, applied whenever the settings row can't be read - no database,
// a query error, or a deleted row. They reproduce the pre-mixed-hero behaviour
// (the 3 latest news, newest first) as closely as a default can.
const (
	heroDefaultNewsCount = 3
	heroDefaultMaxImages = 4
	heroDefaultOrderMode = "updated"
	// heroMaxPerSide bounds each side independently - there is deliberately no
	// overall cap, but a hero of 200 slides is a mistake, not a configuration.
	// /admin/hero clamps to the same number on the way in; this clamp is for
	// rows written before that one existed.
	heroMaxPerSide = 10
)

// heroSlide is one panel of the home page slideshow, whether it came from a news
// item or from an admin-uploaded hero_slides row. The template knows only this
// shape, so the two sources cannot drift apart visually.
type heroSlide struct {
	// Href is "" for a slide with no destination, which renders as an
	// unclickable panel (an <a> with no href).
	Href        string
	TargetBlank bool
	// ImageURL is already resolved - render.HeroImage for news rows, image_url
	// for admin slides - and is "" rather than blank-but-set when there is none.
	ImageURL   string
	BadgeLabel string
	Title      string
	Excerpt    string
	// Date is the zero time for image slides: they carry no publication date, and
	// the template drops the date line rather than printing a 1970 stamp.
	Date time.Time
	// SortTime orders the "updated" mode and is never displayed. News uses
	// published_at rather than news_items.updated_at, which the feed worker
	// rewrites on any content change (see UpsertNewsItem) - a re-derived
	// thumbnail would otherwise resurface a week-old story above a fresh one.
	SortTime time.Time
}

// heroConfig is the hero_settings row with defaults applied and values clamped,
// so callers never have to re-check it.
type heroConfig struct {
	OrderMode string
	NewsCount int32
	MaxImages int32
}

// heroConfig reads the hero settings, falling back to the defaults for anything
// it can't read or recognise. Uncached, like adsForLayout, so an admin's edit
// shows up on the next page load; it's one primary-key row on one page.
func (h *Handler) heroConfig(ctx context.Context) heroConfig {
	cfg := heroConfig{
		OrderMode: heroDefaultOrderMode,
		NewsCount: heroDefaultNewsCount,
		MaxImages: heroDefaultMaxImages,
	}
	if h.q == nil {
		return cfg
	}
	row, err := h.q.GetHeroSettings(ctx)
	if err != nil {
		return cfg
	}
	cfg.NewsCount = clampHeroCount(row.NewsCount)
	cfg.MaxImages = clampHeroCount(row.MaxImages)
	// Validated rather than trusted: an ENUM value added later without matching
	// handler support should degrade to the default ordering, not to a hero that
	// silently comes out in query order.
	switch mode := string(row.OrderMode); mode {
	case "random", "updated", "news_first", "images_first":
		cfg.OrderMode = mode
	}
	return cfg
}

// clampHeroCount bounds a configured slide count to 0..heroMaxPerSide.
func clampHeroCount(n int32) int32 {
	if n < 0 {
		return 0
	}
	if n > heroMaxPerSide {
		return heroMaxPerSide
	}
	return n
}

// heroSlides builds the home page hero: the latest published news mixed with the
// active admin image slides, ordered per /admin/hero's settings.
//
// Every failure degrades to fewer slides rather than an error - a broken hero
// must never take the landing page down with it, the same posture newsGroups and
// adsForLayout take.
func (h *Handler) heroSlides(ctx context.Context) []heroSlide {
	if h.q == nil {
		return nil
	}
	cfg := h.heroConfig(ctx)

	var news, images []heroSlide
	// Both sides are skipped entirely at zero: a LIMIT 0 round-trip is pure waste,
	// and "0 news" is a supported configuration (an all-promo hero).
	if cfg.NewsCount > 0 {
		if items, err := h.q.ListLatestPublished(ctx, cfg.NewsCount); err == nil {
			news = make([]heroSlide, 0, len(items))
			for _, it := range items {
				news = append(news, newsHeroSlide(it))
			}
		}
	}
	if cfg.MaxImages > 0 {
		if rows, err := h.q.ListActiveHeroSlides(ctx, cfg.MaxImages); err == nil {
			images = make([]heroSlide, 0, len(rows))
			for _, s := range rows {
				images = append(images, imageHeroSlide(s))
			}
		}
	}
	return composeHero(news, images, cfg.OrderMode)
}

// newsHeroSlide renders a news item as a hero panel. hot_release items are the
// station's own articles and open on the site; everything else is aggregated and
// links out to the original outlet.
func newsHeroSlide(it sqlc.NewsItem) heroSlide {
	own := it.Source == "hot_release"
	href := it.Url.String
	if own {
		href = "/news/" + it.Slug.String
	}
	return heroSlide{
		Href:        href,
		TargetBlank: !own,
		ImageURL:    render.HeroImage(it.ImageUrl, it.ThumbUrl),
		BadgeLabel:  models.SourceLabel(string(it.Source)),
		Title:       it.Title,
		Excerpt:     it.Excerpt.String,
		Date:        it.PublishedAt,
		SortTime:    it.PublishedAt,
	}
}

// imageHeroSlide renders an admin-uploaded slide as a hero panel.
func imageHeroSlide(s sqlc.HeroSlide) heroSlide {
	href := s.LinkUrl.String
	return heroSlide{
		Href: href,
		// A new-tab flag on a slide with no link would put target="_blank" on an
		// href-less <a>, which means nothing.
		TargetBlank: s.OpenInNewTab && href != "",
		ImageURL:    s.ImageUrl,
		BadgeLabel:  s.BadgeLabel,
		Title:       s.Title,
		Excerpt:     s.Excerpt.String,
		SortTime:    s.UpdatedAt,
	}
}

// composeHero merges the two sides per the configured order mode.
func composeHero(news, images []heroSlide, mode string) []heroSlide {
	out := make([]heroSlide, 0, len(news)+len(images))
	switch mode {
	case "images_first":
		out = append(append(out, images...), news...)
	default:
		out = append(append(out, news...), images...)
	}

	switch mode {
	case "random":
		// The package-level shuffle, not a per-request rand.New(NewSource(now)):
		// concurrent requests landing in the same clock tick would seed
		// identically and produce the same "random" order.
		rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	case "news_first", "images_first":
		// Already in order: each side arrives sorted from its own query.
	default: // "updated"
		// Stable, so ties keep each side's query order - published_at DESC for
		// news, sort_order ASC for slides. Equal timestamps are common (feed
		// items sharing a publish minute), and an unstable sort would reshuffle
		// them on every page load and strand sort_order as a tiebreak.
		sort.SliceStable(out, func(i, j int) bool { return out[i].SortTime.After(out[j].SortTime) })
	}
	return out
}
