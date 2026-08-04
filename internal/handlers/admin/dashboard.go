package admin

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	appmw "github.com/classyfm/classyfm/internal/middleware"
	"github.com/classyfm/classyfm/internal/models"
	"github.com/classyfm/classyfm/internal/radio"
	"github.com/classyfm/classyfm/internal/schedule"
)

// recentWindow is the "new this week" window behind every stat card's sub-line
// and the dashboard's ingest pulse.
const recentWindow = 7 * 24 * time.Hour

// staleFetchFactor is how many polling intervals a feed source may go without a
// successful fetch before the dashboard calls it stale. Three rather than one so a
// single missed pass (a restart, one flaky response) isn't an alert.
const staleFetchFactor = 3

// activityRows is how many audit entries the dashboard lists. Enough to show what
// changed since the admin last looked without turning into the full trail page.
const activityRows = 6

// alertNamesShown caps how many entity names an alert spells out before falling
// back to "+N more", so one badly configured import can't produce a paragraph.
const alertNamesShown = 3

// dashboardData is the /admin landing page: what's on air, what needs attention,
// how much content exists, and what changed recently. Every field degrades to its
// zero value rather than failing the page - see Dashboard.
type dashboardData struct {
	Base       baseData
	Today      dashToday
	NowPlaying radio.NowPlaying
	Airtime    airtimeMap
	Ingest     ingestChart
	Listeners  listenerChart
	// PeakToday feeds the on-air strip, not the chart, so it is read separately:
	// it has to keep showing while the chart is in an intraday grouping, which
	// loads no daily rows at all.
	PeakToday      int
	Stats          []dashStat
	Attention      []dashAlert
	Feeds          []dashFeed
	RecentActivity []sqlc.AuditLog
}

// dashToday is the on-air strip's server-rendered state. OnAir/Next are nil when
// nothing is scheduled; admin-dashboard.js refreshes the same values in place.
type dashToday struct {
	Weekday   string
	Date      string
	Clock     string // station-local HH:MM at render time
	OnAir     *schedule.Row
	Next      *schedule.Row
	Remaining int // slots still to come today, excluding whatever is on air
}

// dashStat is one content tile. Key selects the tile's icon in the template
// (Go templates need a constant name in {{template}}, so the icon can't be data).
type dashStat struct {
	Key   string
	Label string
	Value int64
	Sub   string
	Href  string
}

// dashAlert is one row of the "Needs attention" list. Level is "warn" (something is
// broken or invisible to visitors) or "info" (worth knowing, working as configured).
type dashAlert struct {
	Level  string
	Title  string
	Detail string
	Href   string
	Action string
}

// dashFeed is one aggregation source's health line. Enabled is tracked separately
// from OK because a deliberately disabled source must not read as a failure.
type dashFeed struct {
	Label      string
	Enabled    bool
	OK         bool
	Status     string
	LastFetch  time.Time // zero = never fetched
	ItemCount  int32
	LatestItem time.Time // newest article's published_at; zero = none stored
	Href       string
}

// Dashboard renders the admin landing page. It reads a lot of small tables, and
// every read is best-effort on purpose: this is the page an admin lands on to find
// out what is wrong, so a broken query must degrade one panel, never 500 the page.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	data := dashboardData{Base: h.base(r, "Dashboard", "dashboard")}

	now := time.Now().In(schedule.Loc)
	data.Today = dashToday{
		Weekday: models.Weekday(int(now.Weekday())),
		Date:    now.Format("02 Jan 2006"),
		Clock:   now.Format("15:04"),
	}
	data.NowPlaying = h.radio.Current(r.Context())
	group := listenerGroupFor(r.URL.Query().Get("listeners"))
	// Station-local midnight, ingestDay-1 days back: a day-bucketed chart's first
	// column starts at the beginning of that day, not ingestDay*24h before this
	// instant. Shared by the ingest chart and the daily listener grouping, whose
	// windows are the same length (listenerDay == ingestDay).
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, schedule.Loc).AddDate(0, 0, -(ingestDay - 1))

	if h.q == nil {
		// Say so rather than leave the panels blank, which would read as "nothing
		// to report" when in fact nothing could be read at all. The charts are built
		// from empty input rather than left as zero structs, so they render their
		// empty state instead of a 0x0 viewBox.
		data.Attention = []dashAlert{{Level: "warn", Title: "No database configured",
			Detail: "The panel is running without a database connection, so none of the content pages are available."}}
		data.Airtime = buildAirtimeMap(nil, now)
		data.Ingest = buildIngestChart(nil, now, schedule.Loc)
		data.Listeners = h.listenerChart(r.Context(), group, now, since)
		h.r.Page(w, http.StatusOK, "admin/dashboard", data)
		return
	}
	ctx := r.Context()

	rows := schedule.TodayRows(ctx, h.q)
	data.Today.OnAir, _ = schedule.Current(rows)
	data.Today.Next = schedule.Next(rows)
	for i := range rows {
		if !rows[i].OnAir && !rows[i].Ended {
			data.Today.Remaining++
		}
	}

	news := h.newsStats(ctx)
	content := h.dashboardContent(ctx)
	data.Stats = h.dashboardStats(ctx, news, content)
	data.Feeds = h.dashboardFeeds(ctx, news)
	data.Attention = h.dashboardAlerts(ctx, news, data.Feeds, content)

	data.Airtime = buildAirtimeMap(content.Slots, now)
	arrivals, _ := h.q.ListRecentNewsArrivals(ctx, since)
	data.Ingest = buildIngestChart(arrivals, now, schedule.Loc)

	data.Listeners = h.listenerChart(ctx, group, now, since)
	data.PeakToday = h.peakToday(ctx, now)

	if u := appmw.CurrentUser(r); u != nil && u.Role == "superadmin" {
		if logs, err := h.q.ListAuditLogs(ctx, sqlc.ListAuditLogsParams{Search: "%", Limit: activityRows}); err == nil {
			data.RecentActivity = logs
		}
	}

	h.r.Page(w, http.StatusOK, "admin/dashboard", data)
}

// listenerChart loads whichever table the requested grouping needs and lays the
// chart out. The daily grouping reads the per-day rollup; the intraday groupings
// read the raw sample trail. Both reads are best-effort like everything else on
// this page, and a nil h.q still yields a correctly shaped set of empty buckets so
// the card renders its empty state rather than a 0x0 viewBox.
func (h *Handler) listenerChart(ctx context.Context, g listenerGroup, now, dailySince time.Time) listenerChart {
	var buckets []listenerBucket
	if g.Step == 0 {
		var rows []sqlc.ListenerStat
		if h.q != nil {
			rows, _ = h.q.ListListenerStats(ctx, dailySince)
		}
		buckets = dailyBuckets(rows, g, now, schedule.Loc)
	} else {
		var rows []sqlc.ListenerSample
		if h.q != nil {
			rows, _ = h.q.ListListenerSamples(ctx, now.Add(-time.Duration(g.Buckets)*g.Step))
		}
		buckets = sampleBuckets(rows, g, now, schedule.Loc)
	}
	return buildListenerChart(buckets, g)
}

// peakToday returns the highest audience recorded so far today, for the on-air
// strip. Read separately from the chart because the strip must keep showing it in
// every grouping, including the intraday ones that load no daily rows. No row yet
// is the normal state just after midnight, so any error is simply zero.
func (h *Handler) peakToday(ctx context.Context, now time.Time) int {
	if h.q == nil {
		return 0
	}
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, schedule.Loc)
	row, err := h.q.GetListenerDay(ctx, day)
	if err != nil {
		return 0
	}
	return int(row.PeakListeners)
}

// newsStats returns per-source news counts keyed by source, or an empty map on
// error. One query covers both the hot-release and aggregated tiles, the review
// queue, and each feed's real freshness.
func (h *Handler) newsStats(ctx context.Context) map[string]sqlc.NewsStatsBySourceRow {
	stats := map[string]sqlc.NewsStatsBySourceRow{}
	rows, err := h.q.NewsStatsBySource(ctx, time.Now().Add(-recentWindow))
	if err != nil {
		return stats
	}
	for _, row := range rows {
		stats[string(row.Source)] = row
	}
	return stats
}

// dashContent is the handful of whole-table reads that both the tiles and the
// alert checks want, loaded once per render rather than once per consumer. All of
// these tables hold tens of rows, so reading them whole is cheaper than the extra
// aggregate queries counting them separately would take.
type dashContent struct {
	Programs   []sqlc.Program
	Slots      []sqlc.ListAllSchedulesWithProgramRow
	Banners    []sqlc.AdBanner
	HeroActive []sqlc.HeroSlide
}

func (h *Handler) dashboardContent(ctx context.Context) dashContent {
	var c dashContent
	c.Programs, _ = h.q.ListAllPrograms(ctx)
	c.Slots, _ = h.q.ListAllSchedulesWithProgram(ctx)
	c.Banners, _ = h.q.ListAdBanners(ctx)
	c.HeroActive, _ = h.q.ListActiveHeroSlides(ctx, heroSlideProbe)
	return c
}

// dashboardStats builds the content tiles.
func (h *Handler) dashboardStats(ctx context.Context, news map[string]sqlc.NewsStatsBySourceRow, c dashContent) []dashStat {
	stats := make([]dashStat, 0, 4)

	stats = append(stats, dashStat{
		Key: "programs", Label: "Programs", Value: int64(len(c.Programs)),
		Sub: fmt.Sprintf("%d active", countActive(c.Programs, func(p sqlc.Program) bool { return p.IsActive })), Href: "/admin/programs",
	})

	broadcasters, _ := h.q.ListAllBroadcasters(ctx)
	stats = append(stats, dashStat{
		Key: "broadcasters", Label: "Broadcasters", Value: int64(len(broadcasters)),
		Sub: fmt.Sprintf("%d active", countActive(broadcasters, func(b sqlc.Broadcaster) bool { return b.IsActive })), Href: "/admin/broadcasters",
	})

	hot := news["hot_release"]
	stats = append(stats, dashStat{
		Key: "hot-release", Label: "Hot Release", Value: hot.Total,
		Sub: fmt.Sprintf("%d new this week", hot.Recent), Href: "/admin/hot-release",
	})

	var aggregated, published, recent int64
	for source, row := range news {
		if source == "hot_release" {
			continue
		}
		aggregated += row.Total
		published += row.Published
		recent += row.Recent
	}
	stats = append(stats, dashStat{
		Key: "newsfeed", Label: "Newsfeed", Value: aggregated,
		Sub: fmt.Sprintf("%d published · +%d this week", published, recent), Href: "/admin/newsfeed",
	})

	return stats
}

// heroSlideProbe bounds the active-slide read used only for a count. ListActiveHeroSlides
// takes a LIMIT because the public page shows a handful; anything past this many is
// already far more than a slideshow can use.
const heroSlideProbe = 100

// dashboardFeeds turns each aggregation source's stored status into a display line,
// pairing the worker's own report (did the fetch succeed) with what actually landed
// in news_items (when the newest article is dated).
func (h *Handler) dashboardFeeds(ctx context.Context, news map[string]sqlc.NewsStatsBySourceRow) []dashFeed {
	sources, err := h.q.ListFeedSources(ctx)
	if err != nil {
		return nil
	}
	feeds := make([]dashFeed, 0, len(sources))
	for _, s := range sources {
		f := dashFeed{
			Label:     models.SourceLabel(string(s.Source)),
			Enabled:   s.IsEnabled,
			OK:        s.LastStatus.String == "ok",
			Status:    s.LastStatus.String,
			ItemCount: s.ItemCount,
			Href:      "/admin/feed-sources",
		}
		switch {
		case !s.IsEnabled:
			f.Status = "Disabled"
		case !s.LastStatus.Valid || s.LastStatus.String == "":
			f.Status = "Not run yet"
		case f.OK:
			f.Status = "Healthy"
		}
		if s.LastFetchedAt.Valid {
			f.LastFetch = s.LastFetchedAt.Time
		}
		if row, ok := news[string(s.Source)]; ok {
			f.LatestItem = row.LatestPublishedAt
		}
		feeds = append(feeds, f)
	}
	return feeds
}

// dashboardAlerts is the triage list: everything currently wrong or worth a look,
// each with the page that fixes it. Warnings (visitors are seeing something broken
// or missing) sort above informational rows.
func (h *Handler) dashboardAlerts(ctx context.Context, news map[string]sqlc.NewsStatsBySourceRow, feeds []dashFeed, c dashContent) []dashAlert {
	var warn, info []dashAlert

	// Feed sources: failing, stale, or switched off.
	stale := staleFetchFactor * h.feedInterval
	for _, f := range feeds {
		switch {
		case !f.Enabled:
			info = append(info, dashAlert{Level: "info", Title: f.Label + " aggregation is off",
				Detail: "The source is disabled, so no new items are being pulled.", Href: f.Href, Action: "Feed sources"})
		case !f.OK:
			warn = append(warn, dashAlert{Level: "warn", Title: f.Label + " feed is failing",
				Detail: truncate(f.Status, 140), Href: f.Href, Action: "Feed sources"})
		case stale > 0 && !f.LastFetch.IsZero() && time.Since(f.LastFetch) > stale:
			warn = append(warn, dashAlert{Level: "warn", Title: f.Label + " hasn't fetched recently",
				Detail: fmt.Sprintf("Last successful run was %s ago; the worker polls every %s.", roughDuration(time.Since(f.LastFetch)), roughDuration(h.feedInterval)),
				Href:   f.Href, Action: "Feed sources"})
		}
	}

	// Aggregated items waiting to be reviewed (unpublished ones are invisible on the site).
	var pending int64
	for source, row := range news {
		if source != "hot_release" {
			pending += row.Total - row.Published
		}
	}
	if pending > 0 {
		info = append(info, dashAlert{Level: "info", Title: fmt.Sprintf("%d newsfeed %s unpublished", pending, plural(pending, "item is", "items are")),
			Detail: "Unpublished items stay hidden from the public news pages.", Href: "/admin/newsfeed", Action: "Review newsfeed"})
	}

	// Active programs missing a schedule slot or a broadcaster. Both lookup
	// queries are already scoped to active programs.
	scheduled := map[uint64]bool{}
	for _, s := range c.Slots {
		scheduled[s.ProgramID] = true
	}
	staffed := map[uint64]bool{}
	if links, err := h.q.ListBroadcasterProgramLinks(ctx); err == nil {
		for _, l := range links {
			staffed[l.ProgramID] = true
		}
	}
	var unscheduled, unstaffed []string
	for _, p := range c.Programs {
		if !p.IsActive {
			continue
		}
		if !scheduled[p.ID] {
			unscheduled = append(unscheduled, p.Title)
			continue // an unscheduled program has no slot to staff either
		}
		if !staffed[p.ID] {
			unstaffed = append(unstaffed, p.Title)
		}
	}
	if len(unscheduled) > 0 {
		warn = append(warn, dashAlert{Level: "warn", Title: fmt.Sprintf("%d active %s no airtime", len(unscheduled), plural(int64(len(unscheduled)), "program has", "programs have")),
			Detail: nameList(unscheduled) + " never appear in the weekly schedule.", Href: "/admin/programs", Action: "Programs"})
	}
	if len(unstaffed) > 0 {
		info = append(info, dashAlert{Level: "info", Title: fmt.Sprintf("%d %s no broadcaster", len(unstaffed), plural(int64(len(unstaffed)), "program has", "programs have")),
			Detail: nameList(unstaffed) + " air without an announcer credit.", Href: "/admin/programs", Action: "Programs"})
	}

	// Ad slots switched on with nothing to show.
	if slots, err := h.q.ListAdSlots(ctx); err == nil {
		for _, s := range slots {
			if !s.IsActive {
				continue
			}
			live := 0
			for _, b := range c.Banners {
				if string(b.Slot) == string(s.Slot) && b.IsActive {
					live++
				}
			}
			if live == 0 {
				detail := "The slot is enabled but has no active banner, so it renders nothing."
				if s.ShowPlaceholder {
					detail = "The slot is enabled with no active banner, so visitors see the placeholder."
				}
				info = append(info, dashAlert{Level: "info", Title: s.Label + " has no banners", Detail: detail, Href: "/admin/ads", Action: "Ads"})
			}
		}
	}

	// Hero configured to mix in image slides that don't exist.
	if settings, err := h.q.GetHeroSettings(ctx); err == nil && settings.MaxImages > 0 && len(c.HeroActive) == 0 {
		info = append(info, dashAlert{Level: "info", Title: "Hero has no active slides",
			Detail: fmt.Sprintf("Up to %d image slides are configured to show, but none are active - the hero falls back to news only.", settings.MaxImages),
			Href:   "/admin/hero", Action: "Hero"})
	}

	// Social links left blank hide their icon everywhere on the public site.
	if links, err := h.q.ListMediaLinks(ctx); err == nil {
		var blank []string
		for _, l := range links {
			if strings.TrimSpace(l.Url) == "" {
				blank = append(blank, string(l.Platform))
			}
		}
		if len(blank) > 0 {
			sort.Strings(blank)
			info = append(info, dashAlert{Level: "info", Title: fmt.Sprintf("%d social %s not set", len(blank), plural(int64(len(blank)), "link is", "links are")),
				Detail: nameList(blank) + " have no URL, so they stay hidden in the header and footer.", Href: "/admin/media", Action: "Media"})
		}
	}

	return append(warn, info...)
}

// countActive counts rows for which active reports true.
func countActive[T any](rows []T, active func(T) bool) int {
	n := 0
	for _, row := range rows {
		if active(row) {
			n++
		}
	}
	return n
}

// nameList renders up to alertNamesShown names, summarising the rest.
func nameList(names []string) string {
	if len(names) <= alertNamesShown {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:alertNamesShown], ", "), len(names)-alertNamesShown)
}

// plural picks one of two phrasings for n.
func plural(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// truncate shortens s to at most max runes, ending with an ellipsis. Feed errors are
// raw driver/HTTP strings and can run to several lines.
func truncate(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}

// roughDuration renders a duration at one unit of precision ("2 hours", "3 days"),
// which is all an "is this stale?" line needs.
func roughDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "less than a minute"
	case d < time.Hour:
		n := int(d.Minutes())
		return fmt.Sprintf("%d %s", n, plural(int64(n), "minute", "minutes"))
	case d < 24*time.Hour:
		n := int(d.Hours())
		return fmt.Sprintf("%d %s", n, plural(int64(n), "hour", "hours"))
	default:
		n := int(d.Hours() / 24)
		return fmt.Sprintf("%d %s", n, plural(int64(n), "day", "days"))
	}
}
