package admin

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/models"
)

// The dashboard's two charts are built here as plain geometry - rectangles, paths
// and label positions in viewBox units - and drawn by the template as inline SVG.
//
// SVG rather than a charting library because the CSP (see middleware/security.go)
// allows neither a CDN script nor an inline style, and because a server-rendered
// chart needs no JavaScript at all: presentation attributes (x/width/fill) are not
// styles, so nothing here can trip the policy. The template stays free of
// arithmetic - every number below is already in final viewBox units.

// vizSeries are the categorical colour slots, in fixed order, assigned per entity
// (never per rank) so editing one program can't repaint the others.
//
// Validated with the dataviz skill's validate_palette.js in light mode with
// --pairs all (the conservative pairlist for a grid layout, where any two blocks
// can end up side by side): lightness band, chroma floor, CVD separation (worst
// deutan ΔE 9.2) and normal-vision separation (worst ΔE 16.3) all pass. A fifth
// hue fails the normal-vision floor whichever one is added, so slot five onward is
// vizOther grey rather than a generated colour.
//
// #1baf7a sits below 3:1 against white, which the same run flags: the relief rule
// applies, and is met because every block and every series is directly labelled and
// legended - colour is never the only channel here.
var vizSeries = []string{
	"#2a78d6", // blue
	"#eb6834", // orange
	"#1baf7a", // aqua
	"#4a3aa7", // violet
}

const (
	vizOther = "#9096ac" // gray-400: the 5th entity onward
	vizGrid  = "#edeef4" // gray-100: hairline gridlines
	vizAxis  = "#9096ac" // gray-400: axis and tick text
	vizLabel = "#4e5468" // gray-600: row labels
	vizNow   = "#e32229" // signal: reserved for live state, and "now" is live state
)

// seriesColor returns the slot for index i, folding everything past the validated
// four into one neutral.
func seriesColor(i int) string {
	if i < len(vizSeries) {
		return vizSeries[i]
	}
	return vizOther
}

// ---------------------------------------------------------------------------
// Airtime map: the broadcast week as a 7 x 24h grid.
// ---------------------------------------------------------------------------

// Geometry, in viewBox units. The map is drawn at a fixed size and scaled by CSS,
// so these are the only numbers that decide its proportions.
const (
	amWidth    = 960
	amLeft     = 46  // day-label gutter
	amRight    = 12  // right margin
	amTop      = 22  // hour-label strip
	amRowH     = 26  // one day
	amRowGap   = 6   // between days
	amBlockGap = 2   // surface gap between touching blocks
	amMinBlock = 4   // narrowest block still worth drawing
	amCharW    = 6.4 // approx. advance of the 11px label font, for the fits-inside test
)

// airtimeBlock is one scheduled slot as drawn. Title is only set when the label
// measurably fits inside the block; Tip always carries the full description.
type airtimeBlock struct {
	X, Y, W, H int
	Fill       string
	TextFill   string // ink or white, whichever clears contrast on Fill
	Title      string
	Tip        string
	Href       string
}

type airtimeRow struct {
	Label  string // "Mon"
	Y      int
	Today  bool
	Blocks []airtimeBlock
}

type vizLegendItem struct {
	Label string
	Fill  string
}

// airtimeMap is the whole chart: rows of blocks plus the chrome around them.
type airtimeMap struct {
	Width, Height int
	Rows          []airtimeRow
	HourTicks     []vizTick // 00, 04 … 24 across the top, with their gridlines
	PlotTop       int
	PlotBottom    int
	NowX          int // 0 when the marker is off (no rows)
	NowLabel      string
	Legend        []vizLegendItem
	Empty         bool
	Slots         int
}

// vizTick is one axis label with the gridline it belongs to.
type vizTick struct {
	X, Y  int
	Label string
}

// buildAirtimeMap lays out every active program's weekly slots. rows come from
// ListAllSchedulesWithProgram (already scoped to active programs); now is used only
// for the "now" marker and today's row highlight.
func buildAirtimeMap(rows []sqlc.ListAllSchedulesWithProgramRow, now time.Time) airtimeMap {
	plotW := amWidth - amLeft - amRight
	m := airtimeMap{
		Width:      amWidth,
		Height:     amTop + 7*amRowH + 6*amRowGap + 6,
		PlotTop:    amTop,
		PlotBottom: amTop + 7*amRowH + 6*amRowGap,
		Empty:      len(rows) == 0,
		Slots:      len(rows),
	}

	minuteX := func(min int) int { return amLeft + min*plotW/1440 }

	// 00 … 20 plus a closing 24, rather than wrapping back to 00: the right edge is
	// the end of the broadcast day, not the start of the next one.
	for h := 0; h <= 24; h += 4 {
		m.HourTicks = append(m.HourTicks, vizTick{X: minuteX(h * 60), Y: amTop - 8, Label: fmt.Sprintf("%02d", h)})
	}

	// Colour is per program, ordered by the program's first appearance in the
	// schedule (which is day/time order), so the palette reads left-to-right on the
	// first row rather than by database id.
	colors := map[string]string{}
	for _, row := range rows {
		if _, seen := colors[row.ProgramTitle]; !seen {
			colors[row.ProgramTitle] = seriesColor(len(colors))
			m.Legend = append(m.Legend, vizLegendItem{Label: row.ProgramTitle, Fill: colors[row.ProgramTitle]})
		}
	}

	todayDOW := int(now.Weekday())
	for day := 0; day < 7; day++ {
		m.Rows = append(m.Rows, airtimeRow{
			Label: models.WeekdayShort(day),
			Y:     amTop + day*(amRowH+amRowGap),
			Today: day == todayDOW,
		})
	}

	for _, row := range rows {
		start, end := clockMinutes(row.StartTime), clockMinutes(row.EndTime)
		host := row.BroadcasterName.String
		tip := fmt.Sprintf("%s · %s–%s", row.ProgramTitle, models.ClockLabel(row.StartTime), models.ClockLabel(row.EndTime))
		if host != "" {
			tip += " · " + host
		}
		href := "/admin/programs/" + fmt.Sprint(row.ProgramID) + "/edit"

		// An overnight slot (23:00-01:00) is stored on the day it starts and runs
		// past midnight, so it draws as two blocks - the same split the on-air
		// rules make (see models.IsAiringFromYesterday).
		day := int(row.DayOfWeek)
		if end <= start {
			m.addBlock(day, start, 1440, colors[row.ProgramTitle], row.ProgramTitle, tip, href)
			m.addBlock((day+1)%7, 0, end, colors[row.ProgramTitle], row.ProgramTitle, tip, href)
			continue
		}
		m.addBlock(day, start, end, colors[row.ProgramTitle], row.ProgramTitle, tip, href)
	}

	if !m.Empty {
		m.NowX = minuteX(now.Hour()*60 + now.Minute())
		m.NowLabel = now.Format("15:04")
	}
	return m
}

// addBlock places one rect on a day row, trimming the surface gap off its right
// edge so neighbouring blocks never touch.
func (m *airtimeMap) addBlock(day, start, end int, fill, title, tip, href string) {
	if day < 0 || day > 6 || end <= start {
		return
	}
	if fill == "" {
		fill = vizOther
	}
	plotW := amWidth - amLeft - amRight
	x := amLeft + start*plotW/1440
	w := (end-start)*plotW/1440 - amBlockGap
	if w < amMinBlock {
		w = amMinBlock
	}
	row := &m.Rows[day]
	label := ""
	// Only label inside the block when the text genuinely fits with padding on both
	// sides; otherwise the tooltip carries it. Clipped text is worse than none.
	if float64(len(title))*amCharW+16 <= float64(w) {
		label = title
	}
	row.Blocks = append(row.Blocks, airtimeBlock{
		X: x, Y: row.Y, W: w, H: amRowH, Fill: fill, TextFill: onColor(fill),
		Title: label, Tip: tip, Href: href,
	})
}

// onColor picks the label colour for text set inside a filled mark - the one place
// text may sit on a series colour - by measuring which of ink/white actually has
// more contrast against that fill rather than guessing from a lightness threshold.
// It matters: white clears 4.4:1 on the blue slot but only 2.7:1 on the aqua one,
// where ink clears 7.2:1.
func onColor(hex string) string {
	const ink = "#171a24"
	l := relLuminance(hex)
	onWhite := 1.05 / (l + 0.05)
	onInk := (l + 0.05) / (relLuminance(ink) + 0.05)
	if onInk > onWhite {
		return ink
	}
	return "#ffffff"
}

// relLuminance is the WCAG relative luminance of a #rrggbb string (0 = black).
func relLuminance(hex string) float64 {
	if len(hex) != 7 {
		return 0
	}
	channel := func(i int) float64 {
		var v int
		fmt.Sscanf(hex[i:i+2], "%02x", &v)
		c := float64(v) / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(1) + 0.7152*channel(3) + 0.0722*channel(5)
}

// clockMinutes turns a MySQL TIME ("HH:MM:SS") into minutes past midnight.
func clockMinutes(s string) int {
	if len(s) < 5 {
		return 0
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	return h*60 + m
}

// ---------------------------------------------------------------------------
// Ingest chart: aggregated news items arriving per day.
// ---------------------------------------------------------------------------

const (
	icWidth   = 720
	icHeight  = 250
	icLeft    = 34 // y-tick gutter
	icRight   = 8
	icTop     = 16
	icBottom  = 26 // x-label strip
	icBarMax  = 24 // never fill the whole band; the leftover is air
	icBarGap  = 6  // between columns
	icSegGap  = 2  // surface gap between stacked segments
	icRadius  = 4  // rounded data-end, square at the baseline
	ingestDay = 14
)

// ingestSeg is one source's slice of one day's column, already stacked.
type ingestSeg struct {
	X, Y, W, H int
	Fill       string
	Path       string // set instead of a rect when this is the rounded top segment
}

type ingestCol struct {
	Segs []ingestSeg
	Tip  string
	// The hover target is the whole band, not just the drawn bar: a one-item day is
	// a 4px sliver, and a zero day has no mark at all to hover.
	HitX, HitY, HitW, HitH int
	Label                  string // only the peak column carries a direct value label
	LabelX, LabelY         int
}

// ingestChart is the stacked-column view of the last ingestDay days.
type ingestChart struct {
	Width, Height int
	Cols          []ingestCol
	YTicks        []vizTick
	XTicks        []vizTick
	Baseline      int
	PlotLeft      int
	PlotRight     int
	Legend        []vizLegendItem
	Total         int
	Days          int
	Empty         bool
}

// ingestSources fixes both the stacking order and the colour slots, so a day where
// one source is missing doesn't recolour the others.
var ingestSources = []string{"klikpositif", "katasumbar", "youtube"}

// buildIngestChart buckets raw arrival timestamps into station-local days. Bucketing
// happens here rather than in SQL because DATE() would use the database session's
// timezone - see the comment on ListRecentNewsArrivals.
func buildIngestChart(arrivals []sqlc.ListRecentNewsArrivalsRow, now time.Time, loc *time.Location) ingestChart {
	c := ingestChart{
		Width: icWidth, Height: icHeight, Days: ingestDay,
		Baseline: icHeight - icBottom, PlotLeft: icLeft, PlotRight: icWidth - icRight,
	}

	// One bucket per day, oldest first, so quiet days keep their column.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	index := map[string]int{}
	days := make([]time.Time, ingestDay)
	counts := make([]map[string]int, ingestDay)
	for i := range days {
		days[i] = today.AddDate(0, 0, -(ingestDay - 1 - i))
		index[days[i].Format("2006-01-02")] = i
		counts[i] = map[string]int{}
	}
	for _, a := range arrivals {
		if i, ok := index[a.CreatedAt.In(loc).Format("2006-01-02")]; ok {
			counts[i][string(a.Source)]++
			c.Total++
		}
	}
	c.Empty = c.Total == 0

	// Round the top of the scale up to a clean number so the ticks read 0/10/20/30.
	peak, peakAt := 0, 0
	for i, day := range counts {
		n := 0
		for _, v := range day {
			n += v
		}
		if n > peak {
			peak, peakAt = n, i
		}
	}
	top, step := niceScale(peak)
	plotH := c.Baseline - icTop
	plotW := c.PlotRight - c.PlotLeft
	band := plotW / ingestDay
	barW := band - icBarGap
	if barW > icBarMax {
		barW = icBarMax
	}

	for v := 0; v <= top; v += step {
		y := c.Baseline - v*plotH/top
		c.YTicks = append(c.YTicks, vizTick{X: c.PlotLeft - 6, Y: y, Label: fmt.Sprint(v)})
	}

	for i, day := range counts {
		bandX := c.PlotLeft + i*band
		x := bandX + (band-barW)/2
		col := ingestCol{HitX: bandX, HitY: icTop, HitW: band, HitH: c.Baseline - icTop}
		y := c.Baseline
		total := 0
		var parts []string
		for si, source := range ingestSources {
			n := day[source]
			if n == 0 {
				continue
			}
			total += n
			parts = append(parts, fmt.Sprintf("%s %d", models.SourceLabel(source), n))
			h := n * plotH / top
			if h < icSegGap+1 {
				h = icSegGap + 1
			}
			y -= h
			col.Segs = append(col.Segs, ingestSeg{
				X: x, Y: y, W: barW, H: h - icSegGap, Fill: seriesColor(si),
			})
		}
		// The topmost segment carries the rounded data-end; everything below it stays
		// square, and the bar sits square on the baseline.
		if n := len(col.Segs); n > 0 {
			head := col.Segs[n-1]
			col.Segs[n-1].Path = roundedTopPath(head.X, head.Y, head.W, head.H, icRadius)
		}
		col.Tip = fmt.Sprintf("%s · %d %s", days[i].Format("02 Jan"), total, plural(int64(total), "item", "items"))
		if len(parts) > 0 {
			col.Tip += " — " + strings.Join(parts, ", ")
		}
		// Label selectively: the peak column only, so the number means something.
		if i == peakAt && total > 0 {
			col.Label = fmt.Sprint(total)
			col.LabelX = x + barW/2
			col.LabelY = y - 6
		}
		c.Cols = append(c.Cols, col)

		// Every other day gets an x label, plus the month on the first one.
		if i%2 == 0 || i == ingestDay-1 {
			label := days[i].Format("2")
			if i == 0 || days[i].Day() == 1 {
				label = days[i].Format("2 Jan")
			}
			c.XTicks = append(c.XTicks, vizTick{X: x + barW/2, Y: c.Baseline + 16, Label: label})
		}
	}

	seen := map[string]bool{}
	for _, day := range counts {
		for s := range day {
			seen[s] = true
		}
	}
	// Legend order follows ingestSources - the stacking order - not the data, so it
	// reads bottom-up against the bars.
	for si, source := range ingestSources {
		if seen[source] {
			c.Legend = append(c.Legend, vizLegendItem{Label: models.SourceLabel(source), Fill: seriesColor(si)})
		}
	}
	return c
}

// ---------------------------------------------------------------------------
// Listener chart: the station's peak audience per bucket, at a selectable
// grouping.
// ---------------------------------------------------------------------------

// listenerDay matches ingestDay so the daily grouping and the ingest card below it
// are the same chart of the same fortnight, read the same way.
const listenerDay = ingestDay

// listenerGroup is one grouping option. The bucket counts are chosen against the
// shared 720-unit plot: 14/24/36 columns leave bars 24/22/12px wide, still bars
// rather than hairlines. Every grouping means the same thing - the peak within the
// bucket - so switching never changes what the chart is saying.
type listenerGroup struct {
	Key     string        // ?listeners= value: "daily" | "hourly" | "5min"
	Label   string        // chip text
	Step    time.Duration // bucket width; 0 means a calendar day (see dailyBuckets)
	Buckets int           // column count
	Every   int           // label every Nth column on the x axis
	Unit    string        // reads as "Peak per <unit>"
	Window  string        // reads as "..., <window>"
}

// listenerGroups is also the order the chips render in. The first entry is the
// default.
var listenerGroups = []listenerGroup{
	{Key: "daily", Label: "Daily", Step: 0, Buckets: listenerDay, Every: 2,
		Unit: "day", Window: "last 14 days"},
	{Key: "hourly", Label: "Hourly", Step: time.Hour, Buckets: 24, Every: 3,
		Unit: "hour", Window: "last 24 hours"},
	{Key: "5min", Label: "5 minutes", Step: 5 * time.Minute, Buckets: 36, Every: 6,
		Unit: "5 minutes", Window: "last 3 hours"},
}

// listenerGroupFor resolves the query value, falling back to the default for
// anything unrecognised so a hand-edited URL can't produce a broken chart.
func listenerGroupFor(key string) listenerGroup {
	for _, g := range listenerGroups {
		if g.Key == key {
			return g
		}
	}
	return listenerGroups[0]
}

// listenerBucket is one column's data, independent of how wide the bucket is.
type listenerBucket struct {
	Start   time.Time
	Peak    int
	Samples int // 0 = never sampled, which is not the same as peaked at zero
}

// listenerGroupLink is one grouping chip.
type listenerGroupLink struct {
	Label   string
	Href    string
	Current bool
}

// listenerChart is the single-series column view of one grouping. It reuses the
// ingest chart's geometry constants and its column/segment types - one series is
// the stacked chart with exactly one segment per column - so the two charts cannot
// drift apart visually.
type listenerChart struct {
	Width, Height int
	Cols          []ingestCol
	YTicks        []vizTick
	XTicks        []vizTick
	Baseline      int
	PlotLeft      int
	PlotRight     int
	Caption       string // "Peak per hour, last 24 hours"
	EmptyNote     string
	Groups        []listenerGroupLink
	PeakAll       int // highest bucket peak in the window
	Empty         bool
}

// dailyBuckets lays the daily rollup out oldest-first over a fixed span, so a day
// the sampler didn't run keeps its (empty) place rather than letting the remaining
// days close ranks and imply continuous coverage.
func dailyBuckets(rows []sqlc.ListenerStat, g listenerGroup, now time.Time, loc *time.Location) []listenerBucket {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	buckets := make([]listenerBucket, g.Buckets)
	index := map[string]int{}
	for i := range buckets {
		buckets[i].Start = today.AddDate(0, 0, -(g.Buckets - 1 - i))
		index[buckets[i].Start.Format("2006-01-02")] = i
	}
	for _, row := range rows {
		// stat_date is a DATE already holding the station-local day the sampler
		// computed, so it is formatted as-is. Deliberately NOT converted with
		// .In(loc): that would shift it a day backwards. sampleBuckets below is the
		// opposite case - read both comments before touching either.
		i, ok := index[row.StatDate.Format("2006-01-02")]
		if !ok {
			continue
		}
		buckets[i].Peak = int(row.PeakListeners)
		buckets[i].Samples = int(row.SampleCount)
	}
	return buckets
}

// sampleBuckets folds the raw trail into fixed-width buckets, the last of which is
// the current (still filling) one.
func sampleBuckets(rows []sqlc.ListenerSample, g listenerGroup, now time.Time, loc *time.Location) []listenerBucket {
	newest := truncateTo(now.In(loc), g.Step)
	buckets := make([]listenerBucket, g.Buckets)
	index := map[string]int{}
	for i := range buckets {
		buckets[i].Start = newest.Add(-time.Duration(g.Buckets-1-i) * g.Step)
		index[buckets[i].Start.Format("2006-01-02 15:04")] = i
	}
	for _, row := range rows {
		// sampled_at is a DATETIME, which round-trips as a true instant (the DSN
		// carries loc=Local), so it must be converted into station-local time before
		// bucketing - the opposite of the DATE handling in dailyBuckets.
		i, ok := index[truncateTo(row.SampledAt.In(loc), g.Step).Format("2006-01-02 15:04")]
		if !ok {
			continue
		}
		if n := int(row.Listeners); n > buckets[i].Peak {
			buckets[i].Peak = n
		}
		buckets[i].Samples++
	}
	return buckets
}

// truncateTo rounds t down to the start of its step-wide bucket, in t's own zone.
// Built from time.Date rather than t.Truncate, which measures absolute time from
// the epoch and so only lands on a local boundary for whole-hour zone offsets.
func truncateTo(t time.Time, step time.Duration) time.Time {
	if step >= time.Hour {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	}
	m := int(step / time.Minute)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()-t.Minute()%m, 0, 0, t.Location())
}

// buildListenerChart lays out one column per bucket. The geometry is shared by
// every grouping; only the labels and tooltips differ.
func buildListenerChart(buckets []listenerBucket, g listenerGroup) listenerChart {
	c := listenerChart{
		Width: icWidth, Height: icHeight,
		Baseline: icHeight - icBottom, PlotLeft: icLeft, PlotRight: icWidth - icRight,
		Caption:   fmt.Sprintf("Peak per %s, %s", g.Unit, g.Window),
		EmptyNote: fmt.Sprintf("No listener readings in the %s — readings land every few minutes while the server is running.", g.Window),
	}
	for _, o := range listenerGroups {
		href := "/admin#listeners"
		if o.Key != listenerGroups[0].Key {
			href = "/admin?listeners=" + o.Key + "#listeners"
		}
		c.Groups = append(c.Groups, listenerGroupLink{Label: o.Label, Href: href, Current: o.Key == g.Key})
	}

	n := len(buckets)
	if n == 0 {
		c.Empty = true
		return c
	}

	// Empty means "never sampled", not "peaked at zero" - a station nobody listened
	// to still has a chart, and it should not claim to have no data.
	c.Empty = true
	peakAt := 0
	for i, b := range buckets {
		if b.Samples > 0 {
			c.Empty = false
		}
		if b.Peak > c.PeakAll {
			c.PeakAll, peakAt = b.Peak, i
		}
	}

	top, step := niceScale(c.PeakAll)
	plotH := c.Baseline - icTop
	plotW := c.PlotRight - c.PlotLeft
	band := plotW / n
	barW := band - icBarGap
	if barW > icBarMax {
		barW = icBarMax
	}
	if barW < 1 {
		barW = 1
	}

	for v := 0; v <= top; v += step {
		y := c.Baseline - v*plotH/top
		c.YTicks = append(c.YTicks, vizTick{X: c.PlotLeft - 6, Y: y, Label: fmt.Sprint(v)})
	}

	for i, b := range buckets {
		bandX := c.PlotLeft + i*band
		x := bandX + (band-barW)/2
		col := ingestCol{HitX: bandX, HitY: icTop, HitW: band, HitH: c.Baseline - icTop, Tip: listenerTip(g, b)}

		if b.Peak > 0 {
			h := b.Peak * plotH / top
			if h < icRadius {
				h = icRadius
			}
			y := c.Baseline - h
			col.Segs = append(col.Segs, ingestSeg{
				X: x, Y: y, W: barW, H: h, Fill: seriesColor(0),
				Path: roundedTopPath(x, y, barW, h, icRadius),
			})
			// Label the peak column only, so the one number on the chart is the one
			// worth reading - same rule as the ingest chart.
			if i == peakAt {
				col.Label = fmt.Sprint(b.Peak)
				col.LabelX = x + barW/2
				col.LabelY = y - 6
			}
		}
		c.Cols = append(c.Cols, col)

		// Midnight always gets a tick, whether or not it lands on the every-Nth
		// rhythm: in a 24-hour window the day boundary is the one label that stops
		// "03:00" being read as the wrong day.
		if i%g.Every == 0 || i == n-1 || (g.Step == time.Hour && b.Start.Hour() == 0) {
			c.XTicks = append(c.XTicks, vizTick{X: x + barW/2, Y: c.Baseline + 16, Label: listenerXLabel(g, b.Start, i)})
		}
	}
	return c
}

// listenerXLabel keeps each grouping's axis self-explanatory: days carry the month
// where it changes, and hours carry the date at midnight so a 24-hour span can't be
// read as the wrong day.
func listenerXLabel(g listenerGroup, t time.Time, i int) string {
	switch g.Key {
	case "daily":
		if i == 0 || t.Day() == 1 {
			return t.Format("2 Jan")
		}
		return t.Format("2")
	case "hourly":
		if t.Hour() == 0 {
			return t.Format("2 Jan")
		}
		return t.Format("15:04")
	default:
		return t.Format("15:04")
	}
}

// listenerTip spells a column out, keeping a bucket that was never sampled distinct
// from one that genuinely saw nobody listening.
func listenerTip(g listenerGroup, b listenerBucket) string {
	when := b.Start.Format("02 Jan")
	switch g.Key {
	case "hourly":
		when = b.Start.Format("02 Jan 15:04")
	case "5min":
		when = b.Start.Format("15:04")
	}
	if b.Samples == 0 {
		return when + " · not sampled"
	}
	listeners := plural(int64(b.Peak), "listener", "listeners")
	// At 5-minute granularity a bucket holds a single reading, so calling it a peak
	// would be noise - it is just the number.
	if g.Key == "5min" {
		return fmt.Sprintf("%s · %d %s", when, b.Peak, listeners)
	}
	return fmt.Sprintf("%s · peak %d %s", when, b.Peak, listeners)
}

// roundedTopPath draws a rect with only its top corners rounded - the "4px rounded
// data-end, square at the baseline" mark, which no rx attribute can express.
func roundedTopPath(x, y, w, h, r int) string {
	if r*2 > w {
		r = w / 2
	}
	if r > h {
		r = h
	}
	return fmt.Sprintf("M%d %dq0 %d %d %dh%dq%d 0 %d %dv%dh%dz",
		x, y+r, // start on the left edge, one radius below the top
		-r, r, -r, // arc up over the top-left corner
		w-2*r,   // top edge
		r, r, r, // arc down over the top-right corner
		h-r, // right edge to the baseline
		-w,  // baseline back to the start
	)
}

// niceScale picks the y-axis top and gridline step for a peak value: the smallest
// round step (1/2/5/10/…) that covers the peak in at most five gridlines, so every
// tick label is a whole round number rather than 12.5 rounded to 12.
func niceScale(peak int) (top, step int) {
	if peak <= 0 {
		return 4, 1
	}
	for _, s := range []int{1, 2, 5, 10, 20, 25, 50, 100, 250, 500, 1000, 2500} {
		step = s
		if (peak+s-1)/s <= 5 {
			break
		}
	}
	return ((peak + step - 1) / step) * step, step
}
