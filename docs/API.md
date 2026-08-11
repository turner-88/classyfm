# ClassyFM Public JSON API

The public, read-oriented JSON API for the ClassyFM mobile app. All endpoints are
mounted under **`/api/v1`**.

- **Base URL:** `https://classyfm.co.id/api/v1`
- **Format:** JSON (`Content-Type: application/json; charset=utf-8`)
- **Orientation:** fully **read-only** and **cookie-free** — every endpoint is an open
  `GET`, with no authentication and no request body.

---

## Table of Contents

- [Conventions](#conventions)
  - [Response envelopes](#response-envelopes)
  - [Errors](#errors)
  - [Caching](#caching)
  - [Pagination & filtering](#pagination--filtering)
  - [URLs & images](#urls--images)
  - [Degraded mode](#degraded-mode)
- [Index of Endpoints](#endpoints)
- [App bootstrap endpoints](#app-bootstrap-1)
- [Content endpoints](#content-endpoints)
- [Live endpoints](#live-endpoints)
  - [Playing the live stream in an app](#playing-the-live-stream-in-an-app)
  - [When the backend API is unreachable](#when-the-backend-api-is-unreachable)
- [Advertising endpoints](#advertising)
- [Legal pages](#legal-pages)
- [Object reference](#object-reference)

---

## Conventions

### Response envelopes

| Shape | Used by | Body |
|-------|---------|------|
| **List** | roster/list endpoints | `{ "data": [ ... ] }` |
| **Paginated** | `GET /news?source=…`, <br>`GET /podcasts` | `{ "data": [ ... ], "meta": { "page": 1, "total_pages": 3, "total": 27 } }` |
| **Object** | detail & aggregate endpoints | the object directly (no wrapper) |

### Errors

Errors return `{ "error": "message" }` with the relevant HTTP status:

| Status | Meaning |
|--------|---------|
| `400` | Invalid path id or query parameter |
| `404` | Resource not found, or the feature is not configured |
| `500` | Server / database error |

### Caching

Each response carries a `Cache-Control` header:

- `public, max-age=60` — slowly-changing content (programs, news, podcasts, about,
  ads, home, config).
- `no-store` — live data (now-playing, schedule state, TikTok live).

<br><br>

### Pagination & filtering

Pagination applies only to `GET /news?source=…` and `GET /podcasts`:

- `page` — 1-based, defaults to `1` (values `< 1` are treated as `1`).
- Page size is fixed at **12**.
- `meta.total_pages` is at least `1` even when empty.

### URLs & images

All image and link fields are returned as **absolute URLs** (resolved against the site
origin, `https://classyfm.co.id`). Already-absolute URLs — e.g. aggregated news
thumbnails — pass through unchanged.

### Degraded mode

If the database is unavailable, **list** endpoints degrade gracefully to
`{ "data": [] }` and **detail** endpoints return `404` — they never 500 for a missing
DB.

---

## Endpoints

### App bootstrap endpoints

The two aggregate endpoints an app hits first at launch.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/config`](#get-apiv1config) | open | App bootstrap: identity, stream, social links |
| `GET` | [`/api/v1/home`](#get-apiv1home) | open | Aggregate home-screen feed in one request |

### Content endpoints 

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/programs`](#get-apiv1programs) | open | Active program roster, each flagged on-air |
| `GET` | [`/api/v1/programs/{slug}`](#get-apiv1programsslug) | open | One program with weekly schedule & broadcasters |
| `GET` | [`/api/v1/broadcasters`](#get-apiv1broadcasters) | open | Active broadcaster roster |
| `GET` | [`/api/v1/broadcasters/{slug}`](#get-apiv1broadcastersslug) | open | One broadcaster plus the programs they present |
| `GET` | [`/api/v1/news`](#get-apiv1news) | open | Grouped news preview, or one source paginated |
| `GET` | [`/api/v1/news/{slug}`](#get-apiv1newsslug) | open | One `hot_release` article with gallery & related |
| `GET` | [`/api/v1/podcasts`](#get-apiv1podcasts) | open | Published podcasts, paginated |
| `GET` | [`/api/v1/podcasts/{slug}`](#get-apiv1podcastsslug) | open | One podcast with series & broadcasters |
| `GET` | [`/api/v1/podcast-series`](#get-apiv1podcast-series) | open | Active podcast series list |
| `GET` | [`/api/v1/about`](#get-apiv1about) | open | About-page banner, segments & broadcaster preview |

### Live endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/now-playing`](#get-apiv1now-playing) | open | Stream now-playing metadata & on-air program |
| `GET` | [`/api/v1/schedule/today`](#get-apiv1scheduletoday) | open | Today's schedule with live on-air/progress state |
| `GET` | [`/api/v1/schedule/current`](#get-apiv1schedulecurrent) | open | The currently on-air program |
| `GET` | [`/api/v1/tiktok/live`](#get-apiv1tiktoklive) | open | TikTok live status |

### Advertising endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/ads`](#get-apiv1ads) | open | Ad banners for a page, by placement slot |

> **Note:** the Connect chat is no longer served by this API — it now runs directly
> against Firebase Realtime Database (shared with the mobile app), so there are no
> `/api/v1/connect/*` endpoints.

---

## App bootstrap endpoints

Both are `GET`, cached `public, max-age=60` — the endpoints an app calls at launch.

### `GET /api/v1/config`

App bootstrap: station identity, stream URL, and social links.

```json
{
  "station": { "name": "Classy 103.4 FM", "slogan": "…" },
  "stream_url": "https://c4.siar.us:10340/stream.mp3",
  "social": { "instagram": "https://…", "youtube": "https://…" }
}
```

`social` keys are platform names as configured in the admin panel.

### `GET /api/v1/home`

Aggregate home-screen feed in a single request.

```json
{
  "hero":     [ /* heroSlide objects */ ],
  "on_air":   { /* scheduleRow, or null */ },
  "programs": [ /* program objects, up to 6 */ ],
  "news":     [ /* newsGroup objects, up to 4 items each */ ]
}
```

See [`heroSlide`](#heroslide).

---

## Content endpoints

All are `GET` and cached `public, max-age=60`.

### `GET /api/v1/programs`

Active program roster, each flagged with whether it is on air now.

```json
{ "data": [
  {
    "title": "Morning Show",
    "slug": "morning-show",
    "description": "…",
    "image_url": "https://classyfm.co.id/uploads/abc.jpg",
    "url": "https://classyfm.co.id/program/morning-show",
    "on_air": true
  }
] }
```

See [`program`](#program) for the object fields.

### `GET /api/v1/programs/{slug}`

One program with its full weekly schedule and broadcasters. `404` if the slug is
unknown or the program is inactive.

```json
{
  "title": "Morning Show",
  "slug": "morning-show",
  "description": "…",
  "image_url": "https://classyfm.co.id/uploads/abc.jpg",
  "url": "https://classyfm.co.id/program/morning-show",
  "on_air": true,
  "schedule": [
    { "from_day": 1, "to_day": 5, "start_time": "06:00", "end_time": "09:00", "host": "Anda, Yeni" }
  ],
  "broadcasters": [ /* broadcaster objects */ ]
}
```

`schedule[]` is a [`scheduleGroup`](#schedulegroup); `broadcasters[]` are
[`broadcaster`](#broadcaster) objects.

### `GET /api/v1/broadcasters`

Active broadcaster roster. `on_air` is `true` if any program the broadcaster presents
is airing now.

```json
{ "data": [ /* broadcaster objects */ ] }
```

### `GET /api/v1/broadcasters/{slug}`

One broadcaster plus the programs they present. `404` if unknown.

```json
{
  "name": "Anda",
  "slug": "anda",
  "role": "Announcer",
  "photo_url": "https://classyfm.co.id/uploads/anda.jpg",
  "bio": "…",
  "birth_place": "Padang",
  "birth_date": "…",
  "instagram": "…",
  "twitter": "…",
  "facebook": "…",
  "url": "https://classyfm.co.id/broadcasters/anda",
  "on_air": false,
  "programs": [ /* program objects */ ]
}
```

### `GET /api/v1/news`

Two modes:

**Without `source`** (or with an unrecognized one) — grouped preview, up to 6 items per
source group:

```json
{ "data": [
  { "source": "klikpositif", "label": "KlikPositif", "items": [ /* news items */ ] },
  { "source": "katasumbar",  "label": "KataSumbar",  "items": [ … ] },
  { "source": "hot_release", "label": "Hot Release", "items": [ … ] },
  { "source": "youtube",     "label": "YouTube",     "items": [ … ] }
] }
```

**With a valid `source`** — that source's items, paginated:

| Query param | Notes |
|-------------|-------|
| `source` | One of `youtube`, `klikpositif`, `katasumbar`, `hot_release`. <br>Invalid values fall back to grouped mode. |
| `page` | 1-based, page size 12. |

```json
{ "data": [ /* news items */ ], "meta": { "page": 1, "total_pages": 4, "total": 42 } }
```

Items are [`newsItem`](#newsitem) objects. Note: `hot_release` items link to the
on-site `/news/{slug}`; other (aggregated) sources keep their external `url`.

### `GET /api/v1/news/{slug}`

One `hot_release` article, with split paragraphs, a mid-article image gallery, and
related articles. `404` if unknown (only `hot_release` articles have on-site detail).

```json
{
  "source": "hot_release",
  "source_label": "Hot Release",
  "title": "…",
  "slug": "…",
  "excerpt": "…",
  "image_url": "https://…",
  "url": "https://classyfm.co.id/news/…",
  "published_at": "2026-08-09T10:00:00Z",
  "is_featured": true,
  "paragraphs": ["…", "…"],
  "gallery_index": 3,
  "middle_images": ["https://…", "https://…"],
  "related": [ /* news items */ ]
}
```

`gallery_index` is the paragraph index after which the `middle_images` gallery should
be inserted.

### `GET /api/v1/podcasts`

Published podcasts, newest first, paginated.

| Query param | Notes |
|-------------|-------|
| `series` | Optional series slug filter. Invalid slugs are ignored (returns all). |
| `page` | 1-based, page size 12. |

```json
{ "data": [ /* podcast objects */ ], "meta": { "page": 1, "total_pages": 2, "total": 18 } }
```

### `GET /api/v1/podcasts/{slug}`

One podcast with its series name and broadcasters. `404` if unknown.

```json
{
  "title": "…",
  "slug": "…",
  "description": "…",
  "spotify_url": "https://open.spotify.com/…",
  "thumb_url": "https://…",
  "series_name": "…",
  "url": "https://classyfm.co.id/podcast/…",
  "broadcasters": [ /* broadcaster objects */ ]
}
```

### `GET /api/v1/podcast-series`

The **active** podcast series (for a filter picker feeding `?series=`). Series marked
inactive in the admin panel are omitted.

```json
{ "data": [ { "name": "…", "slug": "…" } ] }
```

### `GET /api/v1/about`

About-page content: banner, text segments, and a broadcaster preview (max 8).

```json
{
  "banner": {
    "media_type": "video",
    "image_url": "https://…",
    "video_url": "https://youtu.be/…",
    "embed_url": "https://www.youtube.com/embed/…"
  },
  "segments": [ { "segment": "profile", "title": "…", "body": "…" } ],
  "broadcasters": [ /* broadcaster objects */ ]
}
```

`banner.embed_url` is present only when `media_type` is `"video"` and the video URL is
a resolvable YouTube link. `segment` is one of `profile`, `music`, `audience`.

---

## Live endpoints

All are `GET`, cached `no-store` — live stream state and schedule progress.

### `GET /api/v1/now-playing`

Stream now-playing metadata, plus the on-air program when the stream is live.

```json
{
  "artist": "…",
  "song": "…",
  "has_song": true,
  "cover_url": "https://…",
  "live": true,
  "program": { /* scheduleRow, present only when live and a program is on air */ }
}
```

See [`scheduleRow`](#schedulerow). (The Shoutcast listener count is deliberately not
exposed here — it is admin-only.)

### `GET /api/v1/schedule/today`

Today's full schedule with live on-air/progress state.

```json
{ "data": [ /* scheduleRow objects */ ] }
```

### `GET /api/v1/schedule/current`

The currently on-air program as a single [`scheduleRow`](#schedulerow) object, or
`{ "on_air": false }` when nothing is airing.

### `GET /api/v1/tiktok/live`

```json
{ "live": true, "title": "…" }
```

### Playing the live stream in an app

The audio stream is a **direct external Shoutcast MP3** — it lives on the Shoutcast host, **not** on `classyfm.co.id`, and this API
never proxies or redirects the audio. The app plays it **directly**:

1. **Bootstrap the URL.** Read `stream_url` from [`/api/v1/config`](#get-apiv1config)
   once at startup and hand it to the device's native audio player. It is a plain,
   unauthenticated streaming MP3 (Shoutcast) — no headers, no token, no proxy. Prefer
   reading it from `config` over hardcoding it, so the Shoutcast host can be moved
   server-side without an app release.
2. **Drive now-playing from a poll loop — via this API, never Shoutcast directly.** Poll
   [`/api/v1/now-playing`](#get-apiv1now-playing) on a timer to update the now-playing
   UI (and any lock-screen / notification metadata). The API fetches and caches the
   track metadata and `live` state from the Shoutcast server for you, so the app avoids
   CORS and doesn't hammer the Shoutcast box — **do not scrape the Shoutcast endpoints
   yourself.** The response is `no-store` but the server refreshes its upstream metadata
   only every ~12 s, so **polling faster than ~15 s gains nothing** — settle on roughly
   a 15 s interval while the player is active, and pause polling when it is stopped or
   backgrounded without audio.
   - Show `artist` + `song` when `has_song` is `true`; when `false` there is no track
     metadata (station ID / no title) — fall back to the station name.
   - Use `cover_url` for artwork, but it is best-effort and may be `""` — fall back to a
     bundled placeholder or the on-air program image.
3. **Handle live vs. off-air.** Switch the UI between "on air" and "off air" on the
   `live` flag. When `live` is `true`, the optional `program` object (a
   [`scheduleRow`](#schedulerow)) gives the current show, host, and `progress` (0–100);
   [`/api/v1/schedule/current`](#get-apiv1schedulecurrent) returns the same thing
   standalone, and [`/api/v1/schedule/today`](#get-apiv1scheduletoday) backs an "up
   next" list.

### When the backend API is unreachable

Because the audio is a **direct Shoutcast MP3** and this API never proxies it, backend
downtime (network error, timeout, `5xx`) does **not** interrupt playback — only the
*metadata* around it. Keep the audio running and degrade only the now-playing chrome:

1. **Persist `config`.** Cache the last successful [`/api/v1/config`](#get-apiv1config)
   response (at minimum `stream_url`, plus station name/slogan and social links) in local
   storage. At launch, start playback from the cached `stream_url` even when `config`
   can't be re-fetched. There is **no bundled/hardcoded stream URL** — the app never
   invents one. On a true cold start (first-ever launch, nothing cached, and the API
   unreachable) there is no URL to play: show a "stream unavailable / retry" state and
   fetch `config` again once connectivity returns.
2. **Keep audio alive when `now-playing` fails.** A failed/timed-out/`5xx`
   [`/api/v1/now-playing`](#get-apiv1now-playing) poll is a metadata gap, not a stream
   failure — never stop or reset the player because of it. Hold the last-known
   `artist`/`song` and `live` state, or fall back to the station name + bundled
   placeholder art.
3. **Back off, then recover.** On repeated poll failures, widen the interval (e.g.
   exponential backoff up to ~60 s) instead of hammering. On the next successful poll,
   resume the normal ~15 s cadence, refresh the now-playing UI, and re-cache `config`.
   No user action or app restart should be required.

A live listener count is intentionally not available to apps.

---

## Advertising endpoints

`GET`, cached `public, max-age=60`.

### `GET /api/v1/ads`

Ad banners for a page, grouped into `top` and `bottom` placement slots.

| Query param | Notes |
|-------------|-------|
| `page` | Target page key. Invalid keys return `400`. Omitted returns only banners targeted at every page. |

Valid `page` keys: `home`, `about`, `program`, `program_detail`, `live`, `news`,
`news_detail`, `broadcasters`, `broadcaster_detail`.

```json
{
  "top": {
    "slideshow": false,
    "rotate_ms": 6000,
    "placeholder": false,
    "placeholder_text": "",
    "banners": [
      {
        "image_url": "https://classyfm.co.id/uploads/ad.jpg",
        "link_url": "https://sponsor.example",
        "alt": "…",
        "title": "…"
      }
    ]
  },
  "bottom": { /* same shape */ }
}
```

Each slot's `banners` is an array (empty when the slot has none). `slideshow` tells the
client to rotate banners every `rotate_ms` rather than stack them; `placeholder` (with
`placeholder_text`) says an empty slot should hold its space rather than collapse.

---

## Legal pages

The **Privacy Policy** and **Terms & Conditions** are served as static **HTML pages**
(not JSON), under the site origin `https://classyfm.co.id`. Open them in the device
browser or an in-app WebView — do not try to parse them as API responses.

| Page | URL |
|------|-----|
| Privacy Policy | `https://classyfm.co.id/privacy-policy` |
| Terms & Conditions | `https://classyfm.co.id/terms-and-conditions` |

The URLs are stable — link to them from a Settings/Legal screen. The pages are in
Indonesian and their prose may change over time, so link live rather than caching a
local copy.

---

## Object reference

Fields marked *(optional)* are omitted from the JSON when empty.

### program

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | |
| `slug` | string | |
| `description` | string | *(optional)* |
| `image_url` | string | *(optional)* absolute URL |
| `url` | string | on-site program URL |
| `on_air` | bool | airing now |

### scheduleGroup

| Field | Type | Notes |
|-------|------|-------|
| `from_day` | int | 0 = Sunday … 6 = Saturday |
| `to_day` | int | end of the day range (same numbering) |
| `start_time` | string | `HH:MM` |
| `end_time` | string | `HH:MM` |
| `host` | string | *(optional)* effective broadcaster set, e.g. `"Anda, Yeni"` |

### broadcaster

| Field | Type | Notes |
|-------|------|-------|
| `name` | string | |
| `slug` | string | |
| `role` | string | *(optional)* |
| `photo_url` | string | *(optional)* absolute URL |
| `bio` | string | *(optional)* |
| `birth_place` | string | *(optional)* |
| `birth_date` | string | *(optional)* |
| `instagram` | string | *(optional)* |
| `twitter` | string | *(optional)* |
| `facebook` | string | *(optional)* |
| `url` | string | on-site broadcaster URL |
| `on_air` | bool | presenting an airing program now |

### newsItem

| Field | Type | Notes |
|-------|------|-------|
| `source` | string | `youtube` \| `klikpositif` \| `katasumbar` \| `hot_release` |
| `source_label` | string | human-readable source name |
| `title` | string | |
| `slug` | string | *(optional)* present for `hot_release` |
| `excerpt` | string | *(optional)* |
| `image_url` | string | *(optional)* |
| `url` | string | on-site for `hot_release`, external otherwise |
| `published_at` | string | RFC 3339 timestamp |
| `is_featured` | bool | *(optional)* |

### newsGroup

| Field | Type | Notes |
|-------|------|-------|
| `source` | string | source key |
| `label` | string | display label |
| `items` | newsItem[] | |

### podcast

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | |
| `slug` | string | |
| `description` | string | *(optional)* |
| `spotify_url` | string | *(optional)* |
| `thumb_url` | string | *(optional)* absolute URL |
| `series_name` | string | *(optional)* |
| `broadcaster` | string | *(optional)* list-view only |
| `url` | string | on-site podcast URL |

### scheduleRow

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | program title |
| `slug` | string | *(optional)* |
| `host` | string | *(optional)* |
| `image` | string | *(optional)* absolute URL |
| `start_time` | string | `HH:MM` |
| `end_time` | string | `HH:MM` |
| `on_air` | bool | |
| `progress` | int | 0–100, % through the slot |
| `ended` | bool | slot already finished today |

### heroSlide

| Field | Type | Notes |
|-------|------|-------|
| `href` | string | *(optional)* link target |
| `target_blank` | bool | *(optional)* open in new tab |
| `image_url` | string | *(optional)* |
| `badge_label` | string | *(optional)* |
| `title` | string | |
| `excerpt` | string | *(optional)* |
| `date` | string | *(optional)* RFC 3339 timestamp |
