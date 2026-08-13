# ClassyFM Radio — Website Development Plan

> Public website + matching admin for **Classy 103.4 FM** (classyfm.co.id).
> This plan is **phased** and designed to be executed **partially** — each phase is a
> self-contained, shippable increment. Check items off as you go.

---

## 1. Goals & Scope

Build a public website with a **persistent floating radio player** and an admin panel
that manages exactly what the public pages consume — nothing more, nothing less.

**Public pages**

1. **Home** — hero (3 latest newsfeed items), today's radio programs, newsfeed, *Hot Release*.
2. **Program** — all radio programs + weekly schedule.
3. **Media** — ClassyFM on Instagram, Facebook, X, YouTube.
4. **News** — full newsfeed (aggregated + Hot Release).

**Cross-cutting:** floating play/pause internet-radio button on **every** page, with
live now-playing metadata.

**Admin pages** mirror public data needs: Hot Release CRUD, Programs CRUD + schedule,
Media links, feed-source config/monitoring, and station settings (stream URL, etc.).

---

## 2. Confirmed Decisions

| Area | Decision |
|------|----------|
| Language / runtime | **Go** (monolith) |
| Templating | `html/template` (server-rendered) |
| Router | **chi** |
| Database | **MySQL** |
| DB access | **sqlc** (typed queries from raw SQL) |
| Migrations | `golang-migrate` (or `goose`) |
| CSS | **Tailwind CSS** (standalone CLI, no Node app) |
| Icons | **Lucide** (inline SVG) |
| News aggregation | **Automated** — background worker pulls sources on a ticker, caches in MySQL |
| Radio stream | `https://c4.siar.us:10340/stream.mp3` (Shoutcast, 128kbps MP3) |
| Now-playing API | `https://classyfm.co.id/streaming/info` → `{artist, song, cover.url, title}` |
| Program feed | `https://classyfm.co.id/streaming/info/program` (upstream weekly schedule) |

**Sources for the newsfeed:** ClassyFM YouTube (RSS/Atom feed, no API key), KlikPositif,
KataSumbar (RSS if available, else HTML scrape), plus admin-authored **Hot Release**.

---

## 3. Architecture & Project Layout

Standard Go monolith. Server renders HTML; a small amount of vanilla JS drives the
radio player and now-playing polling. No SPA framework.

```
classyfm/
├── cmd/
│   └── server/main.go            # wiring: config, db, router, worker, http.Server
├── internal/
│   ├── config/                   # env-based config loader
│   ├── db/
│   │   ├── migrations/           # *.up.sql / *.down.sql
│   │   ├── queries/              # *.sql for sqlc
│   │   └── sqlc/                 # generated code (sqlc generate)
│   ├── handlers/
│   │   ├── public/               # home, program, media, news
│   │   └── admin/                # auth + CRUD handlers
│   ├── models/                   # domain structs / view models
│   ├── feeds/                    # aggregation engine
│   │   ├── youtube.go
│   │   ├── klikpositif.go
│   │   ├── katasumbar.go
│   │   ├── worker.go             # ticker orchestration
│   │   └── source.go             # Source interface
│   ├── radio/                    # now-playing proxy/cache + stream config
│   ├── middleware/               # auth, logging, recovery, CSRF
│   └── render/                   # template cache + helpers
├── web/
│   ├── templates/
│   │   ├── layouts/base.html     # includes floating radio player partial
│   │   ├── partials/             # header, footer, radio-player, news-card, program-card
│   │   ├── public/               # home, program, media, news
│   │   └── admin/                # login, dashboard, *_list, *_form
│   ├── static/
│   │   ├── css/tailwind.css      # source (@tailwind directives)
│   │   ├── css/app.css           # generated output
│   │   ├── js/radio.js           # player + now-playing polling
│   │   ├── js/admin.js
│   │   └── img/
│   └── icons/                    # lucide SVGs used inline
├── sqlc.yaml
├── tailwind.config.js
├── Makefile                      # tailwind, sqlc, migrate, run, build
├── .env.example
└── PLAN.md
```

**Conventions**
- Handlers thin; business logic in `feeds`/`radio`/`models`.
- All DB access through sqlc-generated methods; no ad-hoc SQL in handlers.
- Templates precompiled into a cache at boot (dev: reload on change via build tag).
- Errors logged with `slog`; user-facing pages show friendly error states.

---

## 4. Data Model (MySQL)

Design the aggregated feed and Hot Release as **one unified table** so the home hero,
newsfeed, and News page all read from a single source with a `source` discriminator.

```sql
-- news_items: aggregated cache + admin Hot Release
CREATE TABLE news_items (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  source        ENUM('youtube','klikpositif','katasumbar','hot_release') NOT NULL,
  external_id   VARCHAR(255) NULL,        -- video id / article guid; NULL for hot_release
  title         VARCHAR(500) NOT NULL,
  slug          VARCHAR(255) NULL,        -- internal slug for hot_release detail pages
  excerpt       TEXT NULL,
  content       MEDIUMTEXT NULL,          -- HTML body, hot_release only
  url           VARCHAR(1000) NULL,       -- external link (aggregated) 
  image_url     VARCHAR(1000) NULL,
  published_at  DATETIME NOT NULL,
  is_published  TINYINT(1) NOT NULL DEFAULT 1,
  is_featured   TINYINT(1) NOT NULL DEFAULT 0,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_source_external (source, external_id),
  KEY idx_feed (is_published, published_at),
  KEY idx_source (source, published_at)
);

-- programs + weekly schedule
CREATE TABLE programs (
  id          BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  title       VARCHAR(255) NOT NULL,
  slug        VARCHAR(255) NOT NULL UNIQUE,
  description TEXT NULL,
  host        VARCHAR(255) NULL,
  image_url   VARCHAR(1000) NULL,
  sort_order  INT NOT NULL DEFAULT 0,
  is_active   TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE program_schedules (
  id          BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  program_id  BIGINT UNSIGNED NOT NULL,
  day_of_week TINYINT NOT NULL,           -- 0=Sun..6=Sat
  start_time  TIME NOT NULL,
  end_time    TIME NOT NULL,
  KEY idx_prog (program_id),
  KEY idx_day (day_of_week, start_time),
  CONSTRAINT fk_sched_prog FOREIGN KEY (program_id) REFERENCES programs(id) ON DELETE CASCADE
);

-- media/social links for Media page
CREATE TABLE media_links (
  id         BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  platform   ENUM('instagram','facebook','x','youtube') NOT NULL,
  label      VARCHAR(255) NULL,
  url        VARCHAR(1000) NOT NULL,
  handle     VARCHAR(255) NULL,
  embed_html TEXT NULL,                    -- optional embed snippet
  sort_order INT NOT NULL DEFAULT 0,
  is_active  TINYINT(1) NOT NULL DEFAULT 1
);

-- feed source config + monitoring
CREATE TABLE feed_sources (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  source          ENUM('youtube','klikpositif','katasumbar') NOT NULL UNIQUE,
  is_enabled      TINYINT(1) NOT NULL DEFAULT 1,
  endpoint        VARCHAR(1000) NULL,      -- channel id / RSS URL / base URL
  last_fetched_at DATETIME NULL,
  last_status     VARCHAR(255) NULL,       -- ok / error message
  item_count      INT NOT NULL DEFAULT 0
);

-- key/value settings (stream url, station name, youtube key ref, etc.)
CREATE TABLE settings (
  k  VARCHAR(191) PRIMARY KEY,
  v  TEXT NOT NULL
);

-- admin users
CREATE TABLE users (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  email         VARCHAR(191) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name          VARCHAR(255) NOT NULL,
  role          ENUM('admin','editor') NOT NULL DEFAULT 'admin',
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
  token      CHAR(64) PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  expires_at DATETIME NOT NULL,
  KEY idx_exp (expires_at)
);
```

**Seed settings:** `stream_url`, `nowplaying_url`, `station_name`, `station_tagline`,
`youtube_channel_id`.

---

## 5. External Integrations

| Integration | Endpoint / method | Notes |
|-------------|-------------------|-------|
| **Audio stream** | `https://c4.siar.us:10340/stream.mp3` | HTML5 `<audio>`; stored in `settings.stream_url` |
| **Now playing** | `https://classyfm.co.id/streaming/info` | JSON; proxy+cache to avoid CORS & rate issues, poll ~15–20s |
| **Program feed (upstream)** | `https://classyfm.co.id/streaming/info/program` | Optional import source for the schedule |
| **YouTube** | Atom feed: `https://www.youtube.com/feeds/videos.xml?channel_id={id}` | No API key; parse Atom/XML, map to `news_items(source='youtube')` |
| **KlikPositif** | RSS feed if exposed, else HTML scrape (goquery) | Category filter optional |
| **KataSumbar** | RSS feed if exposed, else HTML scrape (goquery) | Category filter optional |

> **Scraping note:** prefer RSS. If scraping, isolate CSS selectors in one place per
> source so upstream layout changes are a one-file fix. Set a descriptive User-Agent,
> respect robots.txt, and rate-limit.

---

## 6. Aggregation Engine

- `Source` interface: `Fetch(ctx) ([]NewsItem, error)` + `Name()`.
- `worker.go`: `time.Ticker` (default 30 min, configurable) iterates enabled sources,
  fetches, **upserts** by `(source, external_id)`, updates `feed_sources` status/counters.
- Runs on boot (immediate first pass) then on interval; guarded by context for graceful
  shutdown. Per-source failures are isolated (one bad source ≠ whole run fails).
- Manual "Refresh now" trigger available from admin (calls the same run function).
- Hot Release is **not** touched by the worker (admin-owned rows).

---

## 7. Public Pages — Specification

### Floating Radio Player (all pages)
- Fixed bottom-right button; play/pause toggles a single `<audio>` element.
- Expands to show now-playing **cover / artist / song** (from cached `/info`).
- State persists across navigation via `localStorage` (playing flag) + small JS;
  because pages are server-rendered, re-attach on load and resume if flag set.
  *(Optional later: keep audio alive via a Turbo/htmx-style persistent element.)*
- Accessible: real `<button>`, `aria-pressed`, keyboard toggle, visible focus.

### 7.1 Home
- **Hero:** 3 latest `news_items` (any source, `is_published`, newest `published_at`).
- **Today's programs:** `program_schedules` for current `day_of_week`, ordered by time,
  with "on air now" highlight.
- **Newsfeed:** paginated/most-recent mixed feed (all sources).
- **Hot Release section:** `news_items WHERE source='hot_release'`, newest first.

### 7.2 Program
- All active programs (cards: image, title, host, description).
- Weekly schedule grid (day × time) from `program_schedules`.

### 7.3 Media
- Sections for Instagram, Facebook, X, YouTube from `media_links` (links + optional embeds).

### 7.4 News
- Full aggregated feed, filterable by source, paginated.
- Hot Release items open an internal detail page (`/news/{slug}`) rendering `content`;
  aggregated items link out to the original `url`.

**SEO:** semantic HTML, per-page `<title>`/meta, Open Graph, sitemap.xml, RSS out (optional).

---

## 8. Admin Pages — Specification

Mirror public data needs exactly.

- **Auth:** email/password login, bcrypt, server-side sessions, CSRF on all forms.
- **Dashboard:** feed-source health (last fetch, status, counts), quick links.
- **Hot Release:** list + create/edit/delete (title, slug, excerpt, HTML content, image,
  published/featured toggles, publish date).
- **News (aggregated):** read-only list with source filter; per-item publish/unpublish &
  feature toggle; **Refresh now** button.
- **Programs:** CRUD + manage weekly schedule slots; sort order; active toggle.
- **Media:** CRUD social links/embeds per platform.
- **Feed Sources:** enable/disable, edit endpoint (channel id / RSS URL), view status.
- **Settings:** stream URL, now-playing URL, station name/tagline, YouTube channel id.
- **Users:** (optional) manage admin/editor accounts.

---

## 9. Phased Execution Roadmap

Each phase is independently shippable. Do them in order, or cherry-pick after Phase 1.

### Phase 0 — Project scaffold  *(foundation)*  ✅ DONE
- [x] `go mod init`, chi router, `cmd/server/main.go`, graceful shutdown.
- [x] Config loader (`.env`), structured logging (`slog`).
- [x] MySQL connection pool; `sqlc.yaml`; migrate tooling; `Makefile`.
- [x] Tailwind standalone CLI build pipeline; base layout + Lucide inline icons.
- [x] Health check route; static file serving.

### Phase 1 — Radio player + shell  *(highest visible value, no data deps)*
- [x] Base template with header/footer + **floating player** partial.
- [x] `radio.js`: play/pause, `<audio>` for stream, persist state across pages.
- [x] Now-playing proxy handler (server fetch + cache `/streaming/info`), JS polling.
- [ ] Seed `settings` (stream_url, nowplaying_url, station_name).  *(deferred — needs DB; currently sourced from env/config)*
- [x] Stub Home/Program/Media/News pages with real layout.

### Phase 2 — Programs (public + admin)  ✅ DONE
- [x] Migrations + sqlc for `programs`, `program_schedules`.
- [x] Admin auth (users, sessions, login, middleware, CSRF).
- [x] Admin Programs CRUD + schedule editor.
- [x] Public Program page (cards + weekly grid); "today's programs" on Home.

### Phase 3 — Hot Release (public + admin)  ✅ DONE
- [x] Migrations/sqlc for `news_items` (hot_release path first).
- [x] Admin Hot Release CRUD (rich/HTML content, images, publish/feature).
- [x] Home Hot Release section; internal news detail page.

### Phase 4 — Automated newsfeed aggregation  ✅ DONE
- [x] `feed_sources` config + admin screen (`/admin/feed-sources`); `Source` interface
      (`internal/feeds/source.go`) + worker/ticker (`internal/feeds/worker.go`).
- [x] YouTube source (RSS/Atom feed) — no key needed, channel id seeded in
      `feed_sources.endpoint` (falls back to `YOUTUBE_CHANNEL_ID` env if cleared).
      Follows the same HTTP-fetch pattern as `internal/radio/radio.go` (context-scoped
      request, timeout, User-Agent) but parses Atom XML (`encoding/xml`) instead of JSON.
- [x] KlikPositif + KataSumbar sources — both expose standard WordPress RSS at
      `/feed/`, no scraping needed (`internal/feeds/wordpress.go`, shared parser).
- [x] Upsert dedupe (`UpsertNewsItem`, preserves admin publish/feature flags on
      re-fetch); per-source status (`feed_sources.last_status/last_fetched_at`);
      "Refresh now" button; admin aggregated-news list (`/admin/newsfeed`) with
      source filter + publish/feature toggles.
- [x] Wired Home hero (3 latest across all sources) + Home newsfeed section + full
      **News** page (`/news`, filterable + paginated).

### Phase 5 — Media page  ✅ DONE
- [x] `media_links` simplified to exactly one row per platform (`platform` is
      UNIQUE) with a single `url` column — no label/handle/embed content. Migration
      `0003_simplify_media_links` drops the earlier free-CRUD columns and seeds the
      4 rows with ClassyFM's real account URLs (found on the live
      classyfm.co.id site).
- [x] Admin `/admin/media`: fixed list of 4 platform rows, each with just a URL
      field, edited in place (mirrors the `feed_sources` admin pattern — no
      separate New/Edit/Delete pages, since there are always exactly 4 rows).
- [x] Public Media page (`/media`): one badge per platform (brand-logo SVG +
      label), shown only when that platform's URL is set, linking out to the
      account page. Brand icons added to `web/templates/partials/icons.html`
      (`icon-instagram`/`icon-facebook`/`icon-x`/`icon-youtube`, sourced from
      Simple Icons).
- [x] *(Superseded)* An earlier iteration of this phase supported admin-authored
      live embeds (`embed_html` + a dedicated sanitizer). Reverted in favor of the
      simpler fixed-badge design above — `internal/sanitize/embed.go` removed.

### Phase 6 — Polish & launch  *(in progress)*
- [x] SEO: Open Graph/Twitter meta + canonical link (`web/templates/layouts/base.html`),
      `/robots.txt` + `/sitemap.xml` (static pages + published Hot Release slugs, in
      `internal/handlers/public/public.go`). New `SITE_URL` config/env var.
- [ ] Responsive QA (mobile player behavior) and a11y pass — needs manual
      browser/screen-reader testing; not yet done.
- [x] Error pages: friendly 500 (`web/templates/public/error.html`) wired through a
      custom panic-recovery middleware (`internal/middleware/recover.go`) that
      replaces chi's default `Recoverer`, alongside the existing 404 page.
- [x] Rate limiting: in-memory per-IP limiter (`internal/middleware/ratelimit.go`),
      applied to `POST /admin/login` (10 req/min) to slow brute-force attempts.
- [x] Security headers: CSP, X-Frame-Options, X-Content-Type-Options,
      Referrer-Policy, Permissions-Policy, HSTS in prod
      (`internal/middleware/security.go`).
- [x] HTTPS/proxy config: nginx + Cloudflare real-IP restore + Let's Encrypt via
      certbot (`deploy/classyfm.remorac.com`, `deploy/install.sh`) — already in
      place from earlier deploy work.
- [x] Caching: now-playing metadata already cached (Phase 1); static assets now get
      an explicit `Cache-Control` (embed.FS has no ModTime for conditional GETs).
- [x] DB indexes verified — `news_items`, `program_schedules`, `sessions` all have
      indexes matching their query patterns (`0001_init.up.sql`).
- [x] Backups: `deploy/backup.sh` (mysqldump + gzip, 14-day retention) installed as
      a systemd timer (`classyfm-backup.timer`/`.service`) by `install.sh`.
- [x] Deploy: systemd unit + nginx + certbot already present (`deploy/classyfm.service`,
      `deploy/install.sh`); log rotation is handled by journald (the service logs to
      the journal, not a file, since it runs as `Type=simple` under systemd).

---

## 10. Cross-Cutting Concerns

- **Security:** bcrypt, CSRF tokens, secure/HttpOnly session cookies, input validation,
  `html/template` auto-escaping (be careful with `template.HTML` for embeds/Hot Release
  content — sanitize with bluemonday).
- **Performance:** cache now-playing (single upstream fetch shared by all visitors),
  cache aggregated feed reads, precompiled templates, indexed queries.
- **Resilience:** aggregation failures isolated; stale cache served if upstream down.
- **Config/secrets:** DB DSN, session secret via env, never committed.
- **Observability:** structured logs, feed-source status surfaced in admin dashboard.

---

## 11. Open Questions / Follow-ups

1. ~~YouTube API key~~ — **resolved**: switched to the free RSS/Atom feed, no API key
   needed. Channel ID resolved from classyfm.co.id's YouTube link
   (`youtube.com/user/classyfm`): **`UC7yseGthOK8sOtahtxvxDbg`**
   ("Classy 103.4 FM Padang"). Feed URL:
   `https://www.youtube.com/feeds/videos.xml?channel_id=UC7yseGthOK8sOtahtxvxDbg`.
2. ~~KlikPositif / KataSumbar RSS~~ — **resolved**: both are WordPress sites exposing
   standard RSS 2.0 at `https://klikpositif.com/feed/` and `https://katasumbar.com/feed/`
   (~10 latest items each, no auth). Using the general feed (no category filter);
   links point back to the original article, satisfying attribution.
3. **Hot Release content** — plain rich text, or full HTML/embeds? (drives sanitizer + editor)
4. ~~Media page~~ — **resolved**: one fixed badge per platform linking to the
   account page, admin edits just the URL (see Phase 5).
5. **Deployment target** — VPS (systemd) vs Docker; domain/reverse proxy already set up?
6. **Auth scope** — single admin, or multiple editors with roles?

---

*Next step when you're ready: say which phase to start (recommend **Phase 0 → 1**), and
I'll scaffold it.*
