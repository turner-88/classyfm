# ClassyFM Radio — Second Iteration Plan

> Additive work on top of `PLAN.md` (Phases 0–6, all shipped except the deferred
> responsive/a11y QA pass). This file adds **Phases 7–9**. Each is independently
> shippable; do them in order or cherry-pick.

---

## Context

Phase 1 of `classyfm.co.id` gave the site a working floating player, program
schedule, and news aggregation. Four gaps surfaced in real use:

1. **Programs** show almost no schedule detail on the public site — cards don't
   say which day/time they air, there's no "on air now" signal anywhere despite
   `PLAN.md` originally calling for one, and program images are entered as raw
   URLs with no real upload path.
2. **Now-playing metadata is sourced from `https://classyfm.co.id/streaming/info`,
   which will be decommissioned.** The floating player also fetches cover art
   from that source today but never actually renders it (dead client-side data).
3. There's no dedicated page for the live stream — only the floating pill, which
   has no room for stream detail (bitrate, listener count, live/offline status).
4. News cards with no image render an empty gap instead of a placeholder.

This plan replaces the retiring now-playing source with the Shoutcast server's
own native endpoints (verified live, see Phase 8), adds real file uploads for
program banners, computes "on air now" including overnight-spanning shows,
supports a different host per schedule slot (a program can run Mon–Fri with
one host and Saturday with another), adds a placeholder graphic for imageless
news items, and turns the Home page hero into a full-width image slideshow.

**Two corrections to assumptions carried over from `PLAN.md`**, confirmed by
reading the current code:
- `goquery` is **not** a dependency yet — KlikPositif/KataSumbar parse RSS via
  `encoding/xml`, not HTML scraping. Phase 8 adds it fresh.
- `deploy/classyfm.service` runs with `ProtectHome=read-only` +
  `ProtectSystem=strict`. A new on-disk uploads directory under the app's own
  working directory (`/home/remorac/classyfm`) **will fail to write** in
  production unless the unit gets an explicit `ReadWritePaths=`.

---

## Phase 7 — Programs v2: richer display, on-air highlight, real image uploads

Most of this phase needs **no schema migration**:
- Overnight schedules: `program_schedules.start_time`/`end_time` have no `CHECK`
  constraint — `start >= end` is rejected only in Go
  (`internal/handlers/admin/programs.go`, `ScheduleCreate`).
- "On air now": reuse the existing `ListSchedulesByDay` query (already joins
  program title/slug/host/image_url), called twice — today's `day_of_week` and
  yesterday's — with the wraparound comparison done in Go.
- Per-program weekly chips: reuse the existing `ListAllSchedulesWithProgram`
  query, grouped by `ProgramID` in Go instead of by `DayOfWeek`.
- Image upload: still just a string written into the existing
  `programs.image_url VARCHAR(1000)` column via `CreateProgram`/`UpdateProgram`.

**One exception**: a program can already have multiple schedule slots today
(`program_schedules` is one-to-many per `program_id`), but `host` currently
lives only on `programs` — one host for the whole program. Real stations need
per-slot overrides (e.g. "Pagi Semangat" airs Mon–Fri 07:00–10:00 with host
Budi, then Sat 08:00–10:00 with host Siska). That needs a small migration —
see §7.2.

### 7.1 On-air helpers

`internal/models/schedule.go` — append:

```go
// IsAiringToday reports whether a slot whose day_of_week is *today* is airing
// right now. now/start/end are "HH:MM:SS" strings, so plain string comparison
// works. Overnight slots (start > end, e.g. 23:00–01:00) air from start until
// midnight.
func IsAiringToday(now, start, end string) bool {
	switch {
	case start < end:
		return now >= start && now < end
	case start > end:
		return now >= start
	default: // start == end: invalid, admin validation prevents this
		return false
	}
}

// IsAiringFromYesterday reports whether an overnight slot whose day_of_week is
// *yesterday* (start > end) is still airing — the second half of a show like
// 23:00 Mon–01:00 Tue, checked at 00:30 Tue where the row's day is Monday.
func IsAiringFromYesterday(now, start, end string) bool {
	return start > end && now < end
}
```

### 7.2 Schema: per-slot host override

New migration `internal/db/migrations/0004_schedule_host.up.sql`:
```sql
ALTER TABLE program_schedules ADD COLUMN host VARCHAR(255) NULL AFTER end_time;
```
`.down.sql`: `ALTER TABLE program_schedules DROP COLUMN host;`

Query changes in `internal/db/queries/programs.sql`:
- `CreateSchedule`: add `host` as a 5th param —
  `INSERT INTO program_schedules (program_id, day_of_week, start_time, end_time, host) VALUES (?, ?, ?, ?, ?)`.
- `ListSchedulesForProgram`: unchanged (`SELECT *` already picks up the new
  column).
- `ListSchedulesByDay`: add `s.host AS slot_host` to the select list (alongside
  the existing `p.host AS program_host`) so callers can resolve slot-vs-program
  host.
- `ListAllSchedulesWithProgram`: currently doesn't select `p.host` at all — add
  both `s.host AS slot_host` and `p.host AS program_host` so the Program page's
  per-card chips can resolve/display the right host per slot.

`internal/models/schedule.go` — add a resolver:
```go
// ResolveHost returns the schedule slot's own host if set, else the program's
// default host. Empty string if neither is set.
func ResolveHost(slotHost, programHost sql.NullString) string {
	if slotHost.Valid && slotHost.String != "" {
		return slotHost.String
	}
	return programHost.String
}
```

Admin: `internal/handlers/admin/programs.go`'s `ScheduleCreate` reads an
additional optional `host` form field, passed straight to `CreateSchedule`
(empty string is fine — `sql.NullString` conversion via the existing
`toNullString` helper). `web/templates/admin/programs_form.html`'s "add
schedule slot" sub-form gets a 4th input (`Host`, optional, placeholder text
like "Kosongkan untuk pakai host program"); the existing slot list table gets
a "Host" column showing the resolved value (slot host, or the program's
default in a muted/italic style, so it's clear which one is in effect).

### 7.3 Program page — replace weekly grid with enriched cards

`internal/handlers/public/public.go`:
- Add view-models: `scheduleRow` gains `ProgramID uint64` + `OnAir bool`, and
  resolves `Host` via `models.ResolveHost(row.SlotHost, row.ProgramHost)`. Add
  `programSlot{Day int8; StartTime, EndTime, Host string}` and
  `programCard{Program sqlc.Program; Slots []programSlot; OnAir bool}`.
- Add `computeOnAir(q, r) map[uint64]bool`: calls `ListSchedulesByDay` for
  today and yesterday, applies `models.IsAiringToday` /
  `models.IsAiringFromYesterday`, returns the set of currently-airing program
  IDs. Shared by `Home()` and `Program()`.
- Rewrite `Program()`: drop the `[7][]scheduleRow` grid entirely. Load
  `ListActivePrograms`, group `ListAllSchedulesWithProgram` rows by
  `ProgramID` into `programSlot`s (via `models.ClockLabel` + `ResolveHost`),
  call `computeOnAir`, assemble `[]programCard`.
- Add `models.GroupSlots(slots []programSlot) []scheduleGroup` in
  `internal/models/schedule.go`: slots are already ordered by `day_of_week`
  ascending (per `ListAllSchedulesWithProgram`'s `ORDER BY`), so a single
  left-to-right pass can merge consecutive days sharing identical
  start/end/host into one range — turning "Pagi Semangat"'s 6 raw rows (Mon,
  Tue, Wed, Thu, Fri all 07:00–10:00/Budi, plus Sat 08:00–10:00/Siska) into
  exactly 2 display chips: "Senin–Jumat 07:00–10:00 · Budi" and "Sabtu
  08:00–10:00 · Siska":
  ```go
  type scheduleGroup struct {
      FromDay, ToDay        int8
      StartTime, EndTime    string
      Host                  string
  }
  func GroupSlots(slots []programSlot) []scheduleGroup {
      var groups []scheduleGroup
      for _, s := range slots {
          if n := len(groups); n > 0 {
              last := &groups[n-1]
              if last.ToDay+1 == s.Day && last.StartTime == s.StartTime &&
                  last.EndTime == s.EndTime && last.Host == s.Host {
                  last.ToDay = s.Day
                  continue
              }
          }
          groups = append(groups, scheduleGroup{s.Day, s.Day, s.StartTime, s.EndTime, s.Host})
      }
      return groups
  }
  ```
  Note: this only merges within Sunday(0)→Saturday(6) order and does not wrap
  Sat→Sun (e.g. a program airing every single day would show as one group
  Sun–Sat only if rows start at day 0; a 7-day-identical schedule stored
  starting elsewhere would render as two adjacent groups) — an acceptable
  display-only limitation, not a correctness issue.

`web/templates/public/program.html`: delete the "Jadwal Mingguan" grid table.
Card grid per program: image or `icon-image-placeholder` fallback (`h-40
w-full`), a "Sedang Mengudara" badge + highlighted border when `.OnAir`,
title/description, and — instead of raw per-day chips — one chip per
`models.GroupSlots` group: `{{weekday $g.FromDay}}{{if ne $g.FromDay $g.ToDay}}–{{weekday $g.ToDay}}{{end}} {{$g.StartTime}}–{{$g.EndTime}}{{if $g.Host}} · {{$g.Host}}{{end}}`
(drop the program-level "host" line under the title now that host is
per-slot — show it only inside each chip, since a program with multiple
hosts across slots has no single "the" host to headline).

### 7.4 Home page — on-air highlight, including overnight spillover

`internal/handlers/public/public.go`, `Home()`: compute `spillover` (today's
`ListSchedulesByDay` call is not enough — an overnight show that started
*yesterday* has `day_of_week = yesterday`, so without also checking yesterday's
rows here, "on air now" can point at nothing right after midnight). Prepend
`spillover` (always `OnAir: true`) to today's rows before rendering. Each row's
`Host` is resolved via `models.ResolveHost(row.SlotHost, row.ProgramHost)` so
the Home "today" list shows the correct host for that specific slot (e.g.
Siska on Saturday, Budi on weekdays), not just the program's default.

`web/templates/public/home.html` "Program Hari Ini": add the same "Sedang
Mengudara" badge + subtle highlight background per row when `.OnAir`.

### 7.5 Loosen schedule validation

`internal/handlers/admin/programs.go`, `ScheduleCreate`: change the rejection
condition from `start >= end` to `start == end` only. `parseClock` (HH:MM →
HH:MM:00) needs no change; the `<input type=time>` slot form already accepts
any two times.

### 7.6 Real file upload for program banner images

**Config** (`internal/config/config.go`): add `UploadDir string`, `getenv(
"UPLOAD_DIR", "web/uploads")`. Add `UPLOAD_DIR=web/uploads` to `.env.example`.

**New file** `internal/handlers/admin/uploads.go`:
- Constants: `maxUploadBytes = 5 << 20`; `allowedImageExt` maps sniffed
  content-type (`image/jpeg`, `image/png`, `image/webp`, `image/gif`) → file
  extension.
- `func (h *Handler) saveUploadedImage(w http.ResponseWriter, r *http.Request,
  field, subdir string) (string, error)`:
  - Wrap the request with `http.MaxBytesReader` before
    `r.ParseMultipartForm(...)`.
  - `r.FormFile(field)`; if `http.ErrMissingFile`, return `("", nil)` — no
    upload submitted, caller keeps the existing image.
  - Sniff real content-type via `http.DetectContentType` on the first 512
    bytes (never trust the client-supplied filename), reject anything not in
    `allowedImageExt` or over the size limit.
  - Generate a random filename (16 bytes `crypto/rand`, mirroring
    `internal/middleware/csrf.go`'s pattern) + resolved extension;
    `os.MkdirAll` + `os.Create` under `filepath.Join(h.uploadDir, subdir,
    filename)`.
  - Return the public path, e.g. `/uploads/programs/<random>.jpg`.
  - Known limitation (comment only, not blocking): replacing/deleting a
    program's image doesn't delete the old file on disk — acceptable for v1.

**`internal/handlers/admin/admin.go`**: add `uploadDir string` to `Handler`,
thread through `New(...)` — update the one call site in `cmd/server/main.go`.

**`internal/handlers/admin/programs.go`**: turn `programFromForm` into a
`(h *Handler)` method taking `(w, r)`; it now parses multipart form data,
reads `current_image_url` (hidden field, carries the existing path forward)
as the default, and overwrites it with `h.saveUploadedImage(...)`'s result
when a new file was submitted. Update `ProgramCreate`/`ProgramUpdate` call
sites and surface upload errors by re-rendering the form with a friendly
message (same pattern as existing validation errors).

**`web/templates/admin/programs_form.html`**: add `enctype="multipart/form-
data"` to the form tag; replace the `image_url` text input with: existing-
image preview (`<img>` if `.Program.ImageUrl.Valid`), a `type="file"` input
(`accept="image/png,image/jpeg,image/webp,image/gif"`), and a hidden
`current_image_url` field carrying `.Program.ImageUrl.String` forward.

**Static serving** (`cmd/server/main.go`): `os.MkdirAll(cfg.UploadDir, 0o755)`
at boot; register `r.Handle("/uploads/*", http.StripPrefix("/uploads/",
http.FileServer(http.Dir(cfg.UploadDir))))` with a long immutable
`Cache-Control` (filenames are random-per-upload and never mutated in place —
unlike the embedded `/static/*`, this is real `http.Dir`, so normal
conditional-GET/ModTime behavior works too).

**Deploy** (required in production, not optional):
- `deploy/classyfm.service`: add `ReadWritePaths=/home/remorac/classyfm/web/uploads`
  (`ProtectHome=read-only` otherwise blocks all writes under `/home`, including
  this app's own working directory).
- `deploy/install.sh`: `mkdir -p "$APP_DIR/web/uploads/programs"` so the
  directory pre-exists with correct ownership before the service starts.
- `.gitignore`: add `/web/uploads/` (parallel to the existing `/backups/`
  entry) — uploaded content is runtime data, never committed.

### 7.7 Verification

- [ ] `go build ./...`; run the new `0004_schedule_host` migration; `make sqlc`
      regenerates `programs.sql.go` with the new `SlotHost`/`ProgramHost`
      fields (this phase is *not* a sqlc no-op, unlike originally assumed,
      because of §7.2's schema change).
- [ ] Admin: create an overnight slot (e.g. `23:00`–`01:00`) — confirm it's now
      accepted, and that `08:00`–`08:00` (exact equal) is still rejected.
- [ ] Admin: give one program two slots — Mon–Fri 07:00–10:00 with host
      "Budi" and Sat 08:00–10:00 with host "Siska" (leave Sunday unset) —
      confirm `/program`'s card shows exactly two grouped chips ("Senin–Jumat
      07:00–10:00 · Budi", "Sabtu 08:00–10:00 · Siska"), not six individual
      per-day chips.
- [ ] Confirm a slot left with no host falls back to displaying the program's
      own `host` field (not blank) both on `/program` chips and Home's "Program
      Hari Ini" list.
- [ ] With "now" inside that overnight window (test around actual midnight, or
      temporarily edit a slot to straddle current time): load `/` and
      `/program`, confirm the "Sedang Mengudara" badge appears on the right
      program both before and after midnight (i.e. both the "today" and
      "yesterday spillover" cases render correctly).
- [ ] Upload a valid image on `/admin/programs/{id}/edit` — confirm
      `programs.image_url` becomes `/uploads/programs/<random>.<ext>` and it
      renders on `/program`.
- [ ] Try an oversized file and a renamed non-image file — confirm both are
      rejected with a friendly error and no DB write happens.
- [ ] After deploying, confirm the systemd service can actually write to
      `web/uploads` (will silently fail without `ReadWritePaths=`).

---

## Phase 8 — Live streaming v2: Shoutcast-native metadata, cover art, `/live` page

**Verified live against the actual Shoutcast server** (`c4.siar.us:10340`,
Shoutcast v2.6.1.777) before designing this phase:

- `GET /stats?json=1` → clean JSON: `{"currentlisteners":14,...,"songtitle":"",
  "streamstatus":1,"bitrate":"128",...}`. **`songtitle` was empty across three
  repeated checks** — this field cannot be trusted as the metadata source, but
  `streamstatus`/`bitrate`/`currentlisteners` are reliable "detail" fields.
- `GET /played.html` → an HTML "Stream History" table. The current-song row is
  marked with a 3rd `<td>` containing "Current Song", e.g.:
  `<tr><td>22:02:52</td><td><i>Empty Title</i></td><td><b>Current Song</b></td></tr>`
  — meaning no metadata was available at that instant. Historical (2-`<td>`)
  rows are reliably `"Artist - Title"`, e.g. `"Kotak - Tanah Airku"`, `"Bruno
  Mars - Talking To The Moon"`; jingles appear the same way (`"CLASSY FM - #1
  Choice Radio Station in Padang"`), harmless to treat as `artist="CLASSY FM"`.
- `/7.html` and `/currentsong` returned nothing useful; `/stats` (no
  `?json=1`) is the XML equivalent, not needed since the JSON variant exists.

**Decisions**: drop `classyfm.co.id/streaming/info` entirely (it's being
decommissioned, no fallback). New primary text source: scrape `/played.html`.
New detail source: `/stats?json=1` (bitrate/listeners/live status — never its
`songtitle`). Cover art: free, keyless iTunes Search API, used **only** to
enrich cover art (never to override the Shoutcast-sourced artist/song text —
local/Indonesian content won't match iTunes's catalog reliably), skipped
entirely when there's no current metadata.

### 8.1 Config

`internal/config/config.go`: remove `NowPlayingURL`. Add `ShoutcastBaseURL
string`, defaulting to `deriveShoutcastBase(cfg.StreamURL)` (parses the host
out of `StreamURL`, e.g. `https://c4.siar.us:10340/stream.mp3` →
`https://c4.siar.us:10340`), overridable via `SHOUTCAST_BASE_URL` env var.
Update `.env.example` accordingly.

### 8.2 New dependency

```
go get github.com/PuerkitoBio/goquery
go mod tidy
```

### 8.3 Rewrite `internal/radio/radio.go`

New `NowPlaying` shape:

```go
type NowPlaying struct {
	Artist    string `json:"artist"`
	Song      string `json:"song"`
	HasSong   bool   `json:"has_song"`  // false = "Empty Title" state, no metadata
	CoverURL  string `json:"cover_url"` // best-effort iTunes artwork; "" if none
	Live      bool   `json:"live"`      // streamstatus == 1
	Bitrate   string `json:"bitrate"`
	Listeners int    `json:"listeners"`
}
```

`Service` keeps its existing 12s TTL-cache pattern for `Current(ctx)` /
`HandleNowPlaying`, plus a small separate iTunes cache (keyed by `"artist -
song"`, refetched only when that key changes, with a 1h safety TTL so a
repeated 15s poll for the same song doesn't re-hit iTunes's undocumented rate
limit).

- `fetchPlayedHTML(ctx)`: `GET {shoutcastBase}/played.html` (context timeout,
  `User-Agent: ClassyFM-Web/1.0`, mirrors the existing HTTP-fetch conventions
  in this package), parse with `goquery`, find the `<tr>` with 3 `<td>`s, read
  its 2nd cell. Empty or literally `"Empty Title"` → `hasSong = false`.
  Otherwise `strings.SplitN(title, " - ", 2)` → artist, song (single part
  falls back to `song = title`, empty artist).
- `fetchStats(ctx)`: `GET {shoutcastBase}/stats?json=1`, decode into a small
  struct (`currentlisteners`, `streamstatus`, `bitrate`) — **not**
  `songtitle`, confirmed unreliable above.
- `coverArt(ctx, artist, song)`: `GET https://itunes.apple.com/search?term=
  <artist> - <song>&media=music&entity=song&limit=1`, take
  `results[0].artworkUrl100`, upgrade resolution by replacing `100x100bb` with
  `600x600bb`. Skipped entirely when `!hasSong`.
- `refresh()`: fetch `played.html` and `stats?json=1` independently (one
  failing must not blank the other — merge onto last-cached-value per field on
  partial failure, same resilience spirit as today), call `coverArt` only when
  `hasSong`.
- `HandleNowPlaying` keeps its existing route/shape (`GET /api/nowplaying`,
  `Cache-Control: no-store`) — one endpoint serves both the floating player
  and the new `/live` page; the floating player simply ignores fields it
  doesn't render.

`cmd/server/main.go`: update the `radio.NewService(cfg.StreamURL, ...)` call
site to pass `cfg.ShoutcastBaseURL` instead of the removed `NowPlayingURL`.

### 8.4 New `/live` page

`internal/handlers/public/public.go`: add `Live(w, r)` — SSR an initial
`h.radio.Current(ctx)` snapshot into the page (JS polling takes over
immediately after load), same `base()`/`Page()` pattern as `Media()`.

`cmd/server/main.go`: `r.Get("/live", ph.Live)` alongside the other flat
public routes. Add `/live` to `Sitemap()` in `public.go`. Add a "Live" link in
`web/templates/partials/header.html`'s nav.

`web/templates/public/live.html` (new, follows `media.html`'s `content`-block
pattern): large cover-art box (image or `icon-image-placeholder` fallback),
now-playing text (or "Sedang mengudara" when `!HasSong`), a big play/pause
button, and a detail `<dl>` of Status (on air/offline) / Bitrate / Listener
count — all with `js-np-*` classes so the shared JS (below) can update them
alongside the floating player.

### 8.5 `radio.js` / `radio-player.html` — class-based, multi-target rendering

**Why this needs care**: `radio.js` currently guards its *entire* setup with a
single `window.__radioPlayerInit` flag, because Turbo (confirmed: real
Hotwired Turbo, `web/static/js/turbo.js`) re-executes body `<script>` tags on
every visit. That guard works today only because the floating player's button
lives inside the `data-turbo-permanent` element and is never replaced. The new
`/live` page's own big button is **not** inside that permanent element — it's
freshly rendered every Turbo visit — so it needs its own per-element bind
check, separate from the one-time audio+polling setup.

- `radio-player.html`: button gets class `js-radio-toggle` (was ID-only), add
  a `<img class="js-np-cover hidden ...">` thumbnail next to the title, title
  element gets class `js-np-title` (keep the `#radio-audio` ID — it's still
  the one singleton).
- `radio.js`: split into (a) one-time setup gated by `__radioPlayerInit`
  (audio element event listeners + the 15s `/api/nowplaying` poll loop,
  rendering into every element matching `.js-np-title` / `.js-np-cover` /
  `.js-np-bitrate` / `.js-np-listeners` / `.js-np-live-badge` via
  `querySelectorAll`, not `getElementById`) and (b) a per-visit rebind step
  that runs unconditionally, `querySelectorAll(".js-radio-toggle")`, skipping
  any button already marked `dataset.bound`.
- Cover placeholder: when `!has_song || !cover_url`, hide `.js-np-cover` and
  show a sibling `.js-np-cover-placeholder` (reusing the same
  `icon-image-placeholder` icon added in Phase 9 — reference that phase's
  icon rather than duplicating it).

### 8.6 Verification

- [ ] `go build ./...` after adding `goquery`.
- [ ] `curl localhost:8080/api/nowplaying` — confirm shape `{artist, song,
      has_song, cover_url, live, bitrate, listeners}`; confirm `has_song`
      flips to `false` when `/played.html`'s current row is `Empty Title`.
- [ ] Load `/live` directly (hard load) — big button plays/pauses and stays in
      sync with the floating player's icon state (shared `<audio>` events).
- [ ] Navigate `/` → `/live` → `/program` via in-app links (Turbo, not hard
      reloads) — confirm `/live`'s button still gets a working click listener
      on every visit, and the floating player never double-fires from
      duplicate listeners.
- [ ] Confirm cover art renders in both the floating player and `/live` when a
      real song is playing, and the placeholder shows correctly when there's
      no current metadata.
- [ ] Watch for iTunes over-calling: confirm only one iTunes request fires per
      song change (not once per 15s poll) while a song holds steady.
- [ ] `/sitemap.xml` includes `/live`.

---

## Phase 9 — News placeholder image

### 9.1 New icon

`web/templates/partials/icons.html` — add `{{define "icon-image-
placeholder"}}` (Lucide "image" glyph inline SVG); sizing is controlled by the
wrapping container at each call site, not the icon itself.

### 9.2 Template edits

- `web/templates/partials/news-card.html`: add an `{{else}}` branch to the
  existing `{{if .ImageUrl.Valid}}` — a `bg-gray-100` `h-40 w-full` box
  centering `icon-image-placeholder`.
- `web/templates/public/news_detail.html`: same idea, `{{else}}` branch using
  `aspect-video w-full rounded-xl bg-gray-100` (no fixed height exists here
  today, unlike the card context).
- `web/templates/public/home.html` Hot Release section: **do not duplicate the
  placeholder markup** — this section currently has its own inline near-copy
  of the card markup instead of calling the shared `news-card` partial like
  the Hero/Newsfeed sections above it do. Refactor it to call `{{template
  "news-card" .}}` instead, matching the rest of the page (incidental bonus:
  picks up the same "Hot Release" source label the News page already shows).

No Go changes needed: `toNullString()` (`internal/handlers/admin/
programs.go`) and the feed worker's ingestion path already normalize both
true-`NULL` and empty-string `image_url` to `sql.NullString{Valid: false}`, so
the existing `{{if .ImageUrl.Valid}}` check is already sufficient — it just
needs the `{{else}}` branch.

### 9.3 Verification

- [ ] `make css` (rebuild Tailwind; classes used here already exist elsewhere,
      no `tailwind.config.js` change expected).
- [ ] Create a Hot Release item with no image — confirm the placeholder shows
      on Home's Hero section, Home's Hot Release section, `/news`, and its own
      `/news/{slug}` detail page.
- [ ] Confirm an aggregated feed item with no image in its source content also
      shows the placeholder on `/news` and Home's Newsfeed section.
- [ ] Confirm items **with** an image are unaffected — no visual regression.

---

## Phase 10 — Home hero: full-width slideshow

Currently `internal/handlers/public/public.go`'s `Home()` already selects the
3 latest published news items (any source) into `.Hero`; `home.html` renders
them as a small 3-column grid of `news-card` partials. This phase only
changes the *rendering* — no handler/query change needed, `.Hero` is reused
as-is. Depends on Phase 9's `icon-image-placeholder` icon for Hero items with
no image.

### 10.1 Template — `web/templates/public/home.html`

Replace the current "Berita Terbaru" grid (the `.Hero` section) with a
full-bleed slideshow, breaking out of `container-page`'s max-width so the
image spans edge-to-edge:

```html
{{if .Hero}}
<section class="relative h-[420px] w-full overflow-hidden bg-gray-900 sm:h-[520px]" data-hero-slideshow>
  {{range $i, $item := .Hero}}
  <a href="{{if eq $item.Source "hot_release"}}/news/{{$item.Slug.String}}{{else}}{{$item.Url.String}}{{end}}"
     {{if ne $item.Source "hot_release"}}target="_blank" rel="noopener"{{end}}
     class="hero-slide group absolute inset-0 block h-full w-full transition-opacity duration-700 {{if ne $i 0}}pointer-events-none opacity-0{{end}}">
    {{if $item.ImageUrl.Valid}}
    <img src="{{$item.ImageUrl.String}}" alt="" class="absolute inset-0 h-full w-full object-cover">
    {{else}}
    <div class="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-brand to-brand-dark text-white/30">{{template "icon-image-placeholder"}}</div>
    {{end}}
    <div class="absolute inset-0 bg-gradient-to-t from-black/85 via-black/30 to-transparent"></div>
    <div class="container-page absolute inset-x-0 bottom-0 pb-10 pt-24 text-white">
      <p class="text-xs font-semibold uppercase tracking-wide text-white/80">{{fmtDate $item.PublishedAt}}</p>
      <h3 class="mt-2 max-w-2xl text-2xl font-extrabold leading-tight sm:text-3xl">{{$item.Title}}</h3>
      <p class="mt-0 max-h-0 max-w-xl overflow-hidden text-sm text-white/90 opacity-0 transition-all duration-300 group-hover:mt-3 group-hover:max-h-24 group-hover:opacity-100">
        {{$item.Excerpt.String}}
      </p>
      <span class="mt-3 inline-flex items-center gap-1 text-sm font-semibold opacity-0 transition-opacity duration-300 group-hover:opacity-100">Baca Selengkapnya &rarr;</span>
    </div>
  </a>
  {{end}}
  {{if gt (len .Hero) 1}}
  <div class="absolute bottom-4 right-4 z-10 flex gap-2">
    {{range $i, $item := .Hero}}
    <button type="button" class="hero-dot h-2.5 w-2.5 rounded-full bg-white/50 transition-colors {{if eq $i 0}}bg-white{{end}}" data-slide-dot="{{$i}}" aria-label="Slide"></button>
    {{end}}
  </div>
  {{end}}
</section>
{{else}}
<section class="container-page py-12">
  <h2 class="text-2xl font-bold">Berita Terbaru</h2>
  <p class="mt-1 text-gray-500">Belum ada berita.</p>
</section>
{{end}}
```

Notes:
- The outer `<section>` has no `container-page` class (so the background
  image/gradient spans the full browser width); an inner `container-page` div
  wraps only the text block so it lines up with the rest of the page's
  padding.
- "Hover to read more": title/date are always visible; the excerpt and "Baca
  Selengkapnya →" are collapsed (`max-h-0 opacity-0`) until `group-hover`
  expands them — matches the requested "hover to read more" affordance
  without a second click.
- Reuses the same source/slug/external-url branching `news-card.html` already
  implements (internal Hot Release → `/news/{slug}`; aggregated → external
  `.Url`, opened in a new tab).
- Imageless Hero items show the Phase 9 placeholder icon over a brand-colored
  gradient background instead of a broken `<img>`.
- If `.Hero` has 0 items, fall back to the original plain heading + "Belum ada
  berita." message (no empty slideshow chrome).

If `internal/render/render.go`'s template `FuncMap` has no arithmetic helper,
skip any `{{add $i 1}}`-style 1-indexing in labels (the markup above avoids it
by using a plain `aria-label="Slide"` rather than a numbered one) — not worth
adding a new template func for a cosmetic label.

### 10.2 New JS — `web/static/js/hero-slideshow.js`

Small vanilla script, added to `base.html` via
`<script src="/static/js/hero-slideshow.js" defer></script>`:

- On each execution (Turbo re-runs body `<script>`s per visit — see Phase 8),
  look up `document.querySelector("[data-hero-slideshow]")`; if absent (any
  page other than Home), return immediately — no init, no listeners.
- Collect `.hero-slide` elements and `.hero-dot` buttons (matched by index).
- `showSlide(i)`: add `pointer-events-none opacity-0` to every slide except
  index `i`; toggle each dot between `bg-white` (active) and `bg-white/50`.
- Autoplay: `setInterval(() => showSlide((i + 1) % count), 6000)`, skipped
  entirely when `matchMedia('(prefers-reduced-motion: reduce)').matches` or
  when there's only one slide.
- Dot click: `showSlide(i)` + reset the autoplay interval so a manual jump
  doesn't get immediately overridden by the timer.
- Pause on `mouseenter`/`focusin` of the section, resume on
  `mouseleave`/`focusout` (so hovering to read the revealed excerpt doesn't
  get cut short by an autoplay advance).
- Unlike the floating radio player, this section is **not**
  `data-turbo-permanent` — it's destroyed and recreated by Turbo on every
  visit to Home, so its `setInterval` is naturally cleaned up too (the old DOM
  node + closure are garbage-collected); no manual teardown/guard needed here.

### 10.3 Verification

- [ ] Load `/` — hero image touches both browser edges (no side gutters), and
      the slideshow autoplays every ~6s with dots reflecting the active slide.
- [ ] Hover a slide — excerpt and "Baca Selengkapnya →" fade/expand in, and
      autoplay pauses while hovered; moving the mouse away resumes it.
- [ ] Click a dot — jumps to that slide immediately without waiting for the
      timer.
- [ ] With exactly 1 Hero item — no dots render, no autoplay, no JS errors.
- [ ] With 0 Hero items (fresh install / everything unpublished) — the plain
      "Belum ada berita." fallback shows, no leftover slideshow chrome.
- [ ] A Hero item with no image renders the Phase 9 placeholder over the
      brand-gradient background rather than a broken image.
- [ ] Navigate `/` → another page → back to `/` several times via in-app links
      — confirm slide speed doesn't visibly accelerate (would indicate stacked
      intervals from a missing cleanup).
- [ ] Check on a narrow mobile viewport — text stays legible over the image
      (gradient contrast) and the fixed height doesn't look awkward.

---

## Config / `.env.example` net diff (all three phases)

| Var | Change |
|---|---|
| `NOWPLAYING_URL` | **removed** (`classyfm.co.id/streaming/info` is being decommissioned) |
| `SHOUTCAST_BASE_URL` | **added**, default `https://c4.siar.us:10340` (derivable from `STREAM_URL`) |
| `UPLOAD_DIR` | **added**, default `web/uploads` |

## Dependency / deploy summary

- New migration `internal/db/migrations/0004_schedule_host.{up,down}.sql`
  (adds `program_schedules.host`) — run it, then `make sqlc`.
- `go get github.com/PuerkitoBio/goquery && go mod tidy` — new dependency for
  `/played.html` scraping.
- `deploy/classyfm.service`: add `ReadWritePaths=/home/remorac/classyfm/web/uploads`
  — required, not optional, given `ProtectHome=read-only`.
- `deploy/install.sh`: `mkdir -p "$APP_DIR/web/uploads/programs"`.
- `.gitignore`: add `/web/uploads/`.

## Critical files

- `internal/db/migrations/0004_schedule_host.up.sql` + `internal/db/queries/
  programs.sql` — per-slot host schema/query change (Phase 7)
- `internal/radio/radio.go` — now-playing source rewrite (Phase 8)
- `internal/handlers/public/public.go` — `Program()`/`Home()` on-air +
  host-resolution logic, new `Live()` handler
- `internal/handlers/admin/programs.go` + new `internal/handlers/admin/
  uploads.go` — file upload, per-slot host form field
- `web/static/js/radio.js` + `web/templates/partials/radio-player.html` —
  class-based multi-target now-playing rendering
- `web/templates/public/home.html` + new `web/static/js/hero-slideshow.js` —
  full-width hero slideshow (Phase 10)
- `cmd/server/main.go` — new `/live` route, `/uploads/*` static route, updated
  `radio.NewService(...)` call
- `internal/config/config.go` — `ShoutcastBaseURL`, `UploadDir`
- `deploy/classyfm.service` — `ReadWritePaths=`
