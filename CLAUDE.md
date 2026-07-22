# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

ClassyFM — the website + admin panel for an Indonesian radio station (Classy 103.4 FM,
Padang). Go server rendering HTML templates, MySQL storage, a public site (home,
programs, live stream, news, broadcasters) and an admin panel (session-auth CMS for
programs, broadcasters, news/hot-release, feed sources, users). Ships as a single
self-contained binary with templates and static assets embedded via `go:embed`.

## Commands

```
make run              # go run ./cmd/server (loads .env if present)
make build            # builds Tailwind CSS then the binary to bin/classyfm
make css              # rebuild web/static/css/app.css from tailwind.css (minified)
make css-watch         # rebuild CSS on change during frontend work
make sqlc             # regenerate internal/db/sqlc from internal/db/queries + migrations
make tidy             # go mod tidy
make create-admin email=... password=... name="..."   # bootstrap an admin user
make migrate-up / migrate-down / migrate-create name=x  # golang-migrate against DATABASE_URL
```

Standard Go tooling applies (`go build ./...`, `go vet ./...`, `go test ./...`); there is
currently no test suite. `gofmt`/`go vet` before committing.

`:8080` on the deploy host is **always production** — see deploy/ below before running
anything that binds to it locally. Copy `.env.example` to `.env` for local dev; the DSN
in `.env` uses MySQL's `tcp(host:port)` syntax, which breaks plain `source .env` (parens
are shell metacharacters) — use `make run`/Make targets rather than sourcing it directly.

## Architecture

**Entry point / wiring**: `cmd/server/main.go` builds `Config`, opens the DB pool
(optional — the server boots without one, disabling DB-backed routes), builds the
`render.Renderer`, `radio.Service`, `feeds.Worker`, and the two handler groups
(`internal/handlers/public`, `internal/handlers/admin`), then wires everything into a
single `chi.Router` in `newRouter`. Read this file first when tracing how a request
reaches a handler.

**Config** (`internal/config`): flat struct loaded from env vars with defaults, never
fails to load (`config.Load()` has no error return) so the app can boot in degraded mode
during early setup. `.env` is picked up by the Makefile, not by the Go binary itself.

**DB layer**: raw SQL in `internal/db/queries/*.sql`, versioned schema in
`internal/db/migrations/*.up.sql`/`*.down.sql` (golang-migrate, sequential numbering),
generated type-safe Go in `internal/db/sqlc` via `sqlc` (`sqlc.yaml`, `make sqlc`). Never
hand-edit `internal/db/sqlc/*.go` — change the `.sql` and regenerate. MySQL `TIME`
columns are mapped to Go `string` ("HH:MM:SS"), not `time.Time`, because
go-sql-driver/mysql returns them as raw bytes even with `parseTime=true`; see the
comment in `sqlc.yaml`.

**Rendering** (`internal/render`): each page template is parsed together with
`layouts/*.html` and `partials/*.html` into one `*template.Template` per page, cached by
`"public/<name>"` / `"admin/<name>"`, and executed against `"base"` or `"admin-base"`
depending on prefix. In dev the cache rebuilds every request (`reload=true`) so template
edits show up without a restart; in prod it's built once at startup. Template helpers
(`fmtDate`, `weekday`, `listImage`, `splitParagraphs`, etc.) live in `defaultFuncs()` —
add new cross-template helpers there rather than computing them in handlers.

**Auth** (`internal/middleware/auth.go`, `csrf.go`, `ratelimit.go`, `virtual.go`):
session cookie `classyfm_admin_session` is checked against two paths — a normal
DB-backed session (`sessions` table, FK to `users`) via `Auth`, or a self-contained
HMAC-signed "virtual" token (`virtual.go`) for the break-glass root login
(`root@local.system`), which has no `users` row and is verified without touching the DB.
`Auth` never blocks; `RequireAuth`/`RequireRole` gate specific route groups. All
`/admin/*` routes get `Auth` + `CSRF`; only the inner group requires auth, so
login/logout/forgot-password stay reachable.

**Feed aggregation** (`internal/feeds`): `Worker` polls a list of `Source`
implementations (YouTube RSS, WordPress `/feed/` endpoints per outlet) on `FeedInterval`
and upserts into `news_items`, tracking per-source status in `feed_sources` so one
broken source doesn't block the others. Image resolution
(`resolveImagesForUpsert`/`PreferImage`) deliberately prefers an already-upgraded stored
image over a freshly re-derived one, since re-derivation involves a live network probe
that can fail or regress transiently — don't "simplify" this to always overwrite.
Admin's "Refresh now" button calls `Worker.RunOnce` directly.

**Uploads**: user-uploaded files (program/broadcaster images) live on real disk at
`UploadDir` (served at `/uploads/*`, long immutable cache since filenames are
per-upload-random and never mutated in place) — distinct from `/static/*`, which is
`go:embed`'d and cache-busted via a per-process-start `staticVer` template func instead
of relying on file mtimes (embedded files have zero ModTime).

**Naming note**: the public-facing entity historically called "classiers" was renamed to
"broadcasters" (migration `0016_rename_classiers_to_broadcasters`) — routes, templates,
and Go identifiers now consistently say broadcasters; old `/classiers` URLs 404 with no
redirect.

**cmd/ tools**: `createadmin` (bootstrap a user), `importhotrelease` (scrape/import
historical "hot release" news articles from classyfm.co.id, re-runnable for new
articles), `upgradeimages` (batch hi-res image resolution for existing rows),
`striphtml` (one-off data cleanup utility).

**deploy/**: systemd unit (`classyfm.service`), backup timer/service + `backup.sh`, and
an nginx vhost (`classyfm.remorac.com`) — the production host sits behind Cloudflare, so
CSS/JS deploys need a manual cache purge to verify, not just a redeploy.
