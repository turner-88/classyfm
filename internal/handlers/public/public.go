// Package public holds HTTP handlers for the public-facing pages.
package public

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/models"
	"github.com/classyfm/classyfm/internal/radio"
	"github.com/classyfm/classyfm/internal/render"
)

// onAirTitleTTL bounds how long the current on-air program title is cached, so a
// burst of /api/nowplaying pollers (one per visitor, every 15s, on every page)
// doesn't hit MySQL on every request.
const onAirTitleTTL = 20 * time.Second

// stationLoc is the station's local timezone (WIB, Padang/West Sumatra), used for
// all schedule/on-air comparisons so correctness doesn't depend on the host OS's
// configured timezone.
var stationLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}()

// newsPageSize is the number of items per page on the full News listing.
const newsPageSize = 12

// Handler renders the public pages.
type Handler struct {
	r       *render.Renderer
	radio   *radio.Service
	q       *sqlc.Queries // nil if no database is configured
	station string
	slogan  string
	siteURL string

	// onAirMu guards a short-TTL cache of the current on-air program, populated
	// by currentOnAir. See onAirTitleTTL.
	onAirMu     sync.RWMutex
	onAirRow    scheduleRow
	onAirCached time.Time
}

// New constructs the public handler. q may be nil in early phases / when no database
// is configured, in which case data-backed sections degrade to empty rather than erroring.
func New(r *render.Renderer, radioSvc *radio.Service, q *sqlc.Queries, station, slogan, siteURL string) *Handler {
	return &Handler{r: r, radio: radioSvc, q: q, station: station, slogan: slogan, siteURL: siteURL}
}

// baseData is the common view-model every page embeds (used by the layout, player,
// and SEO meta tags).
type baseData struct {
	Title         string
	Nav           string // active nav key: home|program|media|news
	StationName   string
	StationSlogan string // tagline; the floating player's last-resort subtitle
	StreamURL     string
	Description   string
	CanonicalURL  string
	OGImage       string // absolute URL; blank suppresses the og:image/twitter:image tags
	Instagram     string // social links for the footer (admin-managed, see /admin/media); blank hides the icon
	Facebook      string
	X             string
	YouTube       string
	Spotify       string
	TikTok        string
	Ads           adSlots // admin-managed ad banners per placement slot (see /admin/ads)
}

// adBanner is one rendered creative: an image, an optional click-through, and the
// accessible text for both.
type adBanner struct {
	ImageURL string
	LinkURL  string
	Alt      string
	Title    string
}

// adSlot is one placement's render payload: the banners plus how the layout should
// present them (all stacked, or rotated client-side). Placeholder says what an
// empty slot does - collapse to nothing, or hold its space with a dashed box.
type adSlot struct {
	Slideshow       bool
	RotateMs        int
	Placeholder     bool
	PlaceholderText string
	Banners         []adBanner
}

// adSlots is the whole ad payload for a page, keyed by placement. A struct with
// named fields rather than a map, so a typo in the layout is a template error
// instead of a silently empty slot.
type adSlots struct {
	Top    adSlot
	Bottom adSlot
}

// base builds the common view-model. description should be a one-sentence summary
// of the page for the meta description / Open Graph tags; the canonical URL is
// derived from the request path (query strings such as pagination/filters are
// intentionally excluded so paginated/filtered variants canonicalize to the plain page).
func (h *Handler) base(r *http.Request, title, nav, description string) baseData {
	b := baseData{
		Title:         title,
		Nav:           nav,
		StationName:   h.station,
		StationSlogan: h.slogan,
		StreamURL:     h.radio.StreamURL(),
		Description:   description,
		CanonicalURL:  h.siteURL + r.URL.Path,
	}
	if h.q != nil {
		if links, err := h.q.ListMediaLinks(r.Context()); err == nil {
			for _, l := range links {
				switch l.Platform {
				case sqlc.MediaLinksPlatformInstagram:
					b.Instagram = l.Url
				case sqlc.MediaLinksPlatformFacebook:
					b.Facebook = l.Url
				case sqlc.MediaLinksPlatformX:
					b.X = l.Url
				case sqlc.MediaLinksPlatformYoutube:
					b.YouTube = l.Url
				case sqlc.MediaLinksPlatformSpotify:
					b.Spotify = l.Url
				case sqlc.MediaLinksPlatformTiktok:
					b.TikTok = l.Url
				}
			}
		}
	}
	b.Ads = h.adsForLayout(r.Context(), adPageKey(r))
	return b
}

// adPageKey returns the ad-targeting key for the page being rendered, derived
// from the matched chi route pattern. Nav can't serve this purpose: /program and
// /program/{slug} share a nav value (as do the news and broadcaster pages), and
// Nav also drives nav highlighting and hides the floating player on /live.
//
// An unrecognized route - including the two error paths, which have no matched
// pattern - returns "", which matches no target row, so only banners targeted at
// every page would render there.
func adPageKey(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return ""
	}
	return models.AdPageKeyForRoute(rctx.RoutePattern())
}

// adsForLayout loads the ad banners the layout renders for the given page,
// grouped by placement slot. pageKey selects which banners apply; a banner with
// no target rows applies to every page. Deliberately uncached so an admin's edit
// shows up on the next page load; both queries hit small, indexed tables.
//
// Anything that goes wrong - no database, a query error, a slot the layout has no
// render site for - degrades to an empty slot, which renders nothing.
func (h *Handler) adsForLayout(ctx context.Context, pageKey string) adSlots {
	var out adSlots
	if h.q == nil {
		return out
	}
	slots, err := h.q.ListAdSlots(ctx)
	if err != nil {
		return out
	}
	banners, err := h.q.ListActiveAdBannersForPage(ctx, sqlc.AdBannerPagesPage(pageKey))
	if err != nil {
		return out
	}

	for _, s := range slots {
		// A disabled slot renders nothing at all, not even its placeholder.
		if !s.IsActive {
			continue
		}
		rotateMs := int(s.RotateSecs) * 1000
		if rotateMs < 2000 {
			rotateMs = 6000
		}
		cur := adSlot{
			Slideshow:       s.DisplayMode == sqlc.AdSlotsDisplayModeSlideshow,
			RotateMs:        rotateMs,
			Placeholder:     s.ShowPlaceholder,
			PlaceholderText: s.PlaceholderText,
		}
		for _, b := range banners {
			if string(b.Slot) != string(s.Slot) {
				continue
			}
			alt := b.AltText
			if alt == "" {
				alt = b.Title
			}
			cur.Banners = append(cur.Banners, adBanner{
				ImageURL: b.ImageUrl,
				LinkURL:  b.LinkUrl.String,
				Alt:      alt,
				Title:    b.Title,
			})
		}
		switch s.Slot {
		case sqlc.AdSlotsSlotTop:
			out.Top = cur
		case sqlc.AdSlotsSlotBottom:
			out.Bottom = cur
		}
	}
	return out
}

// scheduleRow is the view-model for one weekly schedule slot (used on Home's
// "On Air" card and Live's full schedule list).
type scheduleRow struct {
	StartTime    string
	EndTime      string
	ProgramTitle string
	ProgramSlug  string
	ProgramHost  string
	ProgramImage string
	OnAir        bool
	Progress     int  // 0-100, only meaningful when OnAir
	Ended        bool // slot already finished earlier today (dimmed in the timeline)
}

// scheduleState is the minimal per-row poll payload for /api/schedule/today:
// title/time/host/image are static for the day, only on-air/ended/progress change.
type scheduleState struct {
	OnAir    bool `json:"on_air"`
	Progress int  `json:"progress"`
	Ended    bool `json:"ended"`
}

// programCard is the Program page's per-program view-model: image/description
// and whether it's currently on air. The full weekly schedule is only shown on
// the program detail page, not this listing.
type programCard struct {
	Program sqlc.Program
	OnAir   bool
}

// computeOnAir returns the set of program IDs currently airing "right now",
// checking both today's schedule rows and yesterday's (to catch the second half
// of an overnight-spanning slot, e.g. 23:00-01:00, whose day_of_week is
// yesterday). Shared by Home and Program.
func computeOnAir(ctx context.Context, q *sqlc.Queries) map[uint64]bool {
	now := time.Now().In(stationLoc)
	nowClock := now.Format("15:04:05")
	todayDOW := int8(now.Weekday())
	yesterdayDOW := int8((int(todayDOW) + 6) % 7)

	onAir := map[uint64]bool{}
	if rows, err := q.ListSchedulesByDay(ctx, todayDOW); err == nil {
		for _, row := range rows {
			if models.IsAiringToday(nowClock, row.StartTime, row.EndTime) {
				onAir[row.ProgramID] = true
			}
		}
	}
	if rows, err := q.ListSchedulesByDay(ctx, yesterdayDOW); err == nil {
		for _, row := range rows {
			if models.IsAiringFromYesterday(nowClock, row.StartTime, row.EndTime) {
				onAir[row.ProgramID] = true
			}
		}
	}
	return onAir
}

// todayScheduleRows returns today's full schedule (each row flagged OnAir),
// including any overnight-spanning slot that started yesterday and is still
// airing (spillover - otherwise "on air now" could point at nothing right after
// midnight). Shared by Home and Live. Returns nil if no database is configured.
func (h *Handler) todayScheduleRows(ctx context.Context) []scheduleRow {
	if h.q == nil {
		return nil
	}
	var today []scheduleRow
	now := time.Now().In(stationLoc)
	nowClock := now.Format("15:04:05")
	todayDOW := int8(now.Weekday())
	yesterdayDOW := int8((int(todayDOW) + 6) % 7)

	if rows, err := h.q.ListSchedulesByDay(ctx, yesterdayDOW); err == nil {
		for _, row := range rows {
			if models.IsAiringFromYesterday(nowClock, row.StartTime, row.EndTime) {
				today = append(today, scheduleRow{
					StartTime:    models.ClockLabel(row.StartTime),
					EndTime:      models.ClockLabel(row.EndTime),
					ProgramTitle: row.ProgramTitle,
					ProgramSlug:  row.ProgramSlug,
					ProgramHost:  models.ResolveHost(row.SlotHost, row.ProgramHost),
					ProgramImage: row.ProgramImageUrl.String,
					OnAir:        true,
					Progress:     models.Progress(nowClock, row.StartTime, row.EndTime),
				})
			}
		}
	}
	if rows, err := h.q.ListSchedulesByDay(ctx, todayDOW); err == nil {
		for _, row := range rows {
			onAir := models.IsAiringToday(nowClock, row.StartTime, row.EndTime)
			progress := 0
			if onAir {
				progress = models.Progress(nowClock, row.StartTime, row.EndTime)
			}
			today = append(today, scheduleRow{
				StartTime:    models.ClockLabel(row.StartTime),
				EndTime:      models.ClockLabel(row.EndTime),
				ProgramTitle: row.ProgramTitle,
				ProgramSlug:  row.ProgramSlug,
				ProgramHost:  models.ResolveHost(row.SlotHost, row.ProgramHost),
				ProgramImage: row.ProgramImageUrl.String,
				OnAir:        onAir,
				Progress:     progress,
				Ended:        models.HasEnded(nowClock, row.StartTime, row.EndTime),
			})
		}
	}
	return today
}

// ScheduleTodayJSON serves today's on-air/progress state as JSON, polled by
// schedule.js to keep the Live page's full schedule list highlight and
// progress bar live without a page reload. Static fields (title/time/host/
// image) aren't repeated here - the client matches this array to its rendered
// rows by index.
func (h *Handler) ScheduleTodayJSON(w http.ResponseWriter, r *http.Request) {
	rows := h.todayScheduleRows(r.Context())
	states := make([]scheduleState, len(rows))
	for i, row := range rows {
		states[i] = scheduleState{OnAir: row.OnAir, Progress: row.Progress, Ended: row.Ended}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(states)
}

// currentScheduleRow returns the currently on-air row from rows (if any) and
// its index within rows. Used by Home's single-card "On Air" view.
func currentScheduleRow(rows []scheduleRow) (*scheduleRow, int) {
	for i := range rows {
		if rows[i].OnAir {
			return &rows[i], i
		}
	}
	return nil, -1
}

// currentScheduleJSON is the /api/schedule/current response shape: the full
// currently-on-air row (or on_air:false with no other fields when nothing is
// airing), so Home's spotlight card can update itself - including swapping to
// the next program - without a page reload.
type currentScheduleJSON struct {
	OnAir    bool   `json:"on_air"`
	Title    string `json:"title,omitempty"`
	Slug     string `json:"slug,omitempty"`
	Host     string `json:"host,omitempty"`
	Image    string `json:"image,omitempty"`
	Start    string `json:"start,omitempty"`
	End      string `json:"end,omitempty"`
	Progress int    `json:"progress,omitempty"`
}

// CurrentScheduleJSON serves the currently on-air program as JSON, polled by
// now-playing-card.js to keep Home's "On Air" card live.
func (h *Handler) CurrentScheduleJSON(w http.ResponseWriter, r *http.Request) {
	row, _ := currentScheduleRow(h.todayScheduleRows(r.Context()))
	resp := currentScheduleJSON{}
	if row != nil {
		resp = currentScheduleJSON{
			OnAir:    true,
			Title:    row.ProgramTitle,
			Slug:     row.ProgramSlug,
			Host:     row.ProgramHost,
			Image:    row.ProgramImage,
			Start:    row.StartTime,
			End:      row.EndTime,
			Progress: row.Progress,
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}

// currentOnAir returns whichever program is on air right now (the zero row if
// none or no database is configured), cached briefly since it's consulted on
// every /api/nowplaying poll (every page, every 15s per visitor). Times are
// already display-formatted by todayScheduleRows.
func (h *Handler) currentOnAir(ctx context.Context) scheduleRow {
	h.onAirMu.RLock()
	fresh := time.Since(h.onAirCached) < onAirTitleTTL
	row := h.onAirRow
	h.onAirMu.RUnlock()
	if fresh {
		return row
	}

	row = scheduleRow{}
	if current, _ := currentScheduleRow(h.todayScheduleRows(ctx)); current != nil {
		row = *current
	}

	h.onAirMu.Lock()
	h.onAirRow = row
	h.onAirCached = time.Now()
	h.onAirMu.Unlock()
	return row
}

// nowPlayingJSON is the /api/nowplaying response shape: the song metadata from
// radio.Service plus, when live, the on-air program's title, time range and host
// - the floating player's and /live's two text lines fall back to the title and
// time range, then to the station's name and slogan, instead of a generic
// placeholder, and /live shows the host as its announcer badge. The program
// fields are sent even while a song is playing, since the lines fall back
// independently (a song with no artist tag still shows the program's time range).
type nowPlayingJSON struct {
	radio.NowPlaying
	ProgramTitle string `json:"program_title,omitempty"`
	ProgramStart string `json:"program_start,omitempty"`
	ProgramEnd   string `json:"program_end,omitempty"`
	ProgramHost  string `json:"program_host,omitempty"`
}

// NowPlayingJSON serves now-playing metadata as JSON, polled by the floating
// player (every page) and the /live page.
func (h *Handler) NowPlayingJSON(w http.ResponseWriter, r *http.Request) {
	np := h.radio.Current(r.Context())
	resp := nowPlayingJSON{NowPlaying: np}
	if np.Live {
		row := h.currentOnAir(r.Context())
		resp.ProgramTitle, resp.ProgramStart, resp.ProgramEnd = row.ProgramTitle, row.StartTime, row.EndTime
		resp.ProgramHost = row.ProgramHost
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}

// homeProgramPreview and homeBroadcasterPreview cap how many cards the landing
// page's programs/broadcasters strips show; the full lists live at /program and
// /broadcasters.
const (
	homeProgramPreview     = 6
	homeBroadcasterPreview = 8
)

// Home renders the landing page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	today := h.todayScheduleRows(r.Context())
	current, _ := currentScheduleRow(today)

	var hero []sqlc.NewsItem
	var programs []sqlc.Program
	var broadcasters []sqlc.Broadcaster
	if h.q != nil {
		if items, err := h.q.ListLatestPublished(r.Context(), 3); err == nil {
			hero = items
		}
		if list, err := h.q.ListActivePrograms(r.Context()); err == nil {
			programs = capSlice(list, homeProgramPreview)
		}
		if list, err := h.q.ListActiveBroadcasters(r.Context()); err == nil {
			broadcasters = capSlice(list, homeBroadcasterPreview)
		}
	}
	newsfeed := h.newsGroups(r.Context(), []string{"klikpositif", "katasumbar", "hot_release", "youtube"}, 4, true)

	h.r.Page(w, http.StatusOK, "public/home", struct {
		Base           baseData
		CurrentProgram *scheduleRow
		TodayPrograms  []scheduleRow
		TodayWeekday   string
		Hero           []sqlc.NewsItem
		Newsfeed       []newsGroup
		Programs       []sqlc.Program
		Broadcasters   []sqlc.Broadcaster
	}{
		h.base(r, "Home", "home", h.station+" — radio streaming, programs, and the latest news."),
		current, today, time.Now().In(stationLoc).Format("Monday"), hero, newsfeed, programs, broadcasters,
	})
}

// capSlice returns at most n elements of s.
func capSlice[T any](s []T, n int) []T {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// newsGroup is one source's preview list for the grouped Home/News layout.
type newsGroup struct {
	Source string
	Label  string
	Items  []newsCardItem
}

// newsCardItem wraps a NewsItem with a display-only "featured" flag for the
// news-card partial, decoupled from the item's own is_featured column so a
// page can opt out of the big-card treatment (e.g. /news) without touching
// the underlying flag.
type newsCardItem struct {
	sqlc.NewsItem
	Featured bool
}

// markFeatured wraps items for the news-card partial. When markFeatured is
// true, exactly one item is flagged Featured: the latest (first, since items
// are published_at DESC) item with is_featured=1, or if none is featured,
// the latest item overall — so a group always has one featured card.
func markFeatured(items []sqlc.NewsItem, markOne bool) []newsCardItem {
	wrapped := make([]newsCardItem, len(items))
	for i, it := range items {
		wrapped[i] = newsCardItem{NewsItem: it}
	}
	if !markOne || len(wrapped) == 0 {
		return wrapped
	}
	featuredIdx := 0
	for i, it := range items {
		if it.IsFeatured {
			featuredIdx = i
			break
		}
	}
	wrapped[featuredIdx].Featured = true
	if featuredIdx == 0 {
		return wrapped
	}
	reordered := make([]newsCardItem, 0, len(wrapped))
	reordered = append(reordered, wrapped[featuredIdx])
	for i, it := range wrapped {
		if i != featuredIdx {
			reordered = append(reordered, it)
		}
	}
	return reordered
}

// newsGroups fetches up to perGroup latest published items per source, in the
// given source order, skipping any source with zero published items. When
// markFeaturedItem is true, one item per group is flagged as featured for
// the news-card partial's big-card layout (see markFeatured).
func (h *Handler) newsGroups(ctx context.Context, sources []string, perGroup int32, markFeaturedItem bool) []newsGroup {
	var groups []newsGroup
	if h.q == nil {
		return groups
	}
	for _, src := range sources {
		items, err := h.q.ListPublishedNewsBySource(ctx, sqlc.ListPublishedNewsBySourceParams{
			Source: sqlc.NewsItemsSource(src), Limit: perGroup, Offset: 0,
		})
		if err != nil || len(items) == 0 {
			continue
		}
		groups = append(groups, newsGroup{Source: src, Label: models.SourceLabel(src), Items: markFeatured(items, markFeaturedItem)})
	}
	return groups
}

// Program renders the programs page: active program cards, each showing its own
// weekly airing schedule and an "on air now" highlight.
func (h *Handler) Program(w http.ResponseWriter, r *http.Request) {
	var cards []programCard

	if h.q != nil {
		var programs []sqlc.Program
		if list, err := h.q.ListActivePrograms(r.Context()); err == nil {
			programs = list
		}

		onAir := computeOnAir(r.Context(), h.q)

		for _, p := range programs {
			cards = append(cards, programCard{
				Program: p,
				OnAir:   onAir[p.ID],
			})
		}
	}

	h.r.Page(w, http.StatusOK, "public/program", struct {
		Base  baseData
		Cards []programCard
	}{h.base(r, "Program", "program", "Weekly schedule and list of "+h.station+"'s broadcast programs."), cards})
}

// ProgramDetail renders a single program at /program/{slug}: banner, full
// description, on-air status, and its complete weekly schedule (not the
// compact chip line used on the /program listing).
func (h *Handler) ProgramDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		h.NotFound(w, r)
		return
	}
	slug := chi.URLParam(r, "slug")
	p, err := h.q.GetProgramBySlug(r.Context(), slug)
	if err != nil || !p.IsActive {
		h.NotFound(w, r)
		return
	}
	var slots []models.ProgramSlot
	if rows, err := h.q.ListSchedulesForProgram(r.Context(), p.ID); err == nil {
		for _, row := range rows {
			slots = append(slots, models.ProgramSlot{
				Day: row.DayOfWeek, StartTime: models.ClockLabel(row.StartTime),
				EndTime: models.ClockLabel(row.EndTime),
				Host:    models.ResolveHost(row.Host, p.Host),
			})
		}
	}
	onAir := computeOnAir(r.Context(), h.q)[p.ID]
	h.r.Page(w, http.StatusOK, "public/program_detail", struct {
		Base    baseData
		Program sqlc.Program
		Groups  []models.ScheduleGroup
		OnAir   bool
	}{h.base(r, p.Title, "program", p.Title+" - "+h.station), p, models.GroupSlots(slots), onAir})
}

// Broadcasters renders the list of on-air hosts/reporters at /broadcasters.
func (h *Handler) Broadcasters(w http.ResponseWriter, r *http.Request) {
	var list []sqlc.Broadcaster
	if h.q != nil {
		list, _ = h.q.ListActiveBroadcasters(r.Context())
	}
	h.r.Page(w, http.StatusOK, "public/broadcasters", struct {
		Base         baseData
		Broadcasters []sqlc.Broadcaster
	}{h.base(r, "Broadcasters", "broadcasters", "Meet "+h.station+"'s broadcasters."), list})
}

// BroadcasterDetail renders a single broadcaster's profile at /broadcasters/{slug}.
func (h *Handler) BroadcasterDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		h.NotFound(w, r)
		return
	}
	c, err := h.q.GetActiveBroadcasterBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		h.NotFound(w, r)
		return
	}
	base := h.base(r, c.Name, "broadcasters", "Profile of "+c.Name+" - "+h.station)
	base.OGImage = c.PhotoUrl.String
	h.r.Page(w, http.StatusOK, "public/broadcaster_detail", struct {
		Base        baseData
		Broadcaster sqlc.Broadcaster
	}{base, c})
}

// Live renders the dedicated live-stream page: an SSR snapshot of now-playing
// metadata (JS polling takes over immediately after load, same as the floating
// player), the on-air program the song/singer lines and announcer badge fall
// back to, and today's full program schedule.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	today := h.todayScheduleRows(r.Context())
	current, _ := currentScheduleRow(today)

	h.r.Page(w, http.StatusOK, "public/live", struct {
		Base           baseData
		Now            radio.NowPlaying
		TodayPrograms  []scheduleRow
		TodayWeekday   string
		CurrentProgram *scheduleRow
	}{h.base(r, "Now Playing", "live", "Listen to "+h.station+"'s live broadcast."), h.radio.Current(r.Context()), today, time.Now().In(stationLoc).Format("Monday"), current})
}

// About renders the About Us page: a banner (admin-chosen image or video) plus
// three fixed text segments (profile/music/audience).
func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	var banner sqlc.AboutPageBanner
	var segments []sqlc.AboutPageSegment
	if h.q != nil {
		banner, _ = h.q.GetAboutBanner(r.Context())
		segments, _ = h.q.ListAboutSegments(r.Context())
	}

	embedURL := ""
	if banner.MediaType == sqlc.AboutPageBannerMediaTypeVideo && banner.VideoUrl.Valid {
		if u, ok := youtubeEmbedURL(banner.VideoUrl.String); ok {
			embedURL = u
		}
	}

	h.r.Page(w, http.StatusOK, "public/about", struct {
		Base     baseData
		Banner   sqlc.AboutPageBanner
		Segments []sqlc.AboutPageSegment
		EmbedURL string
	}{
		h.base(r, "About Us", "about", "Get to know "+h.station+" — our profile, our music, and who we play for."),
		banner, segments, embedURL,
	})
}

// youtubeEmbedURL converts a youtube.com/watch, youtu.be, or already-embed URL
// into a youtube.com/embed/<id> URL suitable for an <iframe> src. Returns
// ("", false) for anything it doesn't recognize (e.g. a direct video file URL),
// in which case the caller falls back to a plain <video> tag.
func youtubeEmbedURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	host := strings.TrimPrefix(u.Host, "www.")
	switch host {
	case "youtube.com", "m.youtube.com":
		if u.Path == "/watch" {
			if id := u.Query().Get("v"); id != "" {
				return "https://www.youtube.com/embed/" + id, true
			}
		}
		if strings.HasPrefix(u.Path, "/embed/") {
			return raw, true
		}
	case "youtu.be":
		if id := strings.TrimPrefix(u.Path, "/"); id != "" {
			return "https://www.youtube.com/embed/" + id, true
		}
	}
	return "", false
}

// News renders the full aggregated newsfeed: filterable by ?source= and paginated
// via ?page=.
func (h *Handler) News(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if !isValidSourceFilter(source) {
		source = ""
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := int32((page - 1) * newsPageSize)

	var groups []newsGroup
	var items []newsCardItem
	var total int64
	if source == "" {
		groups = h.newsGroups(r.Context(), []string{"klikpositif", "katasumbar", "hot_release", "youtube"}, 6, false)
	} else if h.q != nil {
		src := sqlc.NewsItemsSource(source)
		rows, _ := h.q.ListPublishedNewsBySource(r.Context(), sqlc.ListPublishedNewsBySourceParams{Source: src, Limit: newsPageSize, Offset: offset})
		items = markFeatured(rows, false)
		total, _ = h.q.CountPublishedNewsBySource(r.Context(), src)
	}
	totalPages := int((total + newsPageSize - 1) / newsPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	h.r.Page(w, http.StatusOK, "public/news", struct {
		Base         baseData
		Groups       []newsGroup
		Items        []newsCardItem
		SourceFilter string
		Page         int
		TotalPages   int
	}{h.base(r, "News", "news", "News and the latest releases about "+h.station+"."), groups, items, source, page, totalPages})
}

func isValidSourceFilter(s string) bool {
	switch s {
	case "youtube", "klikpositif", "katasumbar", "hot_release":
		return true
	default:
		return false
	}
}

// NewsDetail renders a single Hot Release item at /news/{slug}.
func (h *Handler) NewsDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		h.NotFound(w, r)
		return
	}
	slug := chi.URLParam(r, "slug")
	item, err := h.q.GetPublishedNewsItemBySlug(r.Context(), sql.NullString{String: slug, Valid: true})
	if err != nil {
		h.NotFound(w, r)
		return
	}
	base := h.base(r, item.Title, "news", item.Excerpt.String)
	base.OGImage = item.ImageUrl.String
	h.r.Page(w, http.StatusOK, "public/news_detail", struct {
		Base baseData
		Item sqlc.NewsItem
	}{base, item})
}

// NotFound renders a friendly 404.
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	base := h.base(r, "Not Found", "", "")
	base.Ads = adSlots{} // no ads on error pages
	h.r.Page(w, http.StatusNotFound, "public/notfound", struct{ Base baseData }{base})
}

// ServerError renders a friendly 500 page. Used as the recovery target when a
// handler panics (see middleware.Recover).
func (h *Handler) ServerError(w http.ResponseWriter, r *http.Request) {
	base := h.base(r, "Server Error", "", "")
	base.Ads = adSlots{} // no ads on error pages
	h.r.Page(w, http.StatusInternalServerError, "public/error", struct{ Base baseData }{base})
}

// Robots serves a minimal robots.txt: allow everything except the admin panel,
// and point crawlers at the sitemap.
func (h *Handler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /admin/\nSitemap: %s/sitemap.xml\n", h.siteURL)
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// Sitemap serves sitemap.xml: the static public pages plus every published Hot
// Release detail page. Aggregated (external) news items link out to their source
// and aren't ours to list here.
func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	set := sitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: []sitemapURL{
			{Loc: h.siteURL + "/", LastMod: today},
			{Loc: h.siteURL + "/about", LastMod: today},
			{Loc: h.siteURL + "/program", LastMod: today},
			{Loc: h.siteURL + "/live", LastMod: today},
			{Loc: h.siteURL + "/news", LastMod: today},
			{Loc: h.siteURL + "/broadcasters", LastMod: today},
		},
	}
	if h.q != nil {
		if items, err := h.q.ListHotRelease(r.Context(), 1000); err == nil {
			for _, item := range items {
				if item.Slug.String == "" {
					continue
				}
				set.URLs = append(set.URLs, sitemapURL{
					Loc:     h.siteURL + "/news/" + item.Slug.String,
					LastMod: item.UpdatedAt.Format("2006-01-02"),
				})
			}
		}
		if programs, err := h.q.ListActivePrograms(r.Context()); err == nil {
			for _, p := range programs {
				set.URLs = append(set.URLs, sitemapURL{
					Loc:     h.siteURL + "/program/" + p.Slug,
					LastMod: p.UpdatedAt.Format("2006-01-02"),
				})
			}
		}
		if broadcasters, err := h.q.ListActiveBroadcasters(r.Context()); err == nil {
			for _, c := range broadcasters {
				set.URLs = append(set.URLs, sitemapURL{
					Loc:     h.siteURL + "/broadcasters/" + c.Slug,
					LastMod: c.UpdatedAt.Format("2006-01-02"),
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(set)
}
