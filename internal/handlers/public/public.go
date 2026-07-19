// Package public holds HTTP handlers for the public-facing pages.
package public

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
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
	siteURL string

	// onAirMu guards a short-TTL cache of the current on-air program title,
	// populated by currentProgramTitle. See onAirTitleTTL.
	onAirMu     sync.RWMutex
	onAirTitle  string
	onAirCached time.Time
}

// New constructs the public handler. q may be nil in early phases / when no database
// is configured, in which case data-backed sections degrade to empty rather than erroring.
func New(r *render.Renderer, radioSvc *radio.Service, q *sqlc.Queries, station, siteURL string) *Handler {
	return &Handler{r: r, radio: radioSvc, q: q, station: station, siteURL: siteURL}
}

// baseData is the common view-model every page embeds (used by the layout, player,
// and SEO meta tags).
type baseData struct {
	Title        string
	Nav          string // active nav key: home|program|media|news
	StationName  string
	StreamURL    string
	Description  string
	CanonicalURL string
	OGImage      string // absolute URL; blank suppresses the og:image/twitter:image tags
	Instagram    string // social links for the footer (admin-managed, see /admin/media); blank hides the icon
	Facebook     string
	X            string
	YouTube      string
}

// base builds the common view-model. description should be a one-sentence summary
// of the page for the meta description / Open Graph tags; the canonical URL is
// derived from the request path (query strings such as pagination/filters are
// intentionally excluded so paginated/filtered variants canonicalize to the plain page).
func (h *Handler) base(r *http.Request, title, nav, description string) baseData {
	b := baseData{
		Title:        title,
		Nav:          nav,
		StationName:  h.station,
		StreamURL:    h.radio.StreamURL(),
		Description:  description,
		CanonicalURL: h.siteURL + r.URL.Path,
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
				}
			}
		}
	}
	return b
}

// scheduleRow is the view-model for one weekly schedule slot (used on Home's
// "Program Hari Ini" list).
type scheduleRow struct {
	StartTime    string
	EndTime      string
	ProgramTitle string
	ProgramSlug  string
	ProgramHost  string
	ProgramImage string
	OnAir        bool
	Progress     int // 0-100, only meaningful when OnAir
}

// scheduleState is the minimal per-row poll payload for /api/schedule/today:
// title/time/host/image are static for the day, only on-air/progress change.
type scheduleState struct {
	OnAir    bool `json:"on_air"`
	Progress int  `json:"progress"`
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
			})
		}
	}
	return today
}

// ScheduleTodayJSON serves today's on-air/progress state as JSON, polled by
// schedule.js to keep the "Program Hari Ini" list's highlight and progress bar
// live without a page reload. Static fields (title/time/host/image) aren't
// repeated here - the client matches this array to its rendered rows by index.
func (h *Handler) ScheduleTodayJSON(w http.ResponseWriter, r *http.Request) {
	rows := h.todayScheduleRows(r.Context())
	states := make([]scheduleState, len(rows))
	for i, row := range rows {
		states[i] = scheduleState{OnAir: row.OnAir, Progress: row.Progress}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(states)
}

// currentProgramTitle returns the title of whichever program is on air right
// now (empty if none or no database is configured), cached briefly since it's
// consulted on every /api/nowplaying poll (every page, every 15s per visitor).
func (h *Handler) currentProgramTitle(ctx context.Context) string {
	h.onAirMu.RLock()
	fresh := time.Since(h.onAirCached) < onAirTitleTTL
	title := h.onAirTitle
	h.onAirMu.RUnlock()
	if fresh {
		return title
	}

	title = ""
	for _, row := range h.todayScheduleRows(ctx) {
		if row.OnAir {
			title = row.ProgramTitle
			break
		}
	}

	h.onAirMu.Lock()
	h.onAirTitle, h.onAirCached = title, time.Now()
	h.onAirMu.Unlock()
	return title
}

// nowPlayingJSON is the /api/nowplaying response shape: the song metadata from
// radio.Service plus, when live with no song metadata, the on-air program's
// title - the floating player's label falls back to this, then to the station
// name, instead of a generic placeholder.
type nowPlayingJSON struct {
	radio.NowPlaying
	ProgramTitle string `json:"program_title,omitempty"`
}

// NowPlayingJSON serves now-playing metadata as JSON, polled by the floating
// player (every page) and the /live page.
func (h *Handler) NowPlayingJSON(w http.ResponseWriter, r *http.Request) {
	np := h.radio.Current(r.Context())
	resp := nowPlayingJSON{NowPlaying: np}
	if np.Live && !np.HasSong {
		resp.ProgramTitle = h.currentProgramTitle(r.Context())
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}

// Home renders the landing page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	today := h.todayScheduleRows(r.Context())
	var hero []sqlc.NewsItem
	if h.q != nil {
		if items, err := h.q.ListLatestPublished(r.Context(), 3); err == nil {
			hero = items
		}
	}
	newsfeed := h.newsGroups(r.Context(), []string{"hot_release", "youtube", "klikpositif", "katasumbar"}, 3)

	h.r.Page(w, http.StatusOK, "public/home", struct {
		Base          baseData
		TodayPrograms []scheduleRow
		TodayWeekday  string
		Hero          []sqlc.NewsItem
		Newsfeed      []newsGroup
	}{h.base(r, "Beranda", "home", h.station+" — radio streaming, program, dan berita terbaru."), today, time.Now().In(stationLoc).Format("Monday"), hero, newsfeed})
}

// newsGroup is one source's preview list for the grouped Home/News layout.
type newsGroup struct {
	Source string
	Label  string
	Items  []sqlc.NewsItem
}

// newsGroups fetches up to perGroup latest published items per source, in the
// given source order, skipping any source with zero published items.
func (h *Handler) newsGroups(ctx context.Context, sources []string, perGroup int32) []newsGroup {
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
		groups = append(groups, newsGroup{Source: src, Label: models.SourceLabel(src), Items: items})
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
	}{h.base(r, "Program", "program", "Jadwal mingguan dan daftar program siaran "+h.station+"."), cards})
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

// Classiers renders the list of on-air hosts/reporters at /classiers.
func (h *Handler) Classiers(w http.ResponseWriter, r *http.Request) {
	var list []sqlc.Classier
	if h.q != nil {
		list, _ = h.q.ListActiveClassiers(r.Context())
	}
	h.r.Page(w, http.StatusOK, "public/classiers", struct {
		Base      baseData
		Classiers []sqlc.Classier
	}{h.base(r, "Classiers", "classiers", "Kenali penyiar "+h.station+"."), list})
}

// ClassierDetail renders a single classier's profile at /classiers/{slug}.
func (h *Handler) ClassierDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		h.NotFound(w, r)
		return
	}
	c, err := h.q.GetActiveClassierBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		h.NotFound(w, r)
		return
	}
	base := h.base(r, c.Name, "classiers", "Profil "+c.Name+" - "+h.station)
	base.OGImage = c.PhotoUrl.String
	h.r.Page(w, http.StatusOK, "public/classier_detail", struct {
		Base     baseData
		Classier sqlc.Classier
	}{base, c})
}

// Live renders the dedicated live-stream page: an SSR snapshot of now-playing
// metadata (JS polling takes over immediately after load, same as the floating
// player) plus stream detail (status/bitrate/listener count) that the floating
// pill has no room for, and today's full program schedule.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	today := h.todayScheduleRows(r.Context())
	currentProgramTitle := ""
	for _, row := range today {
		if row.OnAir {
			currentProgramTitle = row.ProgramTitle
			break
		}
	}

	h.r.Page(w, http.StatusOK, "public/live", struct {
		Base                baseData
		Now                 radio.NowPlaying
		TodayPrograms       []scheduleRow
		TodayWeekday        string
		CurrentProgramTitle string
	}{h.base(r, "Live", "live", "Dengarkan siaran langsung "+h.station+"."), h.radio.Current(r.Context()), today, time.Now().In(stationLoc).Format("Monday"), currentProgramTitle})
}

// Media renders the social media page: one badge per platform, linking to
// ClassyFM's account page. A blank URL means that badge is hidden.
func (h *Handler) Media(w http.ResponseWriter, r *http.Request) {
	byPlatform := map[sqlc.MediaLinksPlatform]string{}
	if h.q != nil {
		if links, err := h.q.ListMediaLinks(r.Context()); err == nil {
			for _, l := range links {
				byPlatform[l.Platform] = l.Url
			}
		}
	}

	h.r.Page(w, http.StatusOK, "public/media", struct {
		Base      baseData
		Instagram string
		Facebook  string
		X         string
		YouTube   string
	}{
		Base:      h.base(r, "Media", "media", "Ikuti "+h.station+" di Instagram, Facebook, X, dan YouTube."),
		Instagram: byPlatform[sqlc.MediaLinksPlatformInstagram],
		Facebook:  byPlatform[sqlc.MediaLinksPlatformFacebook],
		X:         byPlatform[sqlc.MediaLinksPlatformX],
		YouTube:   byPlatform[sqlc.MediaLinksPlatformYoutube],
	})
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
	var items []sqlc.NewsItem
	var total int64
	if source == "" {
		groups = h.newsGroups(r.Context(), []string{"hot_release", "youtube", "klikpositif", "katasumbar"}, 6)
	} else if h.q != nil {
		src := sqlc.NewsItemsSource(source)
		items, _ = h.q.ListPublishedNewsBySource(r.Context(), sqlc.ListPublishedNewsBySourceParams{Source: src, Limit: newsPageSize, Offset: offset})
		total, _ = h.q.CountPublishedNewsBySource(r.Context(), src)
	}
	totalPages := int((total + newsPageSize - 1) / newsPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	h.r.Page(w, http.StatusOK, "public/news", struct {
		Base         baseData
		Groups       []newsGroup
		Items        []sqlc.NewsItem
		SourceFilter string
		Page         int
		TotalPages   int
	}{h.base(r, "Berita", "news", "Berita dan rilis terbaru seputar "+h.station+"."), groups, items, source, page, totalPages})
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
	h.r.Page(w, http.StatusNotFound, "public/notfound", struct{ Base baseData }{h.base(r, "Tidak Ditemukan", "", "")})
}

// ServerError renders a friendly 500 page. Used as the recovery target when a
// handler panics (see middleware.Recover).
func (h *Handler) ServerError(w http.ResponseWriter, r *http.Request) {
	h.r.Page(w, http.StatusInternalServerError, "public/error", struct{ Base baseData }{h.base(r, "Kesalahan Server", "", "")})
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
			{Loc: h.siteURL + "/program", LastMod: today},
			{Loc: h.siteURL + "/live", LastMod: today},
			{Loc: h.siteURL + "/media", LastMod: today},
			{Loc: h.siteURL + "/news", LastMod: today},
			{Loc: h.siteURL + "/classiers", LastMod: today},
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
		if classiers, err := h.q.ListActiveClassiers(r.Context()); err == nil {
			for _, c := range classiers {
				set.URLs = append(set.URLs, sitemapURL{
					Loc:     h.siteURL + "/classiers/" + c.Slug,
					LastMod: c.UpdatedAt.Format("2006-01-02"),
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(set)
}
