package public

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/markdown"
	"github.com/classyfm/classyfm/internal/models"
	"github.com/classyfm/classyfm/internal/radio"
	"github.com/classyfm/classyfm/internal/render"
)

// This file implements the public JSON API mounted at /api/v1 (see newRouter). It reuses
// the same sqlc queries and helpers the HTML handlers use, mapping the sqlc rows — whose
// sql.NullString/time.Time fields serialize poorly — into clean snake_case DTOs with
// absolute URLs, so the mobile app never sees {"String":"","Valid":false} shapes.

// apiCache selects the Cache-Control policy for a JSON response.
type apiCache int

const (
	cacheNone  apiCache = iota // no-store: live or per-user data
	cacheShort                 // public, max-age=60: content that changes slowly
)

// writeJSON is the single JSON responder for the API, formalizing the ad-hoc convention
// the polling endpoints already use (Content-Type + Cache-Control + encoder).
func writeJSON(w http.ResponseWriter, status int, cache apiCache, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if cache == cacheShort {
		w.Header().Set("Cache-Control", "public, max-age=60")
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeJSONError sends {"error": msg} with the given status.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, cacheNone, map[string]string{"error": msg})
}

// listEnvelope wraps a list result as {"data": ...}.
func listEnvelope(data any) map[string]any { return map[string]any{"data": data} }

// pageEnvelope wraps a paginated list as {"data": ..., "meta": {...}}.
func pageEnvelope(data any, page, totalPages int, total int64) map[string]any {
	return map[string]any{
		"data": data,
		"meta": map[string]any{"page": page, "total_pages": totalPages, "total": total},
	}
}

// absURL makes a stored path or URL absolute against the site origin. Empty stays empty;
// an already-absolute (http/https) URL — e.g. an aggregated news thumbnail — is returned
// unchanged; a site-relative path ("/uploads/x.jpg") is prefixed with siteURL.
func (h *Handler) absURL(p string) string {
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return h.siteURL + p
}

// ---------- DTOs ----------

type programDTO struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	URL         string `json:"url"`
	OnAir       bool   `json:"on_air"`
}

type scheduleGroupDTO struct {
	FromDay   int8   `json:"from_day"`
	ToDay     int8   `json:"to_day"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Host      string `json:"host,omitempty"`
}

type programDetailDTO struct {
	programDTO
	Schedule     []scheduleGroupDTO `json:"schedule"`
	Broadcasters []broadcasterDTO   `json:"broadcasters"`
}

type broadcasterDTO struct {
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Role       string `json:"role,omitempty"`
	PhotoURL   string `json:"photo_url,omitempty"`
	Bio        string `json:"bio,omitempty"`
	BirthPlace string `json:"birth_place,omitempty"`
	BirthDate  string `json:"birth_date,omitempty"`
	Instagram  string `json:"instagram,omitempty"`
	Twitter    string `json:"twitter,omitempty"`
	Tiktok     string `json:"tiktok,omitempty"`
	URL        string `json:"url"`
	OnAir      bool   `json:"on_air"`
}

type broadcasterDetailDTO struct {
	broadcasterDTO
	Programs []programDTO `json:"programs"`
}

type newsItemDTO struct {
	Source      string    `json:"source"`
	SourceLabel string    `json:"source_label"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug,omitempty"`
	Excerpt     string    `json:"excerpt,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	IsFeatured  bool      `json:"is_featured,omitempty"`
}

type newsDetailDTO struct {
	newsItemDTO
	Paragraphs   []string      `json:"paragraphs"`
	GalleryIndex int           `json:"gallery_index"`
	MiddleImages []string      `json:"middle_images"`
	Related      []newsItemDTO `json:"related"`
}

type newsGroupDTO struct {
	Source string        `json:"source"`
	Label  string        `json:"label"`
	Items  []newsItemDTO `json:"items"`
}

type eventDTO struct {
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Category    string     `json:"category"` // "event" | "promo"
	Description string     `json:"description,omitempty"`
	ImageURL    string     `json:"image_url,omitempty"`
	EventDate   *time.Time `json:"event_date,omitempty"` // nil when unset
	Location    string     `json:"location,omitempty"`
	LinkURL     string     `json:"link_url,omitempty"`
	URL         string     `json:"url"` // on-site /event/{slug}, absolute
}

type podcastDTO struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	SpotifyURL  string `json:"spotify_url,omitempty"`
	ThumbURL    string `json:"thumb_url,omitempty"`
	SeriesName  string `json:"series_name,omitempty"`
	Broadcaster string `json:"broadcaster,omitempty"`
	URL         string `json:"url"`
}

type podcastDetailDTO struct {
	podcastDTO
	Broadcasters []broadcasterDTO `json:"broadcasters"`
}

type seriesDTO struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type scheduleRowDTO struct {
	Title     string `json:"title"`
	Slug      string `json:"slug,omitempty"`
	Host      string `json:"host,omitempty"`
	Image     string `json:"image,omitempty"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	OnAir     bool   `json:"on_air"`
	Progress  int    `json:"progress"`
	Ended     bool   `json:"ended"`
}

type heroSlideDTO struct {
	Href        string     `json:"href,omitempty"`
	TargetBlank bool       `json:"target_blank,omitempty"`
	ImageURL    string     `json:"image_url,omitempty"`
	BadgeLabel  string     `json:"badge_label,omitempty"`
	Title       string     `json:"title"`
	Excerpt     string     `json:"excerpt,omitempty"`
	Date        *time.Time `json:"date,omitempty"`
}

// nowPlayingAPIDTO embeds the radio metadata (whose json tags already suit the API) and
// adds the on-air program when the stream is live.
type nowPlayingAPIDTO struct {
	radio.NowPlaying
	Program *scheduleRowDTO `json:"program,omitempty"`
}

type adBannerDTO struct {
	ImageURL string `json:"image_url"`
	LinkURL  string `json:"link_url,omitempty"`
	Alt      string `json:"alt,omitempty"`
	Title    string `json:"title,omitempty"`
}

// adSlotDTO mirrors the web layout's per-placement payload: the banners plus how the
// client should present them (all stacked, or rotated) and what an empty slot does.
type adSlotDTO struct {
	Slideshow       bool          `json:"slideshow"`
	RotateMs        int           `json:"rotate_ms"`
	Placeholder     bool          `json:"placeholder"`
	PlaceholderText string        `json:"placeholder_text,omitempty"`
	Banners         []adBannerDTO `json:"banners"`
}

// adsDTO is the whole ad payload for a page, keyed by placement.
type adsDTO struct {
	Top    adSlotDTO `json:"top"`
	Bottom adSlotDTO `json:"bottom"`
}

// contactDTO is the admin-managed contact block on /config: raw values plus the derived
// dialable links (wa.me / tel:). WhatsAppURL/PhoneHref are omitted when their number has
// no digits. Unlike the website widget, the API is never gated on FEATURE_WHATSAPP — it
// returns whatever the admin saved and lets the app decide how to present it.
type contactDTO struct {
	WhatsAppNumber  string `json:"whatsapp_number,omitempty"`
	WhatsAppMessage string `json:"whatsapp_message,omitempty"`
	WhatsAppURL     string `json:"whatsapp_url,omitempty"`
	Phone           string `json:"phone,omitempty"`
	PhoneHref       string `json:"phone_href,omitempty"`
	Email           string `json:"email,omitempty"`
}

// legalListItemDTO is one row of the /legal index: enough to build a settings/legal menu
// without shipping the full body.
type legalListItemDTO struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

// legalPageDTO is one legal page in full: the Markdown source plus a sanitized HTML
// rendering, so the app can render either natively or in a WebView.
type legalPageDTO struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Intro     string    `json:"intro,omitempty"`
	Body      string    `json:"body"`
	BodyHTML  string    `json:"body_html"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---------- mappers ----------

func (h *Handler) toProgram(p sqlc.Program, onAir bool) programDTO {
	return programDTO{
		Title:       p.Title,
		Slug:        p.Slug,
		Description: p.Description.String,
		ImageURL:    h.absURL(p.ImageUrl.String),
		URL:         h.absURL("/program/" + p.Slug),
		OnAir:       onAir,
	}
}

func (h *Handler) toBroadcaster(b sqlc.Broadcaster, onAir bool) broadcasterDTO {
	return broadcasterDTO{
		Name:       b.Name,
		Slug:       b.Slug,
		Role:       b.Role.String,
		PhotoURL:   h.absURL(b.PhotoUrl.String),
		Bio:        b.Bio.String,
		BirthPlace: b.BirthPlace.String,
		BirthDate:  b.BirthDate.String,
		Instagram:  b.Instagram.String,
		Twitter:    b.Twitter.String,
		Tiktok:     b.Tiktok.String,
		URL:        h.absURL("/broadcasters/" + b.Slug),
		OnAir:      onAir,
	}
}

func toScheduleGroup(g models.ScheduleGroup) scheduleGroupDTO {
	return scheduleGroupDTO{
		FromDay:   g.FromDay,
		ToDay:     g.ToDay,
		StartTime: g.StartTime,
		EndTime:   g.EndTime,
		Host:      g.Host,
	}
}

// toNewsItem mirrors newsHeroSlide's link rule: the station's own hot_release articles
// resolve to the on-site /news/{slug}; every other (aggregated) source keeps its external
// URL.
func (h *Handler) toNewsItem(it sqlc.NewsItem, featured bool) newsItemDTO {
	url := it.Url.String
	if it.Source == "hot_release" {
		url = h.absURL("/news/" + it.Slug.String)
	}
	return newsItemDTO{
		Source:      string(it.Source),
		SourceLabel: models.SourceLabel(string(it.Source)),
		Title:       it.Title,
		Slug:        it.Slug.String,
		Excerpt:     it.Excerpt.String,
		ImageURL:    h.absURL(render.HeroImage(it.ImageUrl, it.ThumbUrl)),
		URL:         url,
		PublishedAt: it.PublishedAt,
		IsFeatured:  featured,
	}
}

// toEvent maps an event row into its DTO: the nullable image/date/location/link fields
// collapse to empty/omitted rather than leaking {"String":...,"Valid":...} shapes, and the
// public URL always resolves to the on-site /event/{slug}.
func (h *Handler) toEvent(e sqlc.Event) eventDTO {
	dto := eventDTO{
		Title:       e.Title,
		Slug:        e.Slug,
		Category:    string(e.Category),
		Description: e.Description,
		ImageURL:    h.absURL(e.ImageUrl.String),
		Location:    e.Location.String,
		LinkURL:     e.LinkUrl.String,
		URL:         h.absURL("/event/" + e.Slug),
	}
	if e.EventDate.Valid {
		d := e.EventDate.Time
		dto.EventDate = &d
	}
	return dto
}

func (h *Handler) toScheduleRow(r scheduleRow) scheduleRowDTO {
	return scheduleRowDTO{
		Title:     r.ProgramTitle,
		Slug:      r.ProgramSlug,
		Host:      r.ProgramHost,
		Image:     h.absURL(r.ProgramImage),
		StartTime: r.StartTime,
		EndTime:   r.EndTime,
		OnAir:     r.OnAir,
		Progress:  r.Progress,
		Ended:     r.Ended,
	}
}

func (h *Handler) toHeroSlide(s heroSlide) heroSlideDTO {
	dto := heroSlideDTO{
		Href:        h.absURL(s.Href),
		TargetBlank: s.TargetBlank,
		ImageURL:    h.absURL(s.ImageURL),
		BadgeLabel:  s.BadgeLabel,
		Title:       s.Title,
		Excerpt:     s.Excerpt,
	}
	if !s.Date.IsZero() {
		d := s.Date
		dto.Date = &d
	}
	return dto
}

// toAdSlot maps a rendered ad slot into its DTO, absolutizing the banner image and link
// URLs (uploads are site-relative; an external link URL passes through absURL unchanged).
func (h *Handler) toAdSlot(s adSlot) adSlotDTO {
	banners := []adBannerDTO{}
	for _, b := range s.Banners {
		banners = append(banners, adBannerDTO{
			ImageURL: h.absURL(b.ImageURL),
			LinkURL:  h.absURL(b.LinkURL),
			Alt:      b.Alt,
			Title:    b.Title,
		})
	}
	return adSlotDTO{
		Slideshow:       s.Slideshow,
		RotateMs:        s.RotateMs,
		Placeholder:     s.Placeholder,
		PlaceholderText: s.PlaceholderText,
		Banners:         banners,
	}
}

// ---------- content handlers ----------

// APIPrograms returns the active program roster with an on-air flag on each.
func (h *Handler) APIPrograms(w http.ResponseWriter, r *http.Request) {
	out := []programDTO{}
	if h.q != nil {
		if list, err := h.q.ListActivePrograms(r.Context()); err == nil {
			onAir := computeOnAir(r.Context(), h.q)
			for _, p := range list {
				out = append(out, h.toProgram(p, onAir[p.ID]))
			}
		}
	}
	writeJSON(w, http.StatusOK, cacheShort, listEnvelope(out))
}

// APIProgramDetail returns one program with its full weekly schedule and broadcasters.
func (h *Handler) APIProgramDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	p, err := h.q.GetProgramBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil || !p.IsActive {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	var slots []models.ProgramSlot
	if rows, err := h.q.ListSchedulesForProgram(r.Context(), p.ID); err == nil {
		for _, row := range rows {
			slots = append(slots, models.ProgramSlot{
				Day: row.DayOfWeek, StartTime: models.ClockLabel(row.StartTime),
				EndTime: models.ClockLabel(row.EndTime), Host: row.BroadcasterName.String,
			})
		}
	}
	onAir := computeOnAir(r.Context(), h.q)[p.ID]

	schedule := []scheduleGroupDTO{}
	for _, g := range models.GroupSlots(slots) {
		schedule = append(schedule, toScheduleGroup(g))
	}
	broadcasters := []broadcasterDTO{}
	if bcs, err := h.q.ListBroadcastersForProgram(r.Context(), sqlc.ListBroadcastersForProgramParams{ProgramID: p.ID}); err == nil {
		for _, b := range bcs {
			broadcasters = append(broadcasters, h.toBroadcaster(b, false))
		}
	}

	writeJSON(w, http.StatusOK, cacheShort, programDetailDTO{
		programDTO:   h.toProgram(p, onAir),
		Schedule:     schedule,
		Broadcasters: broadcasters,
	})
}

// APIBroadcasters returns the active broadcaster roster, each flagged if any program they
// present is airing right now.
func (h *Handler) APIBroadcasters(w http.ResponseWriter, r *http.Request) {
	out := []broadcasterDTO{}
	if h.q != nil {
		list, _ := h.q.ListActiveBroadcasters(r.Context())
		onAirB := map[uint64]bool{}
		for _, s := range h.onAirSlotsWithBroadcasters(r.Context()) {
			for _, b := range s.Broadcasters {
				onAirB[b.ID] = true
			}
		}
		for _, b := range list {
			out = append(out, h.toBroadcaster(b, onAirB[b.ID]))
		}
	}
	writeJSON(w, http.StatusOK, cacheShort, listEnvelope(out))
}

// APIBroadcasterDetail returns one broadcaster and the programs they present.
func (h *Handler) APIBroadcasterDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	c, err := h.q.GetActiveBroadcasterBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	programs := []programDTO{}
	selfOnAir := false
	if list, err := h.q.ListProgramsForBroadcaster(r.Context(), sqlc.ListProgramsForBroadcasterParams{BroadcasterID: c.ID}); err == nil && len(list) > 0 {
		// On air only for the slot this broadcaster personally presents right now.
		onAirProg := map[uint64]bool{}
		for _, s := range h.onAirSlotsWithBroadcasters(r.Context()) {
			for _, b := range s.Broadcasters {
				if b.ID == c.ID {
					onAirProg[s.ProgramID] = true
				}
			}
		}
		for _, p := range list {
			if onAirProg[p.ID] {
				selfOnAir = true
			}
			programs = append(programs, h.toProgram(p, onAirProg[p.ID]))
		}
	}

	writeJSON(w, http.StatusOK, cacheShort, broadcasterDetailDTO{
		broadcasterDTO: h.toBroadcaster(c, selfOnAir),
		Programs:       programs,
	})
}

// APINews returns the newsfeed. Without ?source= it returns the grouped-by-source preview
// (mirroring the web /news landing); with a valid ?source= it returns that source's items
// paginated via ?page=.
func (h *Handler) APINews(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if !isValidSourceFilter(source) {
		source = ""
	}

	if source == "" {
		out := []newsGroupDTO{}
		for _, g := range h.newsGroups(r.Context(), []string{"klikpositif", "katasumbar", "hot_release", "youtube"}, 7, true) {
			items := []newsItemDTO{}
			for _, it := range g.Items {
				items = append(items, h.toNewsItem(it.NewsItem, it.IsFeatured))
			}
			out = append(out, newsGroupDTO{Source: g.Source, Label: g.Label, Items: items})
		}
		writeJSON(w, http.StatusOK, cacheShort, listEnvelope(out))
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := int32((page - 1) * newsPageSize)

	items := []newsItemDTO{}
	var total int64
	if h.q != nil {
		src := sqlc.NewsItemsSource(source)
		rows, _ := h.q.ListPublishedNewsBySource(r.Context(), sqlc.ListPublishedNewsBySourceParams{Source: src, Limit: newsPageSize, Offset: offset})
		for _, it := range rows {
			items = append(items, h.toNewsItem(it, it.IsFeatured))
		}
		total, _ = h.q.CountPublishedNewsBySource(r.Context(), src)
	}
	writeJSON(w, http.StatusOK, cacheShort, pageEnvelope(items, page, totalPagesFor(total), total))
}

// APINewsDetail returns one hot_release article with split paragraphs, its mid-article
// gallery, and related articles.
func (h *Handler) APINewsDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	slug := chi.URLParam(r, "slug")
	item, err := h.q.GetPublishedNewsItemBySlug(r.Context(), sql.NullString{String: slug, Valid: true})
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	paragraphs := []string{}
	if item.Content.Valid && strings.TrimSpace(item.Content.String) != "" {
		paragraphs = render.SplitParagraphs(item.Content.String)
	}
	middle := []string{}
	for _, u := range decodeMiddleImages(item.MiddleImages) {
		middle = append(middle, h.absURL(u))
	}
	related := []newsItemDTO{}
	for _, it := range h.relatedNews(r.Context(), item.ID) {
		related = append(related, h.toNewsItem(it.NewsItem, it.IsFeatured))
	}

	writeJSON(w, http.StatusOK, cacheShort, newsDetailDTO{
		newsItemDTO:  h.toNewsItem(item, item.IsFeatured),
		Paragraphs:   paragraphs,
		GalleryIndex: len(paragraphs) / 2,
		MiddleImages: middle,
		Related:      related,
	})
}

// APIEvents returns the published event list, newest first, optionally filtered to one
// category via ?category=event|promo and paginated via ?page=. An unknown category falls
// back to the unfiltered list rather than 404, mirroring the HTML Event handler.
func (h *Handler) APIEvents(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := int32((page - 1) * newsPageSize)

	items := []eventDTO{}
	var total int64
	if h.q != nil {
		category := ""
		if sel := r.URL.Query().Get("category"); sel == "event" || sel == "promo" {
			category = sel
		}
		if category != "" {
			rows, _ := h.q.ListPublishedEventsByCategory(r.Context(), sqlc.ListPublishedEventsByCategoryParams{
				Category: sqlc.EventsCategory(category), Limit: newsPageSize, Offset: offset,
			})
			for _, e := range rows {
				items = append(items, h.toEvent(e))
			}
			total, _ = h.q.CountPublishedEventsByCategory(r.Context(), sqlc.EventsCategory(category))
		} else {
			rows, _ := h.q.ListPublishedEvents(r.Context(), sqlc.ListPublishedEventsParams{Limit: newsPageSize, Offset: offset})
			for _, e := range rows {
				items = append(items, h.toEvent(e))
			}
			total, _ = h.q.CountPublishedEvents(r.Context())
		}
	}
	writeJSON(w, http.StatusOK, cacheShort, pageEnvelope(items, page, totalPagesFor(total), total))
}

// APIEventDetail returns one published event by slug.
func (h *Handler) APIEventDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	slug := chi.URLParam(r, "slug")
	item, err := h.q.GetPublishedEventBySlug(r.Context(), slug)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, cacheShort, h.toEvent(item))
}

// APIPodcasts returns the published podcast list, newest first, optionally filtered to one
// series via ?series=<slug> and paginated via ?page=.
func (h *Handler) APIPodcasts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := int32((page - 1) * newsPageSize)

	items := []podcastDTO{}
	var total int64
	if h.q != nil {
		selectedSeries := ""
		if sel := r.URL.Query().Get("series"); sel != "" {
			if _, err := h.q.GetPodcastSeriesBySlug(r.Context(), sel); err == nil {
				selectedSeries = sel
			}
		}
		if selectedSeries != "" {
			rows, _ := h.q.ListPublishedPodcastsBySeriesSlug(r.Context(), sqlc.ListPublishedPodcastsBySeriesSlugParams{Slug: selectedSeries, Limit: newsPageSize, Offset: offset})
			for _, p := range rows {
				items = append(items, h.toPodcastList(p.Slug, p.Title, p.Description, p.SpotifyUrl, p.ThumbUrl, p.SeriesName, p.BroadcasterName))
			}
			total, _ = h.q.CountPublishedPodcastsBySeriesSlug(r.Context(), selectedSeries)
		} else {
			rows, _ := h.q.ListPublishedPodcasts(r.Context(), sqlc.ListPublishedPodcastsParams{Limit: newsPageSize, Offset: offset})
			for _, p := range rows {
				items = append(items, h.toPodcastList(p.Slug, p.Title, p.Description, p.SpotifyUrl, p.ThumbUrl, p.SeriesName, p.BroadcasterName))
			}
			total, _ = h.q.CountPublishedPodcasts(r.Context())
		}
	}
	writeJSON(w, http.StatusOK, cacheShort, pageEnvelope(items, page, totalPagesFor(total), total))
}

// toPodcastList maps the shared fields the two podcast-list queries return (typed as
// distinct row structs by sqlc) into one DTO.
func (h *Handler) toPodcastList(slug, title, description, spotify string, thumb sql.NullString, seriesName string, broadcaster sql.NullString) podcastDTO {
	return podcastDTO{
		Title:       title,
		Slug:        slug,
		Description: description,
		SpotifyURL:  h.absURL(spotify),
		ThumbURL:    h.absURL(thumb.String),
		SeriesName:  seriesName,
		Broadcaster: broadcaster.String,
		URL:         h.absURL("/podcast/" + slug),
	}
}

// APIPodcastDetail returns one podcast with its series name and broadcasters.
func (h *Handler) APIPodcastDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	item, err := h.q.GetPublishedPodcastBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	seriesName := ""
	if s, err := h.q.GetPodcastSeries(r.Context(), item.SeriesID); err == nil {
		seriesName = s.Name
	}
	broadcasters := []broadcasterDTO{}
	if bcs, err := h.q.ListPodcastBroadcasters(r.Context(), item.ID); err == nil {
		for _, b := range bcs {
			broadcasters = append(broadcasters, h.toBroadcaster(b, false))
		}
	}

	writeJSON(w, http.StatusOK, cacheShort, podcastDetailDTO{
		podcastDTO: podcastDTO{
			Title:       item.Title,
			Slug:        item.Slug,
			Description: item.Description,
			SpotifyURL:  item.SpotifyUrl,
			ThumbURL:    h.absURL(item.ThumbUrl.String),
			SeriesName:  seriesName,
			URL:         h.absURL("/podcast/" + item.Slug),
		},
		Broadcasters: broadcasters,
	})
}

// APIPodcastSeries returns the podcast series (for a filter picker).
func (h *Handler) APIPodcastSeries(w http.ResponseWriter, r *http.Request) {
	out := []seriesDTO{}
	if h.q != nil {
		if list, err := h.q.ListActivePodcastSeries(r.Context()); err == nil {
			for _, s := range list {
				out = append(out, seriesDTO{Name: s.Name, Slug: s.Slug})
			}
		}
	}
	writeJSON(w, http.StatusOK, cacheShort, listEnvelope(out))
}

// APIAbout returns the About page content: banner, text segments, and a broadcaster
// preview.
func (h *Handler) APIAbout(w http.ResponseWriter, r *http.Request) {
	type bannerDTO struct {
		MediaType string `json:"media_type"`
		ImageURL  string `json:"image_url,omitempty"`
		VideoURL  string `json:"video_url,omitempty"`
		EmbedURL  string `json:"embed_url,omitempty"`
	}
	type segmentDTO struct {
		Segment string `json:"segment"`
		Title   string `json:"title"`
		Body    string `json:"body"`
	}

	var banner bannerDTO
	segments := []segmentDTO{}
	broadcasters := []broadcasterDTO{}
	if h.q != nil {
		if b, err := h.q.GetAboutBanner(r.Context()); err == nil {
			banner = bannerDTO{MediaType: string(b.MediaType), ImageURL: h.absURL(b.ImageUrl.String), VideoURL: b.VideoUrl.String}
			if b.MediaType == sqlc.AboutPageBannerMediaTypeVideo && b.VideoUrl.Valid {
				if u, ok := youtubeEmbedURL(b.VideoUrl.String); ok {
					banner.EmbedURL = u
				}
			}
		}
		if segs, err := h.q.ListAboutSegments(r.Context()); err == nil {
			for _, s := range segs {
				segments = append(segments, segmentDTO{Segment: string(s.Segment), Title: s.Title, Body: s.Body})
			}
		}
		if list, err := h.q.ListActiveBroadcasters(r.Context()); err == nil {
			for _, b := range capSlice(list, broadcasterPreviewMax) {
				broadcasters = append(broadcasters, h.toBroadcaster(b, false))
			}
		}
	}

	writeJSON(w, http.StatusOK, cacheShort, map[string]any{
		"banner":       banner,
		"segments":     segments,
		"broadcasters": broadcasters,
	})
}

// APIAds returns the ad banners for a page, grouped into top/bottom placement slots.
// ?page= selects the target page (see models.AdPages); an empty page returns only the
// banners targeted at every page, matching the web layout's behavior for an unmatched
// route.
func (h *Handler) APIAds(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page != "" && !models.ValidAdPageKey(page) {
		writeJSONError(w, http.StatusBadRequest, "invalid page")
		return
	}
	slots := h.adsForLayout(r.Context(), page)
	writeJSON(w, http.StatusOK, cacheShort, adsDTO{
		Top:    h.toAdSlot(slots.Top),
		Bottom: h.toAdSlot(slots.Bottom),
	})
}

// ---------- feed / live handlers ----------

// APINowPlaying returns the stream's now-playing metadata plus the on-air program when
// live (same body as the web /api/nowplaying).
func (h *Handler) APINowPlaying(w http.ResponseWriter, r *http.Request) {
	np := h.radio.Current(r.Context())
	dto := nowPlayingAPIDTO{NowPlaying: np}
	if np.Live {
		if row := h.currentOnAir(r.Context()); row.ProgramTitle != "" {
			sr := h.toScheduleRow(row)
			dto.Program = &sr
		}
	}
	writeJSON(w, http.StatusOK, cacheNone, dto)
}

// APIScheduleToday returns today's full schedule with live on-air/progress state.
func (h *Handler) APIScheduleToday(w http.ResponseWriter, r *http.Request) {
	out := []scheduleRowDTO{}
	for _, row := range h.todayScheduleRows(r.Context()) {
		out = append(out, h.toScheduleRow(row))
	}
	writeJSON(w, http.StatusOK, cacheNone, listEnvelope(out))
}

// APIScheduleCurrent returns the currently on-air program, or {"on_air": false}.
func (h *Handler) APIScheduleCurrent(w http.ResponseWriter, r *http.Request) {
	if row, _ := currentScheduleRow(h.todayScheduleRows(r.Context())); row != nil {
		writeJSON(w, http.StatusOK, cacheNone, h.toScheduleRow(*row))
		return
	}
	writeJSON(w, http.StatusOK, cacheNone, map[string]any{"on_air": false})
}

// APITikTokLive returns the TikTok card's live state.
func (h *Handler) APITikTokLive(w http.ResponseWriter, r *http.Request) {
	on, title := tiktokLiveState(h.tiktok.Current())
	writeJSON(w, http.StatusOK, cacheNone, map[string]any{"live": on, "title": title})
}

// APIConfig is the app's bootstrap call: station identity, stream URL, and social links.
// (Chat now runs directly against Firebase, so no chat flags are surfaced here.)
func (h *Handler) APIConfig(w http.ResponseWriter, r *http.Request) {
	social := map[string]string{}
	var contact *contactDTO
	if h.q != nil {
		if links, err := h.q.ListMediaLinks(r.Context()); err == nil {
			for _, l := range links {
				social[string(l.Platform)] = l.Url
			}
		}
		if c, err := h.q.GetContactSettings(r.Context()); err == nil {
			contact = &contactDTO{
				WhatsAppNumber:  c.WhatsappNumber,
				WhatsAppMessage: c.WhatsappMessage,
				WhatsAppURL:     whatsAppURL(c.WhatsappNumber, c.WhatsappMessage),
				Phone:           c.Phone,
				PhoneHref:       telHref(c.Phone),
				Email:           c.Email,
			}
		}
	}
	writeJSON(w, http.StatusOK, cacheShort, map[string]any{
		"station":    map[string]string{"name": h.station, "slogan": h.slogan},
		"stream_url": h.radio.StreamURL(),
		"social":     social,
		"contact":    contact,
	})
}

// APILegalPages lists the legal pages (privacy, terms) with just slug/title/updated_at,
// for building a settings/legal menu. Degrades to {"data": []} when the DB is down.
func (h *Handler) APILegalPages(w http.ResponseWriter, r *http.Request) {
	out := []legalListItemDTO{}
	if h.q != nil {
		if pages, err := h.q.ListLegalPages(r.Context()); err == nil {
			for _, p := range pages {
				out = append(out, legalListItemDTO{
					Slug:      string(p.Slug),
					Title:     p.Title,
					UpdatedAt: p.UpdatedAt,
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, cacheShort, listEnvelope(out))
}

// APILegalPage returns one legal page in full. The {slug} must be a known legal slug
// ("privacy" or "terms"); anything else is 404, as is a missing row or an unavailable DB.
func (h *Handler) APILegalPage(w http.ResponseWriter, r *http.Request) {
	slug := sqlc.LegalPagesSlug(chi.URLParam(r, "slug"))
	if slug != sqlc.LegalPagesSlugPrivacy && slug != sqlc.LegalPagesSlugTerms {
		writeJSONError(w, http.StatusNotFound, "legal page not found")
		return
	}
	if h.q == nil {
		writeJSONError(w, http.StatusNotFound, "legal page not found")
		return
	}
	p, err := h.q.GetLegalPage(r.Context(), slug)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "legal page not found")
		return
	}
	writeJSON(w, http.StatusOK, cacheShort, legalPageDTO{
		Slug:      string(p.Slug),
		Title:     p.Title,
		Intro:     p.Intro,
		Body:      p.Body,
		BodyHTML:  string(markdown.ToHTML(p.Body)),
		UpdatedAt: p.UpdatedAt,
	})
}

// APIHome is the aggregate home-screen feed: hero, on-air now, program preview, and the
// grouped newsfeed — one request mirroring the web landing page.
func (h *Handler) APIHome(w http.ResponseWriter, r *http.Request) {
	hero := []heroSlideDTO{}
	for _, s := range h.heroSlides(r.Context()) {
		hero = append(hero, h.toHeroSlide(s))
	}

	programs := []programDTO{}
	if h.q != nil {
		if list, err := h.q.ListActivePrograms(r.Context()); err == nil {
			onAir := computeOnAir(r.Context(), h.q)
			for _, p := range capSlice(list, homeProgramsMax) {
				programs = append(programs, h.toProgram(p, onAir[p.ID]))
			}
		}
	}

	news := []newsGroupDTO{}
	for _, g := range h.newsGroups(r.Context(), []string{"klikpositif", "katasumbar", "hot_release", "youtube"}, 4, true) {
		items := []newsItemDTO{}
		for _, it := range g.Items {
			items = append(items, h.toNewsItem(it.NewsItem, it.IsFeatured))
		}
		news = append(news, newsGroupDTO{Source: g.Source, Label: g.Label, Items: items})
	}

	var onAir *scheduleRowDTO
	if row := h.currentOnAir(r.Context()); row.ProgramTitle != "" {
		sr := h.toScheduleRow(row)
		onAir = &sr
	}

	writeJSON(w, http.StatusOK, cacheShort, map[string]any{
		"hero":     hero,
		"on_air":   onAir,
		"programs": programs,
		"news":     news,
	})
}

// totalPagesFor converts a total row count into a page count (minimum 1) for newsPageSize.
func totalPagesFor(total int64) int {
	pages := int((total + newsPageSize - 1) / newsPageSize)
	if pages < 1 {
		return 1
	}
	return pages
}
