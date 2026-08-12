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
	"github.com/classyfm/classyfm/internal/schedule"
	"github.com/classyfm/classyfm/internal/tiktok"
)

// onAirTitleTTL bounds how long the current on-air program title is cached, so a
// burst of /api/nowplaying pollers (one per visitor, every 15s, on every page)
// doesn't hit MySQL on every request.
const onAirTitleTTL = 20 * time.Second

// stationLoc is the station's local timezone (WIB, Padang/West Sumatra), used for
// all schedule/on-air comparisons so correctness doesn't depend on the host OS's
// configured timezone. It lives in internal/schedule, which the admin dashboard
// shares.
var stationLoc = schedule.Loc

// newsPageSize is the number of items per page on the full News listing.
const newsPageSize = 12

// newsRelatedCount is how many other articles run under a news article. Three
// fills one .grid-cards row at every breakpoint.
const newsRelatedCount = 3

// homeProgramsMax caps the Program preview strip on Home. Six fills two rows of
// the three-column .grid-cards at lg; admins pick which six via sort_order.
const homeProgramsMax = 6

// Handler renders the public pages.
type Handler struct {
	r       *render.Renderer
	radio   *radio.Service
	tiktok  *tiktok.Service
	q       *sqlc.Queries // nil if no database is configured
	station string
	slogan  string
	siteURL string
	gaID    string // GA4 measurement ID; blank means no analytics tag is emitted

	// onAirMu guards a short-TTL cache of the current on-air program, populated
	// by currentOnAir. See onAirTitleTTL.
	onAirMu     sync.RWMutex
	onAirRow    scheduleRow
	onAirCached time.Time
}

// New constructs the public handler. q may be nil in early phases / when no database
// is configured, in which case data-backed sections degrade to empty rather than erroring.
func New(r *render.Renderer, radioSvc *radio.Service, tiktokSvc *tiktok.Service, q *sqlc.Queries, station, slogan, siteURL, gaID string) *Handler {
	return &Handler{r: r, radio: radioSvc, tiktok: tiktokSvc, q: q, station: station, slogan: slogan, siteURL: siteURL, gaID: gaID}
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
	TikTokLive    string // derived from TikTok, not stored; see parseTikTokProfile
	// TikTokLiveOn drives whether the card's live action is offered or greyed
	// out. It is true when TikTok reports a broadcast AND when the live state
	// can't be determined at all - see tiktokLiveState for why unknown reads as
	// on.
	TikTokLiveOn    bool
	TikTokLiveTitle string  // the live room's own title; "" unless genuinely live
	Ads             adSlots // admin-managed ad banners per placement slot (see /admin/ads)
	// GAMeasurementID is the GA4 property the layout should load analytics.js for.
	// Blank (the default when GA_MEASUREMENT_ID is unset) omits the tag entirely.
	GAMeasurementID string
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

		GAMeasurementID: h.gaID,
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
	handle, liveURL := parseTikTokProfile(b.TikTok)
	b.TikTokLive = liveURL
	b.TikTokLiveOn, b.TikTokLiveTitle = tiktokLiveState(h.tiktok.Status(r.Context(), handle))
	b.Ads = h.adsForLayout(r.Context(), adPageKey(r))
	return b
}

// parseTikTokProfile pulls the account handle and the live-room URL out of a
// stored profile URL (https://www.tiktok.com/@classyfm -> "classyfm",
// .../@classyfm/live). It returns "", "" for anything not in @handle form -
// notably the /place/<name>-<id> URL TikTok also hands out for a venue page,
// which has no live room - so the floating card can drop its live action rather
// than link somewhere that 404s. Only one TikTok URL is stored (see
// /admin/media); this saves a second admin field for two values that are
// entirely mechanical given the first.
func parseTikTokProfile(profile string) (handle, liveURL string) {
	u, err := url.Parse(profile)
	if err != nil || u.Host == "" {
		return "", ""
	}
	path := strings.Trim(u.Path, "/")
	at, rest, _ := strings.Cut(path, "/")
	if !strings.HasPrefix(at, "@") || len(at) < 2 {
		return "", ""
	}
	// Rebuilt from the parsed handle rather than appended to the input, so a
	// trailing slash, a query string, or an already-/live URL all normalize to
	// the same thing.
	if rest != "" && rest != "live" {
		return "", ""
	}
	// The API wants the bare uniqueId, the URL wants the @ form.
	return strings.TrimPrefix(at, "@"), u.Scheme + "://" + u.Host + "/" + at + "/live"
}

// tiktokLiveState turns a probe result into what the card should show. It is the
// single place the fail-open policy lives, shared by the page render and the
// polling endpoint: an unknown state (never probed yet, or the probe failed or
// was blocked) offers the live action anyway. Worst case a visitor lands on
// TikTok's own "not live" page - versus a blocked production host quietly
// greying the button out forever, with nothing on screen to say why.
func tiktokLiveState(st tiktok.Status) (on bool, title string) {
	return st.Live || !st.Known, st.Title
}

// adPageKey returns the ad-targeting key for the page being rendered, derived
// from the matched chi route pattern. Nav can't serve this purpose: /program and
// /program/{slug} share a nav value (as do the news and broadcaster pages), and
// Nav also drives nav highlighting, the header "Listen live" CTA state, and the
// floating player's expand-link suppression on /live.
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
// "On Air" card and Live's full schedule list). An alias, not a distinct type: the
// admin dashboard renders the same rows, so the definition lives in the shared
// internal/schedule package while the templates here keep the shorter local name.
type scheduleRow = schedule.Row

// scheduleState is the minimal per-row poll payload for /api/schedule/today:
// title/time/host/image are static for the day, only on-air/ended/progress change.
type scheduleState struct {
	OnAir    bool `json:"on_air"`
	Progress int  `json:"progress"`
	Ended    bool `json:"ended"`
}

// programCard is the Program page's per-program view-model: image/description,
// whether it's currently on air, and a compact airtime summary. Schedule holds
// at most cardScheduleMax merged day-ranges, with MoreSlots counting whatever
// was trimmed - a show airing at a different time every day would otherwise
// stretch its card to seven lines. The complete, untrimmed schedule is on the
// program detail page.
type programCard struct {
	Program   sqlc.Program
	OnAir     bool
	Schedule  []models.ScheduleGroup
	MoreSlots int
}

// cardScheduleMax caps how many airtime lines a program card shows.
const cardScheduleMax = 3

// dayPanel is one tab of the Program page's weekly schedule browser: every slot
// airing on that weekday, in broadcast order. All seven are rendered server-side
// and the tabs just toggle visibility, so switching days needs no round trip.
type dayPanel struct {
	Index int8
	Name  string
	Rows  []scheduleRow
}

// computeOnAir returns the set of program IDs currently airing "right now",
// checking both today's schedule rows and yesterday's (to catch the second half
// of an overnight-spanning slot, e.g. 23:00-01:00, whose day_of_week is
// yesterday). Used by ProgramDetail, which needs the flag for one program and
// has no other reason to load the schedule; Program derives the same state from
// the full slot list it already pulls for its day tabs.
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
// airing. Shared by Home and Live; see schedule.TodayRows for the rules.
func (h *Handler) todayScheduleRows(ctx context.Context) []scheduleRow {
	return schedule.TodayRows(ctx, h.q)
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
	return schedule.Current(rows)
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

// tiktokLiveJSON is the /api/tiktok/live response shape. Live is the display
// decision, not the raw probe result - the unknown-reads-as-live rule lives in
// tiktokLiveState so the browser never has to know about it.
type tiktokLiveJSON struct {
	Live  bool   `json:"live"`
	Title string `json:"title"`
}

// TikTokLiveJSON serves the TikTok card's live state, polled once a minute by
// tiktok-live.js on every page. It reads the cache only: the handle was already
// resolved and the probe already scheduled by whichever page render the poller
// came from, so this path touches neither the database nor TikTok.
func (h *Handler) TikTokLiveJSON(w http.ResponseWriter, r *http.Request) {
	on, title := tiktokLiveState(h.tiktok.Current())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(tiktokLiveJSON{Live: on, Title: title})
}

// Home renders the landing page: hero slideshow, the "On Air Now" band, a program
// preview strip, a broadcasters strip, and the newsfeed. Deliberately no schedule
// timeline; that lives on /live, and the band already answers "what's playing".
//
// The hero mixes the latest news with the image slides managed at /admin/hero;
// see hero.go for the composition rules.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	// Home shows only the one slot that's airing right now, not the day's
	// timeline (that lives on /live), but it still needs the full day to find it:
	// currentScheduleRow scans the rows rather than querying for "now".
	today := h.todayScheduleRows(r.Context())
	current, _ := currentScheduleRow(today)

	// The Program preview strip. Reuses the roster query (already ordered by
	// sort_order) capped to homeProgramsMax, so admins pick the featured six via
	// sort_order with no dedicated flag.
	var programs []sqlc.Program
	if h.q != nil {
		if list, err := h.q.ListActivePrograms(r.Context()); err == nil {
			programs = capSlice(list, homeProgramsMax)
		}
	}

	var broadcasters []sqlc.Broadcaster
	if h.q != nil {
		if list, err := h.q.ListActiveBroadcasters(r.Context()); err == nil {
			broadcasters = capSlice(list, broadcasterPreviewMax)
		}
	}

	hero := h.heroSlides(r.Context())
	newsfeed := h.newsGroups(r.Context(), []string{"klikpositif", "katasumbar", "hot_release", "youtube"}, 4, true)

	h.r.Page(w, http.StatusOK, "public/home", struct {
		Base           baseData
		CurrentProgram *scheduleRow
		Programs       []sqlc.Program
		Broadcasters   []sqlc.Broadcaster
		Hero           []heroSlide
		Newsfeed       []newsGroup
	}{
		h.base(r, "Home", "home", h.station+" — radio streaming, programs, and the latest news."),
		current, programs, broadcasters, hero, newsfeed,
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

// Program renders the programs page: a day-tabbed browser over the whole week's
// schedule, then the full roster of active programs with each card carrying its
// own airtimes.
//
// The entire page comes from two queries - the roster and one pass over every
// schedule slot - because the day panels, the per-card airtimes and the on-air
// highlight are all views of the same rows. Note the on-air state here is
// rendered once and never polled: schedule.js matches /api/schedule/today to
// rows positionally across every .js-schedule-list on a page, which cannot work
// against seven day lists, so the panels deliberately carry no js- hooks and no
// progress bars (a frozen bar reads worse than none).
func (h *Handler) Program(w http.ResponseWriter, r *http.Request) {
	var cards []programCard
	days := newDayPanels()
	todayDOW := int8(time.Now().In(stationLoc).Weekday())

	if h.q != nil {
		var programs []sqlc.Program
		if list, err := h.q.ListActivePrograms(r.Context()); err == nil {
			programs = list
		}

		onAir := map[uint64]bool{}
		slots := map[uint64][]models.ProgramSlot{}

		if rows, err := h.q.ListAllSchedulesWithProgram(r.Context()); err == nil {
			nowClock := time.Now().In(stationLoc).Format("15:04:05")
			yesterdayDOW := int8((int(todayDOW) + 6) % 7)

			for _, row := range rows {
				host := row.BroadcasterName.String

				// A slot is live either because it's today's and running, or
				// because it's yesterday's and spilling past midnight (23:00-01:00).
				airing := row.DayOfWeek == todayDOW && models.IsAiringToday(nowClock, row.StartTime, row.EndTime)
				if row.DayOfWeek == yesterdayDOW && models.IsAiringFromYesterday(nowClock, row.StartTime, row.EndTime) {
					airing = true
				}
				if airing {
					onAir[row.ProgramID] = true
				}

				if row.DayOfWeek >= 0 && row.DayOfWeek <= 6 {
					days[row.DayOfWeek].Rows = append(days[row.DayOfWeek].Rows, scheduleRow{
						StartTime:    models.ClockLabel(row.StartTime),
						EndTime:      models.ClockLabel(row.EndTime),
						ProgramTitle: row.ProgramTitle,
						ProgramSlug:  row.ProgramSlug,
						ProgramHost:  host,
						ProgramImage: row.ProgramImageUrl.String,
						// Only today's panel gets live/past styling; other days
						// are a plain listing with no "now" to measure against.
						OnAir: row.DayOfWeek == todayDOW && airing,
						Ended: row.DayOfWeek == todayDOW && models.HasEnded(nowClock, row.StartTime, row.EndTime),
					})
				}

				// Rows arrive ordered by day then start time, which is exactly
				// what GroupSlots needs to merge consecutive days.
				slots[row.ProgramID] = append(slots[row.ProgramID], models.ProgramSlot{
					Day: row.DayOfWeek, StartTime: models.ClockLabel(row.StartTime),
					EndTime: models.ClockLabel(row.EndTime), Host: host,
				})
			}
		}

		for _, p := range programs {
			groups := models.GroupSlots(slots[p.ID])
			more := 0
			if len(groups) > cardScheduleMax {
				more = len(groups) - cardScheduleMax
				groups = groups[:cardScheduleMax]
			}
			cards = append(cards, programCard{
				Program:   p,
				OnAir:     onAir[p.ID],
				Schedule:  groups,
				MoreSlots: more,
			})
		}
	}

	h.r.Page(w, http.StatusOK, "public/program", struct {
		Base       baseData
		Cards      []programCard
		Days       []dayPanel
		TodayIndex int8
		// The hero's on-air chip. Not derived from Days: those rows only carry
		// OnAir for today's day_of_week, so an overnight slot (23:00-01:00, filed
		// under yesterday) would read as nothing airing. currentOnAir handles the
		// wrap and is TTL-cached, and returns a zero row when nothing is on.
		CurrentProgram scheduleRow
	}{
		h.base(r, "Program", "program", "Weekly schedule and list of "+h.station+"'s broadcast programs."),
		cards, days, todayDOW, h.currentOnAir(r.Context()),
	})
}

// newDayPanels returns the seven empty weekday panels in Sunday-first order, so
// the template can range over a complete week even when the schedule is empty or
// no database is configured.
func newDayPanels() []dayPanel {
	names := models.Weekdays()
	panels := make([]dayPanel, 7)
	for i := range panels {
		panels[i] = dayPanel{Index: int8(i), Name: names[i]}
	}
	return panels
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
				Host:    row.BroadcasterName.String,
			})
		}
	}
	onAir := computeOnAir(r.Context(), h.q)[p.ID]

	broadcasters, _ := h.q.ListBroadcastersForProgram(r.Context(), sqlc.ListBroadcastersForProgramParams{ProgramID: p.ID})

	base := h.base(r, p.Title, "program", p.Title+" - "+h.station)
	base.OGImage = p.ImageUrl.String
	h.r.Page(w, http.StatusOK, "public/program_detail", struct {
		Base         baseData
		Program      sqlc.Program
		Groups       []models.ScheduleGroup
		OnAir        bool
		Broadcasters []sqlc.Broadcaster
	}{base, p, models.GroupSlots(slots), onAir, broadcasters})
}

// broadcasterCard is the Broadcasters page's per-person view-model: the row plus
// whether any program they present is airing right now, derived from schedule
// slots where this broadcaster is assigned.
type broadcasterCard struct {
	Broadcaster sqlc.Broadcaster
	OnAir       bool
}

// broadcasterProgram is one program on a broadcaster's profile, flagged when it
// happens to be the one airing right now.
type broadcasterProgram struct {
	Program sqlc.Program
	OnAir   bool
}

// Broadcasters renders the list of on-air hosts/reporters at /broadcasters.
func (h *Handler) Broadcasters(w http.ResponseWriter, r *http.Request) {
	var cards []broadcasterCard
	if h.q != nil {
		list, _ := h.q.ListActiveBroadcasters(r.Context())

		// One pass over every link, rather than a query per broadcaster: the
		// on-air set is small and the roster is a single page.
		onAirBroadcaster := map[uint64]bool{}
		if links, err := h.q.ListBroadcasterProgramLinks(r.Context()); err == nil {
			onAir := computeOnAir(r.Context(), h.q)
			for _, l := range links {
				if onAir[l.ProgramID] {
					onAirBroadcaster[l.BroadcasterID] = true
				}
			}
		}

		cards = make([]broadcasterCard, 0, len(list))
		for _, b := range list {
			cards = append(cards, broadcasterCard{Broadcaster: b, OnAir: onAirBroadcaster[b.ID]})
		}
	}
	h.r.Page(w, http.StatusOK, "public/broadcasters", struct {
		Base  baseData
		Cards []broadcasterCard
	}{h.base(r, "Broadcasters", "broadcasters", "Meet "+h.station+"'s broadcasters."), cards})
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

	// Only load the schedule when there is something to flag against it.
	var programs []broadcasterProgram
	if list, err := h.q.ListProgramsForBroadcaster(r.Context(), sqlc.ListProgramsForBroadcasterParams{BroadcasterID: c.ID}); err == nil && len(list) > 0 {
		onAir := computeOnAir(r.Context(), h.q)
		for _, p := range list {
			programs = append(programs, broadcasterProgram{Program: p, OnAir: onAir[p.ID]})
		}
	}

	base := h.base(r, c.Name, "broadcasters", "Profile of "+c.Name+" - "+h.station)
	base.OGImage = c.PhotoUrl.String
	h.r.Page(w, http.StatusOK, "public/broadcaster_detail", struct {
		Base        baseData
		Broadcaster sqlc.Broadcaster
		Programs    []broadcasterProgram
	}{base, c, programs})
}

// Live renders the dedicated live-stream page: an SSR snapshot of now-playing
// metadata (JS polling takes over immediately after load, same as the floating
// player), the on-air program the song/singer lines and announcer badge fall
// back to, and today's full program schedule.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	today := h.todayScheduleRows(r.Context())
	current, _ := currentScheduleRow(today)

	var broadcasters []sqlc.Broadcaster
	if h.q != nil {
		if list, err := h.q.ListActiveBroadcasters(r.Context()); err == nil {
			broadcasters = capSlice(list, broadcasterPreviewMax)
		}
	}

	h.r.Page(w, http.StatusOK, "public/live", struct {
		Base           baseData
		Now            radio.NowPlaying
		TodayPrograms  []scheduleRow
		TodayWeekday   string
		CurrentProgram *scheduleRow
		Broadcasters   []sqlc.Broadcaster
	}{h.base(r, "Now Playing", "live", "Listen to "+h.station+"'s live broadcast."), h.radio.Current(r.Context()), today, time.Now().In(stationLoc).Format("Monday"), current, broadcasters})
}

// broadcasterPreviewMax caps a broadcasters strip shown outside /broadcasters
// itself, which lists the whole roster. /about, / (home), and /live carry one.
const broadcasterPreviewMax = 8

// About renders the About Us page: a banner (admin-chosen image or video) and
// three fixed text segments (profile/music/audience), closing with a broadcasters
// preview so the page leads somewhere.
func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	var banner sqlc.AboutPageBanner
	var segments []sqlc.AboutPageSegment
	var broadcasters []sqlc.Broadcaster
	if h.q != nil {
		banner, _ = h.q.GetAboutBanner(r.Context())
		segments, _ = h.q.ListAboutSegments(r.Context())
		if list, err := h.q.ListActiveBroadcasters(r.Context()); err == nil {
			broadcasters = capSlice(list, broadcasterPreviewMax)
		}
	}

	embedURL := ""
	if banner.MediaType == sqlc.AboutPageBannerMediaTypeVideo && banner.VideoUrl.Valid {
		if u, ok := youtubeEmbedURL(banner.VideoUrl.String); ok {
			embedURL = u
		}
	}

	h.r.Page(w, http.StatusOK, "public/about", struct {
		Base         baseData
		Banner       sqlc.AboutPageBanner
		Segments     []sqlc.AboutPageSegment
		EmbedURL     string
		Broadcasters []sqlc.Broadcaster
	}{
		h.base(r, "About Us", "about", "Get to know "+h.station+" — our profile, our music, and who we play for."),
		banner, segments, embedURL, broadcasters,
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
	var lead *newsCardItem
	var items []newsCardItem
	var total int64
	if source == "" {
		groups = h.newsGroups(r.Context(), []string{"klikpositif", "katasumbar", "hot_release", "youtube"}, 6, false)
		groups, lead = popLead(groups)
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
		Lead         *newsCardItem
		Groups       []newsGroup
		Items        []newsCardItem
		SourceFilter string
		Page         int
		TotalPages   int
	}{h.base(r, "News", "news", "News and the latest releases about "+h.station+"."), lead, groups, items, source, page, totalPages})
}

// popLead pulls the newest Hot Release item out of the grouped listing to run as
// the page's lead story, returning the groups with that item removed so it does
// not appear twice on the page. A group left empty by the move is dropped along
// with it, since its banner and "See more" link would head an empty grid.
//
// The lead is deliberately restricted to hot_release - the station's own
// reporting - rather than "whatever is newest": every other source links
// off-site, and the biggest click target on the news page should not leave it.
// With no Hot Release items at all the page simply opens on the rails.
func popLead(groups []newsGroup) ([]newsGroup, *newsCardItem) {
	for i, g := range groups {
		if g.Source != "hot_release" || len(g.Items) == 0 {
			continue
		}
		// Items are published_at DESC, so the first one is the newest.
		lead := g.Items[0]
		lead.Featured = true // renders through news-card's big-card branch
		groups[i].Items = g.Items[1:]
		if len(groups[i].Items) == 0 {
			groups = append(groups[:i], groups[i+1:]...)
		}
		return groups, &lead
	}
	return groups, nil
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

	// The body is split here (not in the template) so the mid-article gallery can
	// be injected at the exact midpoint. GalleryIndex is how many paragraphs run
	// before it; for a short or empty body that lands at/after the end, so the
	// template renders the gallery below the body instead.
	var paragraphs []string
	if item.Content.Valid && strings.TrimSpace(item.Content.String) != "" {
		paragraphs = render.SplitParagraphs(item.Content.String)
	}
	h.r.Page(w, http.StatusOK, "public/news_detail", struct {
		Base         baseData
		Item         sqlc.NewsItem
		Paragraphs   []string
		GalleryIndex int
		MiddleImages []string
		Related      []newsCardItem
	}{base, item, paragraphs, len(paragraphs) / 2, decodeMiddleImages(item.MiddleImages), h.relatedNews(r.Context(), item.ID)})
}

// decodeMiddleImages unpacks the JSON array stored in news_items.middle_images
// into the ordered URL list the gallery renders. A NULL/blank column or malformed
// JSON yields nil - a broken value must not take the article page down.
func decodeMiddleImages(s sql.NullString) []string {
	if !s.Valid || strings.TrimSpace(s.String) == "" {
		return nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(s.String), &urls); err != nil {
		return nil
	}
	return urls
}

// relatedNews picks up to newsRelatedCount other Hot Release articles to run
// under an article. Only hot_release qualifies: every other source links
// off-site, and "keep reading" should keep the reader here.
//
// Fetches one more than it needs so removing the current article still leaves a
// full set, and degrades to nothing on a query error - a failed sidebar must not
// take the article down with it.
func (h *Handler) relatedNews(ctx context.Context, excludeID uint64) []newsCardItem {
	if h.q == nil {
		return nil
	}
	rows, err := h.q.ListHotRelease(ctx, newsRelatedCount+1)
	if err != nil {
		return nil
	}
	kept := make([]sqlc.NewsItem, 0, newsRelatedCount)
	for _, it := range rows {
		if it.ID == excludeID {
			continue
		}
		if len(kept) == newsRelatedCount {
			break
		}
		kept = append(kept, it)
	}
	return markFeatured(kept, false)
}

// PrivacyPolicy renders the privacy policy page, its title/intro/body (Markdown)
// loaded from the legal_pages table and admin-editable at /admin/legal.
func (h *Handler) PrivacyPolicy(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, sqlc.LegalPagesSlugPrivacy, "Kebijakan Privasi", "Kebijakan privasi "+h.station+".")
}

// TermsAndConditions renders the terms & conditions page, backed by the same
// legal_pages table as the privacy page.
func (h *Handler) TermsAndConditions(w http.ResponseWriter, r *http.Request) {
	h.legalPage(w, r, sqlc.LegalPagesSlugTerms, "Syarat dan Ketentuan", "Syarat dan ketentuan "+h.station+".")
}

// legalPage renders a single DB-backed legal page. The fallbackTitle/metaDesc keep
// the page usable in degraded mode (no DB pool, or the row missing) — the hero
// title falls back and the body is simply empty, matching how About tolerates a
// nil h.q rather than erroring.
func (h *Handler) legalPage(w http.ResponseWriter, r *http.Request, slug sqlc.LegalPagesSlug, fallbackTitle, metaDesc string) {
	var page sqlc.LegalPage
	if h.q != nil {
		page, _ = h.q.GetLegalPage(r.Context(), slug)
	}
	title := page.Title
	if title == "" {
		title = fallbackTitle
	}
	h.r.Page(w, http.StatusOK, "public/legal_page", struct {
		Base  baseData
		Title string
		Intro string
		Body  string
	}{h.base(r, title, "", metaDesc), title, page.Intro, page.Body})
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
