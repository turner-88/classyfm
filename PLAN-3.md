# ClassyFM Radio — Third Iteration Plan

> Additive work on top of `PLAN.md` (Phases 0–6) and `PLAN-2.md` (Phases 7–10, all
> confirmed present in the current codebase: per-slot hosts, Shoutcast-native
> now-playing with live/offline detection, `/live` page, image uploads, news
> placeholder icon, hero slideshow). This file adds **Phases 11–16**. Each is
> independently shippable; do them in order or cherry-pick.

---

## Context

This iteration is a mix of rebrand, content restructuring, and two net-new
public features, driven by six concrete asks:

1. **Rebrand** — swap the red primary color for navy, and replace the
   hand-drawn inline-SVG logo/favicon with ClassyFM's real logo, sourced from
   the station's still-live legacy site (`classyfm.co.id`).
2. **Home page** reorder + a new welcome section.
3. **News grouping** — both Home and `/news` currently show one flat,
   date-sorted list mixing all sources; they need to group by source (banner
   heading per group), latest-first within each group.
4. **Program detail page** — programs currently only render as cards on one
   listing page (`/program`); no `/program/{slug}` exists yet.
5. **Classiers (hosts) page** — ClassyFM's on-air talent has no representation
   in this codebase at all today (`host` is just a free-text string on
   `programs`/`program_schedules`). The legacy site has a real "Classiers"
   directory (`classyfm.co.id/crew`) with 8 real people, photos, and bios —
   this becomes a new first-class entity with public list + detail pages,
   admin CRUD, and seed data ported from the legacy site.
6. **Offline handling** — the floating player and `/live` page already surface
   an on-air/offline signal (`radio.Live`, driven by Shoutcast's
   `streamstatus`), but nothing in the UI *acts* on it yet: the play button
   stays clickable and the "no song" fallback text ("Sedang mengudara") shows
   even when the stream itself is down.

**Verified against the live legacy site** (`classyfm.co.id`) as part of this
plan's research, all confirmed downloadable via plain `curl`:
- Logo: `https://classyfm.co.id/frontend/images/classy-logo-kecil.png` (PNG)
- Favicon: `https://classyfm.co.id/favicon.png` (PNG)
- The site's own navy accent color, pulled from its CSS: **`#031c4f`** (used
  52× in `frontend/css/main.css`, alongside a red `#ed0000`/`#c5050f` accent —
  this repo already owns the red half of that pairing, so `#031c4f` is the
  natural, on-brand navy to adopt).
- 8 real "Classiers" (`classyfm.co.id/crew/{id}`), each with a name, photo,
  DOB (day+month, no year), place of birth, a motto/bio line, interests, and
  social handles — full roster and field values captured in §15.4 below.
  **Puti Adelya has no working photo on the source site itself**
  (`<img src="https://classyfm.co.id/file/crew">`, literally the bare
  directory — confirmed broken on their own live site), so her seed row gets
  `photo_url = NULL` and falls back to the placeholder icon like any other
  imageless entity in this codebase.

**Scope decisions made while researching, to avoid over-building:**
- Classiers are **not** relationally linked to `programs`/`program_schedules`
  in this iteration — `host` stays a free-text field. Matching free text to a
  new `classiers.name` would be fragile (typos, "Budi" vs "Budi S.") and
  wasn't asked for. A future iteration can add that link once there's a real
  need to click from a program's host straight to their Classier profile.
- The floating player's initial (pre-JS) state is **not** wired to SSR
  now-playing data — only `/live`'s own big button gets a server-rendered
  correct initial `disabled` state (via its existing `.Now` view-model). The
  floating pill relies on the *existing* "poll `/api/nowplaying` immediately
  on every page load" behavior (`radio.js` already does this unconditionally)
  to reach the correct disabled state within one fetch round-trip. Threading
  `Now` into `baseData` for every single page render is a bigger, unrelated
  refactor not justified by this ask.
- Legacy-site seed photos (Classiers, logo, favicon) are downloaded **once**
  during implementation and committed as static assets under
  `web/static/img/` (embedded via the existing `//go:embed static`
  directive — no code change needed, just adding files). This is distinct
  from the *admin-editable* upload path (`web/uploads/`, `saveUploadedImage`)
  which still exists for admins to change a Classier's photo later.

---

## Phase 11 — Rebrand: navy primary color + real logo/favicon

### 11.1 Color swap

`tailwind.config.js`:
```js
colors: {
  brand: {
    DEFAULT: "#031c4f", // ClassyFM navy (sourced from classyfm.co.id's own CSS)
    dark: "#010f30",
    light: "#26468f",
  },
  ink: "#141414",
},
```
This is the only edit needed for the color itself — every template already
consumes `bg-brand`/`text-brand`/`border-brand`/`from-brand`/`to-brand-dark`/
`ring-brand` Tailwind classes, never raw hex (confirmed via repo-wide grep
across all 21 templates that reference brand colors). Rebuild with `make css`.

**Leave untouched**: `text-red-*`/`bg-red-*`/`border-red-*` utility classes in
`admin/login.html`, `admin/feed_sources_list.html`, `admin/news_form.html`,
`admin/programs_form.html` — these are semantic error/status colors, not the
brand palette, and swapping them would make validation errors and feed-source
failure states look like a neutral/branded state instead of a warning.

### 11.2 Real logo + favicon

Download once (implementation-time, not a runtime fetch):
```bash
mkdir -p web/static/img
curl -sL -o web/static/img/logo.png    "https://classyfm.co.id/frontend/images/classy-logo-kecil.png"
curl -sL -o web/static/img/favicon.png "https://classyfm.co.id/favicon.png"
```

`web/templates/partials/header.html` — replace the inline-SVG icon box with
the real logo image (drop the separate station-name text span; the logo image
already carries the wordmark):
```html
<a href="/" class="flex items-center gap-2">
  <img src="/static/img/logo.png" alt="{{.Base.StationName}}" class="h-10 w-auto">
</a>
```

`web/templates/layouts/base.html` and `web/templates/layouts/admin.html` —
add a real favicon link (currently **absent** from both `<head>`s):
```html
<link rel="icon" type="image/png" href="/static/img/favicon.png">
```

No Go/routing changes needed — `/static/*` already serves everything under
`web/static/` via the existing embedded FS (`web/assets.go`'s
`//go:embed static`), so newly-added files are picked up automatically on the
next `go build`.

### 11.3 Verification
- [ ] `make css && make build` — confirm no Tailwind class errors.
- [ ] Every page: header shows the real logo, browser tab shows the real
      favicon.
- [ ] Spot-check a few brand-colored elements (nav active state, radio
      buttons, news-card source label, hero gradient) — all render navy, not
      red.
- [ ] Admin login error banner and form validation errors still render red
      (unaffected by the swap).

---

## Phase 12 — Home page reorder + welcome section

Current order in `web/templates/public/home.html`: welcome blurb → hero
slideshow → newsfeed grid → today-programs partial → hot release grid.

Requested order: navbar (unchanged, from `base.html`) → **hero news** →
**welcome section** → **live program** → **news** → **hot release** → footer
(unchanged, from `base.html`).

This is purely a template reorder — no handler changes in this phase (the
grouping change to the "news" section itself is Phase 13, done in the same
file). Move the existing welcome `<section>` (lines 2–11) to *after* the hero
slideshow block, before `{{template "today-programs" .}}`. Final block order
in the file:
1. Hero slideshow (`{{if .Hero}}...{{else}}...{{end}}`)
2. Welcome section (unchanged content/markup, just relocated)
3. `{{template "today-programs" .}}` ("live program")
4. News section (Phase 13 rewrites this from the current flat "Newsfeed" grid)
5. Hot Release grid

### 12.1 Verification
- [x] Load `/` — section order top-to-bottom is exactly: header, hero
      slideshow, welcome text, "Program Hari Ini" (live program), news, hot
      release, footer, floating player.
- [x] No visual regression in any individual section's own styling.

---

## Phase 13 — News grouped by source, with a banner heading per group

Both Home and `/news` need: group by source, latest-first within each group,
a banner heading per source group. Per your direction, `/news`'s default
("Semua") view moves from a flat paginated list to grouped preview sections;
selecting a specific source pill still shows today's flat paginated
single-source list (grouping one source into "a group of one" adds nothing).

### 13.1 Handler helper

`internal/handlers/public/public.go` — add a shared grouping helper (no new
SQL needed; reuses the existing `ListPublishedNewsBySource` query, already
used by the paginated path, called with `Offset: 0` per source):

```go
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
```

### 13.2 Home

`Home()`: Hero keeps pulling from `ListLatestPublished`, but only needs 3 now
(the old code fetched 9 and split `[:3]`/`[3:]` because the flat "Newsfeed"
section reused the tail of that same query — that reuse goes away):

```go
if items, err := h.q.ListLatestPublished(r.Context(), 3); err == nil {
    hero = items
}
newsfeed := h.newsGroups(r.Context(), []string{"youtube", "klikpositif", "katasumbar"}, 3)
```

`hot_release` is deliberately excluded from this group set — it already has
its own dedicated "Hot Release" section further down the page (Phase 12's
order), so including it here would show it twice.

`web/templates/public/home.html`, News section — replace the flat
`{{range .Newsfeed}}` grid with grouped sections:
```html
{{if .Newsfeed}}
<section class="container-page py-4">
  <h2 class="text-2xl font-bold">Berita</h2>
  {{range .Newsfeed}}
  <div class="mt-6">
    {{template "news-source-banner" .}}
    <div class="mt-4 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      {{range .Items}}{{template "news-card" .}}{{end}}
    </div>
  </div>
  {{end}}
  <a href="/news" class="mt-6 inline-block text-sm font-medium text-brand hover:underline">Lihat semua berita &rarr;</a>
</section>
{{end}}
```
(`.Newsfeed` is now `[]newsGroup`, not `[]sqlc.NewsItem` — update the struct
field type in `Home()`'s `h.r.Page(...)` call accordingly.)

### 13.3 News page

`News()`: when `source == ""`, build grouped data instead of the flat
paginated query; when a specific source is selected, keep today's exact
flat/paginated behavior unchanged:

```go
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
```
Pass both `Groups []newsGroup` and the existing `Items`/`Page`/`TotalPages`
fields in the page struct; the template branches on `.SourceFilter`.

`web/templates/public/news.html`: keep the existing filter pills unchanged.
Below them:
```html
{{if eq .SourceFilter ""}}
  {{if .Groups}}
  {{range .Groups}}
  <div class="mt-8">
    {{template "news-source-banner" .}}
    <div class="mt-4 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      {{range .Items}}{{template "news-card" .}}{{end}}
    </div>
    <a href="/news?source={{.Source}}" class="mt-3 inline-block text-sm font-medium text-brand hover:underline">Lihat lebih banyak {{.Label}} &rarr;</a>
  </div>
  {{end}}
  {{else}}<div class="mt-8 rounded-xl border border-dashed border-gray-300 p-10 text-center text-gray-400">Belum ada berita.</div>{{end}}
{{else}}
  {{/* existing flat .Items grid + pagination controls, unchanged */}}
{{end}}
```

### 13.4 Source banner partial

New `web/templates/partials/news-source-banner.html` (shared by Home and
News; takes a `newsGroup` as `.`):
```html
{{define "news-source-banner"}}
<div class="flex items-center gap-2 rounded-lg bg-brand px-4 py-2.5 text-white">
  {{if eq .Source "youtube"}}{{template "icon-youtube"}}{{end}}
  <h3 class="text-sm font-bold uppercase tracking-wide">{{.Label}}</h3>
</div>
{{end}}
```
Only YouTube gets a brand-mark icon (already exists in `icons.html`) —
KlikPositif/KataSumbar/Hot Release have no equivalent local logo asset, so
their banners are text-only; adding placeholder logos for those wasn't asked
for and isn't available from the legacy site.

### 13.5 Verification
- [x] `/` — News section shows up to 3 groups (YouTube, KlikPositif,
      KataSumbar), each with a banner heading, latest-first, skipping any
      source with zero items. Hot Release does not appear twice.
- [x] `/news` (no filter) — shows up to 4 groups including Hot Release, each
      capped at 6 items, banner heading per group, "Lihat lebih banyak X"
      link per group navigates to `/news?source=X`.
- [x] `/news?source=youtube` (any single source) — unchanged flat paginated
      behavior, still works exactly as before.
- [x] A source with zero published items doesn't render an empty group/banner
      anywhere. (verified by code inspection: `newsGroups` skips any source
      with `len(items) == 0`; no live source currently has zero items to
      exercise this empirically.)

---

## Phase 14 — Program detail page

No schema/query change needed — `GetProgramBySlug` and
`ListSchedulesForProgram` already exist and return everything required;
`GetProgramBySlug` was defined in Phase 2 but never called from any handler
until now.

### 14.1 Handler

`internal/handlers/public/public.go` — add:
```go
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
```
Note `row.Host` here is `program_schedules.host` directly (`ListSchedulesForProgram`
returns raw `sqlc.ProgramSchedule` rows, not the joined view used elsewhere) —
same `sql.NullString` shape `models.ResolveHost` already expects.

`cmd/server/main.go`: `r.Get("/program/{slug}", ph.ProgramDetail)`.

### 14.2 Template

New `web/templates/public/program_detail.html` (follows `news_detail.html`'s
single-item page pattern): large banner image (or `icon-program-placeholder`
fallback), on-air badge if `.OnAir`, title, full description, and every
`models.GroupSlots` group rendered as a list (not the truncated inline chip
row) — same per-group markup as `program.html` already uses
(`{{weekday $g.FromDay}}...{{$g.Host}}`), just as a vertical list instead of a
wrapped chip line since there's room for it here.

`web/templates/public/program.html`: wrap each card in
`<a href="/program/{{.Program.Slug}}">` so the listing links into the detail
page (currently cards are plain `<div>`s, not links).

`internal/handlers/public/public.go`'s `Sitemap()`: no change required for
`/program` itself, but add one `sitemapURL` per active program slug (mirrors
the existing Hot Release loop) — `ListActivePrograms` is already called
elsewhere in this file, reuse the pattern.

### 14.3 Verification
- [x] `/program` — every card is now a link into `/program/{slug}`.
- [x] `/program/{slug}` for an active program — shows full schedule (all
      groups, not truncated); on-air badge reuses the shared `computeOnAir`
      already exercised by `/program` (no program was live during testing to
      observe directly, but logic is identical).
- [x] `/program/{slug}` for an inactive program or unknown slug — 404.
- [x] `/sitemap.xml` includes one entry per active program.

---

## Phase 15 — Classiers (hosts) page

New first-class entity: public list (`/classiers`) + detail
(`/classiers/{slug}`) pages, full admin CRUD mirroring the existing Programs
admin pattern, and seed data ported from `classyfm.co.id/crew`.

### 15.1 Schema

New migration `internal/db/migrations/0005_classiers.up.sql`:
```sql
CREATE TABLE classiers (
  id          BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  name        VARCHAR(255) NOT NULL,
  slug        VARCHAR(255) NOT NULL UNIQUE,
  role        VARCHAR(255) NULL,
  photo_url   VARCHAR(1000) NULL,
  bio         TEXT NULL,
  birth_place VARCHAR(255) NULL,
  birth_date  VARCHAR(20) NULL,
  instagram   VARCHAR(255) NULL,
  twitter     VARCHAR(255) NULL,
  facebook    VARCHAR(255) NULL,
  sort_order  INT NOT NULL DEFAULT 0,
  is_active   TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```
`.down.sql`: `DROP TABLE classiers;`

`birth_date` is `VARCHAR(20)`, not a SQL `DATE` — the legacy site only shows
day+month ("17 October"), no year; forcing a fake year into a real `DATE`
column would be worse than a plain string.

### 15.2 Queries

New `internal/db/queries/classiers.sql`, mirroring `programs.sql`'s shape:
```sql
-- name: ListActiveClassiers :many
SELECT * FROM classiers WHERE is_active = 1 ORDER BY sort_order ASC, name ASC;

-- name: ListClassiers :many
SELECT * FROM classiers ORDER BY sort_order ASC, name ASC;

-- name: GetClassier :one
SELECT * FROM classiers WHERE id = ?;

-- name: GetActiveClassierBySlug :one
SELECT * FROM classiers WHERE slug = ? AND is_active = 1;

-- name: CreateClassier :execresult
INSERT INTO classiers (name, slug, role, photo_url, bio, birth_place, birth_date, instagram, twitter, facebook, sort_order, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateClassier :exec
UPDATE classiers
SET name=?, slug=?, role=?, photo_url=?, bio=?, birth_place=?, birth_date=?, instagram=?, twitter=?, facebook=?, sort_order=?, is_active=?
WHERE id=?;

-- name: DeleteClassier :exec
DELETE FROM classiers WHERE id = ?;
```
`make sqlc` after adding this file regenerates `internal/db/sqlc` with a new
`Classier` model + these query methods.

### 15.3 Public pages

`internal/handlers/public/public.go`:
```go
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
```
`cmd/server/main.go`: `r.Get("/classiers", ph.Classiers)` and
`r.Get("/classiers/{slug}", ph.ClassierDetail)`.

New `web/templates/public/classiers.html` (grid of cards, follows
`program.html`'s card-grid pattern): photo or a new `icon-classier-placeholder`
fallback (person-silhouette glyph, added to `icons.html` alongside the
existing `icon-program-placeholder`), name, role, each card linking to
`/classiers/{slug}`.

New `web/templates/public/classier_detail.html` (follows `news_detail.html`'s
single-item pattern): large photo/placeholder, name, role, bio paragraph,
birth place/date as a small `<dl>`, and social links (Instagram/Twitter/
Facebook — only rendered when the corresponding field is non-empty) using the
existing brand-logo icons from `icons.html`.

`web/templates/partials/header.html`: add a "Classiers" nav link (`/classiers`).
`internal/handlers/public/public.go`'s `Sitemap()`: add `/classiers` plus one
entry per active classier slug (same loop pattern as programs/hot-release).

### 15.4 Admin CRUD

`internal/handlers/admin/classiers.go` (new file, mirrors
`internal/handlers/admin/programs.go` structurally — list/new/create/edit/
update/delete, `classierFromForm` reading a multipart form and calling
`h.saveUploadedImage(r, "photo", "classiers")` for the image, same
`current_image_url` hidden-field carry-forward pattern):
- `uploadSubdirClassiers = "classiers"` constant in `uploads.go` alongside the
  existing `uploadSubdirPrograms`.
- Routes in `cmd/server/main.go`'s admin group:
  ```go
  pr.Get("/classiers", ah.ClassiersList)
  pr.Get("/classiers/new", ah.ClassierNew)
  pr.Post("/classiers", ah.ClassierCreate)
  pr.Get("/classiers/{id}/edit", ah.ClassierEdit)
  pr.Post("/classiers/{id}", ah.ClassierUpdate)
  pr.Post("/classiers/{id}/delete", ah.ClassierDelete)
  ```
- New `web/templates/admin/classiers_list.html` + `classiers_form.html`,
  mirroring `programs_list.html`/`programs_form.html`'s markup (table list +
  form with name/slug/role/bio/birth place/birth date/social fields/photo
  upload/sort order/active toggle).
- `web/templates/layouts/admin.html`: add a "Classiers" nav link
  (`/admin/classiers`).

### 15.5 Seed data — real roster from `classyfm.co.id/crew`

Download the 7 available photos (Puti Adelya's is broken on the source site
itself, confirmed via direct inspection — leave `photo_url` NULL for her):
```bash
mkdir -p web/static/img/classiers
curl -sL -o web/static/img/classiers/andahayani.jpg      "https://classyfm.co.id/file/crew/9ffe93d693eb4627409f5eb86fdc84f5af426372.jpg"
curl -sL -o web/static/img/classiers/dara-rinjani.jpg    "https://classyfm.co.id/file/crew/b538b5c01e3650584da77c9d195de817065d0a80.jpg"
curl -sL -o web/static/img/classiers/dhanny-yudiska.png  "https://classyfm.co.id/file/crew/381b8721a0a6737f7249bce703ccbcacade06c38.png"
curl -sL -o web/static/img/classiers/edo-prussiano.jpg   "https://classyfm.co.id/file/crew/d39a8797c1760a0dabb701432eb4f1a844d18c84.jpg"
curl -sL -o web/static/img/classiers/jasmine-andrea.jpg  "https://classyfm.co.id/file/crew/65e45f8c378b74dff3476e056c48afd6e7289d10.jpg"
curl -sL -o web/static/img/classiers/via-helvita.png     "https://classyfm.co.id/file/crew/b3eb23a530021be141e9aee29153e94be25bb404.png"
curl -sL -o web/static/img/classiers/yeni-maiasnita.png  "https://classyfm.co.id/file/crew/8d3d7e12a828c2b27b2e95d31c6801c6ebfcc658.png"
```

New migration `internal/db/migrations/0006_seed_classiers.up.sql` (data-only
migration, same pattern as `0002_seed_feed_sources.up.sql`):
```sql
INSERT INTO classiers (name, slug, role, photo_url, bio, birth_place, birth_date, instagram, twitter, facebook, sort_order) VALUES
('Andahayani', 'andahayani', 'Classier',
 '/static/img/classiers/andahayani.jpg',
 'It''s Not About How Life is Treating You but You Are Treating Life. Suka public speaking, broadcasting, dan menjelajahi tempat baru.',
 'Jambi', '17 Oktober', 'anda_joseph', 'anda_joseph', 'Andahayani Yoseph', 1),
('Dara Rinjani', 'dara-rinjani', 'Reporter',
 '/static/img/classiers/dara-rinjani.jpg',
 'Berbuat Baik dan Mandiri. Suka traveling, jurnalistik, dan pencinta kucing.',
 'Riau', '11 Januari', NULL, NULL, 'Zulia Yandani', 2),
('Dhanny Yudiska', 'dhanny-yudiska', 'Classier',
 '/static/img/classiers/dhanny-yudiska.png',
 'Im a grown man... I do whatever i want! Suka musik, film, broadcasting, dan hal-hal yang membawa kebahagiaan.',
 'Maninjau', '18 Agustus', 'dhannyyudiska', 'dhannyyudiska', 'dhanny''arif''yudiska', 3),
('Edo Prussiano', 'edo-prussiano', 'Classier',
 '/static/img/classiers/edo-prussiano.jpg',
 'Kalo mau kaya, ya harus capek! Suka healing.',
 'Padang', '28 Juni', 'edo_pruss17', 'edo_pruss17', NULL, 4),
('Jasmine Andrea', 'jasmine-andrea', 'Classier',
 '/static/img/classiers/jasmine-andrea.jpg',
 'Live is Art. Suka membaca, menulis, musik, film, dan seni.',
 'Batusangkar', '06 September', 'Jasmine Andrea', 'jasmineandrea06', NULL, 5),
('Puti Adelya', 'puti-adelya', 'Classier',
 NULL,
 'It''s just a bad day, not a bad life. Suka buku, film, seni, dan proyek sosial.',
 'Padang', '31 Mei', 'karzmp', NULL, NULL, 6),
('Via Helvita', 'via-helvita', 'Classier',
 '/static/img/classiers/via-helvita.png',
 'Hidup Gak Neko Neko. Suka traveling.',
 'Jakarta', '30 Oktober', 'vivi_chressya', 'vivi_chressya', 'Vivi d'' Chressya', 7),
('Yeni Maiasnita', 'yeni-maiasnita', 'Classier',
 '/static/img/classiers/yeni-maiasnita.png',
 'Always Do The Best And Enjoy Your Life. Suka film, broadcasting, human interest, dan psikologi. Membawakan "Communitalk with Yeni Maiasnita".',
 'Sawahlunto', '28 Mei', 'aunty.cantik', 'auntycantiq', 'aunty cantik', 8);
```
(Watch the doubled `''` for literal apostrophes in `dhanny'arif'yudiska` and
`Vivi d' Chressya` — standard SQL string-escaping, not a typo.)

`.down.sql`: `DELETE FROM classiers WHERE slug IN ('andahayani','dara-rinjani','dhanny-yudiska','edo-prussiano','jasmine-andrea','puti-adelya','via-helvita','yeni-maiasnita');`

### 15.6 Verification
- [x] `go build ./...`; run migrations `0005` and `0006`; `make sqlc`
      regenerates `Classier` model + queries. (No `golang-migrate` CLI
      available in this environment and no `schema_migrations` table exists
      — prior migrations 0001-0004 were applied the same way — so 0005/0006
      were applied directly via `mysql < file.sql`, consistent with existing
      practice here.)
- [x] `/classiers` — 8 cards, 7 with real photos, Puti Adelya shows the
      placeholder icon (not a broken image).
- [x] `/classiers/{slug}` for each — correct bio/birth/social fields, social
      icons only render for handles that are actually set (e.g. Edo Prussiano
      has no Facebook, Dara Rinjani has no Twitter).
- [x] `/admin/classiers` — full CRUD works, including photo upload/replace
      (verified live: created a classier with an uploaded photo, edited it
      with a replacement photo + deactivated it, then deleted it).
- [x] `/sitemap.xml` includes `/classiers` + all 8 slugs.

---

## Phase 16 — Offline handling: disable playback, relabel, hide song info

`radio.Live` (Shoutcast `streamstatus == 1`) already exists and already
drives the `/live` page's Status badge (`.js-np-live-badge`, "On Air"/
"Offline"). What's missing: the play button doesn't get disabled, and the
now-playing title text ("Sedang mengudara") doesn't distinguish "on air, no
metadata yet" from "stream is actually down."

### 16.1 `radio.js` — the core logic

`web/static/js/radio.js`'s `renderNowPlaying(np)` — branch title text and
cover/song visibility on `np.live` first, `np.has_song` second, and disable
every `.js-radio-toggle` button when offline:
```js
function renderNowPlaying(np) {
  var offline = !np.live;
  var titleText = offline
    ? "Offline"
    : (np.has_song ? [np.artist, np.song].filter(Boolean).join(" - ") : "Sedang mengudara");
  document.querySelectorAll(".js-np-title").forEach(function (el) { el.textContent = titleText; });

  document.querySelectorAll(".js-np-cover").forEach(function (el) {
    if (!offline && np.cover_url) { el.src = np.cover_url; el.classList.remove("hidden"); }
    else { el.classList.add("hidden"); }
  });
  document.querySelectorAll(".js-np-cover-placeholder").forEach(function (el) {
    el.classList.toggle("hidden", !offline && !!np.cover_url);
  });

  document.querySelectorAll(".js-np-bitrate").forEach(function (el) { el.textContent = np.bitrate ? np.bitrate + " kbps" : "—"; });
  document.querySelectorAll(".js-np-listeners").forEach(function (el) { el.textContent = typeof np.listeners === "number" ? np.listeners : "—"; });

  document.querySelectorAll(".js-np-live-badge").forEach(function (el) {
    el.textContent = np.live ? "On Air" : "Offline";
    el.classList.toggle("text-green-600", np.live);
    el.classList.toggle("text-gray-400", !np.live);
  });
  // New: the floating pill's small caption ("Live"/"Offline"), same idea as
  // the badge above but for the compact widget that has no room for a <dl>.
  document.querySelectorAll(".js-np-live-label").forEach(function (el) {
    el.textContent = np.live ? "Live" : "Offline";
  });

  document.querySelectorAll(".js-radio-toggle").forEach(function (toggle) {
    toggle.disabled = offline;
    toggle.classList.toggle("opacity-50", offline);
    toggle.classList.toggle("cursor-not-allowed", offline);
    toggle.setAttribute("aria-disabled", offline ? "true" : "false");
  });
  if (offline && !audio.paused) {
    audio.pause();
    setState("paused");
    localStorage.removeItem(STORAGE_KEY);
  }
}
```
No change needed to `toggleClick`/the click-binding loop — a real
`disabled` `<button>` doesn't fire `click` events in any browser, so the
existing listener is automatically inert while disabled.

### 16.2 SSR-correct initial state on `/live`

`web/templates/public/live.html` — the title text and button both need the
offline branch checked *first* (today it only checks `.Now.HasSong`):
```html
<p class="js-np-title text-2xl font-bold leading-snug text-gray-900">
  {{if not .Now.Live}}Offline{{else if .Now.HasSong}}{{if .Now.Artist}}{{.Now.Artist}} &mdash; {{end}}{{.Now.Song}}{{else}}Sedang mengudara{{end}}
</p>

<button type="button" class="js-radio-toggle relative mt-6 inline-flex h-20 w-20 items-center justify-center rounded-full bg-brand text-white transition-colors hover:bg-brand-dark focus:outline-none focus-visible:ring-2 focus-visible:ring-brand focus-visible:ring-offset-2 {{if not .Now.Live}}opacity-50 cursor-not-allowed{{end}}"
        {{if not .Now.Live}}disabled aria-disabled="true"{{end}}
        aria-pressed="false" aria-label="Putar radio">
```
Cover art / placeholder visibility gets the same `not .Now.Live` guard added
to their existing `{{if .Now.CoverURL}}`/`{{if not .Now.CoverURL}}` checks
(offline implies no cover regardless of what `CoverURL` happens to hold).

This gives `/live` a correct disabled state on first paint, before any JS
runs. The floating player (present on every page, including `/live` itself)
has no equivalent SSR data source today (see the Context section's scope
note) — it starts enabled and gets disabled by the *existing* immediate
`pollNowPlaying()` call that already runs unconditionally on every page load,
which is a sub-second gap in practice, not a new behavior this phase
introduces.

### 16.3 Floating pill markup

`web/templates/partials/radio-player.html` — give the small caption a class
so `radio.js` can update it (currently hardcoded "Live" text, never changes):
```html
<p class="truncate text-xs font-semibold uppercase tracking-wide text-brand js-np-live-label">Live</p>
```

### 16.4 Verification
- [ ] With the stream reachable and live: floating pill and `/live` both show
      artist/song (or "Sedang mengudara" if Shoutcast has no current
      metadata), play button enabled and working, caption/badge say "Live"/
      "On Air". **Not verified** — the actual Shoutcast stream is not
      reachable from this dev sandbox (`/api/nowplaying` returns `"live":
      false` even now), so the "online" branch could only be checked by code
      review, not observed live.
- [x] Simulate offline: no simulation needed — the stream is genuinely
      unreachable from this environment right now, so this was verified
      against real data, not a fake host. `/api/nowplaying` returns
      `"live":false`; `/live`'s SSR HTML (fetched with plain `curl`, no JS)
      already has the button `disabled`, `opacity-50 cursor-not-allowed`,
      title text "Offline", cover `<img>` hidden with the placeholder icon
      shown instead, and the badge reading "Offline".
- [ ] If audio was already playing when the stream drops, it audibly stops
      and the button reflects the paused/disabled state. **Not verified** —
      no browser/audio driver available in this environment to actually play
      audio and observe the drop; confirmed by code review only (the
      `offline && !audio.paused` branch in `renderNowPlaying` calls
      `audio.pause()`, mirrors the plan exactly).
- [x] Load `/live` directly (hard load, JS not yet run) while offline —
      button is already disabled server-side, no flash of an enabled button.
      Verified with `curl` (renders no JS at all): the fetched HTML already
      contains `disabled aria-disabled="true"` on `.js-radio-toggle`.
- [ ] Stream comes back: within one 15s poll, button re-enables and playback
      resumes normally. **Not verified** — stream is not reachable from this
      environment at all, so there's no "comes back" transition to observe
      here; logic reviewed by code (same `renderNowPlaying` branch handles
      both directions symmetrically, no online-specific special-casing).

---

## Migration / dependency summary

| Item | Change |
|---|---|
| `internal/db/migrations/0005_classiers.{up,down}.sql` | new `classiers` table |
| `internal/db/migrations/0006_seed_classiers.{up,down}.sql` | seeds the 8 real Classiers |
| `internal/db/queries/classiers.sql` | new — run `make sqlc` after adding |
| `tailwind.config.js` | `brand.{DEFAULT,dark,light}` → navy values |
| `web/static/img/logo.png`, `favicon.png` | new, downloaded from `classyfm.co.id` |
| `web/static/img/classiers/*.{jpg,png}` | new, 7 files downloaded from `classyfm.co.id` |

No new Go module dependencies. No new env vars.

## Critical files

- `tailwind.config.js` — navy rebrand (Phase 11)
- `web/templates/partials/header.html`, `layouts/base.html`, `layouts/admin.html` — real logo/favicon (Phase 11)
- `web/templates/public/home.html` — section reorder (Phase 12) + grouped news rewrite (Phase 13)
- `internal/handlers/public/public.go` — `newsGroups` helper, `Home`/`News` rewrites (Phase 13), new `ProgramDetail` (Phase 14), new `Classiers`/`ClassierDetail` (Phase 15)
- `web/templates/partials/news-source-banner.html` — new shared partial (Phase 13)
- `web/templates/public/news.html` — grouped default view (Phase 13)
- `web/templates/public/program.html` + new `program_detail.html` — detail page (Phase 14)
- `internal/db/migrations/0005_classiers.*`, `0006_seed_classiers.*`, `internal/db/queries/classiers.sql` — new entity (Phase 15)
- `internal/handlers/admin/classiers.go` (new) + `internal/handlers/admin/uploads.go` (`uploadSubdirClassiers`) — admin CRUD (Phase 15)
- new `web/templates/public/classiers.html`, `classier_detail.html`, `admin/classiers_list.html`, `admin/classiers_form.html`
- `web/static/js/radio.js`, `web/templates/public/live.html`, `web/templates/partials/radio-player.html` — offline handling (Phase 16)
- `cmd/server/main.go` — new routes: `/program/{slug}`, `/classiers`, `/classiers/{slug}`, `/admin/classiers*`
