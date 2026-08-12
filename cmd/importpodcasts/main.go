// Command importpodcasts backfills the podcasts table from the legacy classy
// database dump (backup-old-db.sql). The old site had no podcast entity: episodes
// were authored as ordinary CMS posts whose body embedded a Spotify episode via an
// <iframe src="...open.spotify.com/embed/episode/<id>...">. This tool finds those
// posts, groups them into podcast_series by their title prefix, and inserts one
// podcasts row each, resolving artwork through Spotify's oEmbed endpoint exactly like
// the admin handler does.
//
// It is a one-off migration tool, not a recurring worker. Dedupe is by canonical
// Spotify URL (the episode id is stable), checked against whatever is already in the
// database, so the command is safe to interrupt and re-run: a re-run skips every
// episode already imported.
//
// Defaults to a dry run (-dry-run=true); pass -dry-run=false to actually write rows
// and fetch thumbnails.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/config"
	"github.com/classyfm/classyfm/internal/db"
	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/spotify"
)

func main() {
	file := flag.String("file", "backup-old-db.sql", "path to the legacy MySQL dump to read the post table from")
	envFile := flag.String("env", ".env", "path to a .env file to load DATABASE_DSN etc. from (skipped if absent)")
	dryRun := flag.Bool("dry-run", true, "print what would be imported without writing rows or fetching thumbnails")
	limit := flag.Int("limit", 0, "stop after importing this many new episodes (0 = unlimited)")
	delay := flag.Duration("delay", 300*time.Millisecond, "delay between Spotify oEmbed requests")
	flag.Parse()

	// Unlike the server (which gets its env from systemd/Makefile), this one-off command
	// is run by hand, so it loads .env itself. A real line parser handles the tcp(host:port)
	// DSN that plain `source .env` chokes on. Already-set env vars still win.
	if err := config.LoadEnvFile(*envFile); err != nil {
		log.Fatalf("load %s: %v", *envFile, err)
	}

	cfg := config.Load()
	if cfg.DatabaseDSN == "" {
		log.Fatal("DATABASE_DSN is not set")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("connect to mysql: %v", err)
	}
	defer pool.Close()

	q := sqlc.New(pool)

	posts, err := parsePosts(*file)
	if err != nil {
		log.Fatalf("parse %s: %v", *file, err)
	}
	log.Printf("parsed %d post rows from %s", len(posts), *file)

	imp := &importer{
		q:         q,
		pool:      pool,
		client:    &http.Client{Timeout: 6 * time.Second},
		delay:     *delay,
		dryRun:    *dryRun,
		limit:     *limit,
		seriesIDs: make(map[string]uint64),
		seenURLs:  make(map[string]bool),
		seenSlugs: make(map[string]bool),
	}
	if err := imp.loadExisting(ctx); err != nil {
		log.Fatalf("load existing podcasts: %v", err)
	}
	log.Printf("loaded %d existing podcasts for dedupe", len(imp.seenURLs))

	if err := imp.run(ctx, posts); err != nil {
		log.Fatalf("import: %v", err)
	}
}

// post is the subset of the legacy `post` columns this tool needs.
type post struct {
	title            string
	content          string
	status           string
	publishDateStart string
	createdAt        string
	deletedAt        string
}

type importer struct {
	q      *sqlc.Queries
	pool   *sql.DB
	client *http.Client
	delay  time.Duration
	dryRun bool
	limit  int

	seriesIDs map[string]uint64 // series slug -> id (resolved/created lazily)
	seenURLs  map[string]bool   // canonical spotify_url already present (DB + this run)
	seenSlugs map[string]bool   // podcast slug already present (DB + this run)

	imported int
	skipped  int
}

// loadExisting preloads the spotify_url and slug of every podcast already stored, so a
// re-run skips episodes it imported before and so generated slugs never collide with
// existing rows.
func (imp *importer) loadExisting(ctx context.Context) error {
	rows, err := imp.pool.QueryContext(ctx, "SELECT slug, spotify_url FROM podcasts")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var slug, url string
		if err := rows.Scan(&slug, &url); err != nil {
			return err
		}
		imp.seenSlugs[slug] = true
		imp.seenURLs[url] = true
	}
	return rows.Err()
}

func (imp *importer) run(ctx context.Context, posts []post) error {
	for _, p := range posts {
		if imp.limit > 0 && imp.imported >= imp.limit {
			break
		}
		if p.deletedAt != "" {
			continue // soft-deleted post
		}
		epID := episodeID(p.content)
		if epID == "" {
			continue // not a podcast post
		}
		spotifyURL := "https://open.spotify.com/episode/" + epID
		if spotify.EmbedURL(spotifyURL) == "" {
			log.Printf("skip: unusable spotify url %q (%s)", spotifyURL, p.title)
			continue
		}
		if imp.seenURLs[spotifyURL] {
			imp.skipped++
			continue // already imported (or a duplicate embed within the dump)
		}

		rule := matchSeries(p.title)
		if rule == nil {
			log.Printf("skip: title matches no known series: %q", p.title)
			continue
		}
		title := rule.cleanTitle(p.title)
		slug := imp.uniqueSlug(slugify(title))
		desc := cleanDescription(p.content)
		createdAt := parseLegacyTime(p.publishDateStart, p.createdAt)

		imp.seenURLs[spotifyURL] = true

		if imp.dryRun {
			log.Printf("[dry-run] %-28s | %s", rule.name, title)
			imp.imported++
			continue
		}

		seriesID, err := imp.seriesID(ctx, rule)
		if err != nil {
			return fmt.Errorf("resolve series %q: %w", rule.name, err)
		}
		thumb := imp.fetchThumb(ctx, spotifyURL)

		res, err := imp.q.CreatePodcast(ctx, sqlc.CreatePodcastParams{
			Title:       title,
			Slug:        slug,
			SeriesID:    seriesID,
			Description: desc,
			SpotifyUrl:  spotifyURL,
			ThumbUrl:    thumb,
			IsPublished: true,
		})
		if err != nil {
			return fmt.Errorf("insert %q: %w", title, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("last insert id for %q: %w", title, err)
		}
		if err := imp.q.SetPodcastCreatedAt(ctx, sqlc.SetPodcastCreatedAtParams{
			CreatedAt: createdAt,
			ID:        uint64(id),
		}); err != nil {
			return fmt.Errorf("set created_at for %q: %w", title, err)
		}
		log.Printf("imported #%d %-28s | %s", id, rule.name, title)
		imp.imported++
		time.Sleep(imp.delay)
	}

	if imp.dryRun {
		log.Printf("dry run complete: %d episodes would be imported, %d already present", imp.imported, imp.skipped)
	} else {
		log.Printf("done: %d imported, %d already present", imp.imported, imp.skipped)
	}
	return nil
}

// seriesID resolves the series by slug, creating it (with the rule's canonical name and
// sort order) the first time it is seen. Results are memoized for the run.
func (imp *importer) seriesID(ctx context.Context, rule *seriesRule) (uint64, error) {
	if id, ok := imp.seriesIDs[rule.slug]; ok {
		return id, nil
	}
	s, err := imp.q.GetPodcastSeriesBySlug(ctx, rule.slug)
	if err == nil {
		imp.seriesIDs[rule.slug] = s.ID
		return s.ID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := imp.q.CreatePodcastSeries(ctx, sqlc.CreatePodcastSeriesParams{
		Name:      rule.name,
		Slug:      rule.slug,
		SortOrder: rule.sortOrder,
	})
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	log.Printf("created series %q (id %d)", rule.name, id)
	imp.seriesIDs[rule.slug] = uint64(id)
	return uint64(id), nil
}

// fetchThumb resolves the episode artwork, best-effort: a failure just stores no
// thumbnail, exactly like the admin handler.
func (imp *importer) fetchThumb(ctx context.Context, spotifyURL string) sql.NullString {
	oe, err := spotify.FetchOEmbed(ctx, imp.client, spotifyURL)
	if err != nil || oe.ThumbnailURL == "" {
		if err != nil {
			log.Printf("thumbnail fetch failed for %s: %v", spotifyURL, err)
		}
		return sql.NullString{}
	}
	return sql.NullString{String: oe.ThumbnailURL, Valid: true}
}

// uniqueSlug returns base, or base-2, base-3, ... if the slug is already taken by an
// existing or already-imported podcast, and records the result as taken.
func (imp *importer) uniqueSlug(base string) string {
	if base == "" {
		base = "podcast"
	}
	candidate := base
	for i := 2; imp.seenSlugs[candidate]; i++ {
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	imp.seenSlugs[candidate] = true
	return candidate
}

// --- series classification ------------------------------------------------------

type seriesRule struct {
	name      string
	slug      string
	sortOrder int32
	prefix    *regexp.Regexp // anchored, case-insensitive; matches "<series><sep>"
}

// cleanTitle strips the matched "<series><separator>" prefix so the stored title is not
// redundant with the separately-displayed series. It falls back to the full title if
// stripping would leave nothing.
func (r *seriesRule) cleanTitle(title string) string {
	stripped := strings.TrimSpace(r.prefix.ReplaceAllString(title, ""))
	if stripped == "" {
		return strings.TrimSpace(title)
	}
	return stripped
}

// seriesRules are matched in order; the first whose prefix matches the title wins. The
// three series already seeded by migration 0030 keep sort_order 1-3; the rest extend it.
var seriesRules = []*seriesRule{
	{"Talkshow Bebas Pusing", "talkshow-bebas-pusing", 1, regexp.MustCompile(`(?i)^\s*(?:talkshow\s+)?bebas pusing\s*[:\-]\s*`)},
	{"Communitalk with Yeni Maiasnita", "communitalk-with-yeni-maiasnita", 2, regexp.MustCompile(`(?i)^\s*communitalk(?:\s+with yeni maiasnita)?\s*[:\-]\s*`)},
	{"Special Talkshow", "special-talkshow", 3, regexp.MustCompile(`(?i)^\s*special talkshow\s*[:\-]\s*`)},
	{"The Art of Leadership", "the-art-of-leadership", 4, regexp.MustCompile(`(?i)^\s*the art of leadership\s*[:\-]\s*`)},
	{"Parliament Talk", "parliament-talk", 5, regexp.MustCompile(`(?i)^\s*parliament talk\s*[:\-]\s*`)},
	{"Semen Padang Journal", "semen-padang-journal", 6, regexp.MustCompile(`(?i)^\s*semen padang journal\s*[:\-]\s*`)},
	{"Kanal BRI", "kanal-bri", 7, regexp.MustCompile(`(?i)^\s*kanal bri\s*[:\-]\s*`)},
	{"Podcast Bank Indonesia", "podcast-bank-indonesia", 8, regexp.MustCompile(`(?i)^\s*podcast bank indonesia\s*[:\-]\s*`)},
}

func matchSeries(title string) *seriesRule {
	for _, r := range seriesRules {
		if r.prefix.MatchString(title) {
			return r
		}
	}
	return nil
}

// --- content helpers ------------------------------------------------------------

var (
	episodeRe = regexp.MustCompile(`open\.spotify\.com/embed/episode/([A-Za-z0-9]+)`)
	iframeRe  = regexp.MustCompile(`(?is)<iframe.*?</iframe>`)
	tagRe     = regexp.MustCompile(`<[^>]*>`)
	spaceRe   = regexp.MustCompile(`\s+`)
	slugRe    = regexp.MustCompile(`[^a-z0-9]+`)
)

// episodeID returns the first embedded Spotify episode id in a post body, or "".
func episodeID(content string) string {
	m := episodeRe.FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	return m[1]
}

// cleanDescription turns a post body into plain text: drop the Spotify iframe, strip
// remaining tags, decode HTML entities, and collapse whitespace. Many posts have no
// prose beyond the player, which yields "" (the column is NOT NULL, empty is fine).
func cleanDescription(content string) string {
	s := iframeRe.ReplaceAllString(content, " ")
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// slugify mirrors internal/handlers/admin/podcasts.go: lowercase, non-word runs to
// hyphens, trimmed.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// parseLegacyTime parses a "YYYY-MM-DD HH:MM:SS" dump value, trying primary then
// fallback, and defaults to now if neither is usable.
func parseLegacyTime(primary, fallback string) time.Time {
	for _, v := range []string{primary, fallback} {
		if t, err := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(v)); err == nil {
			return t
		}
	}
	return time.Now()
}

// --- dump parser ----------------------------------------------------------------

// post column order in the dump's INSERT statement.
const (
	colTitle            = 1
	colContent          = 3
	colStatus           = 7
	colPublishDateStart = 8
	colCreatedAt        = 11
	colDeletedAt        = 14
	colCount            = 15
)

// cp1252High maps the 0x80-0x9F byte range, where Windows-1252 differs from Latin-1,
// to its Unicode code points (e.g. 0x93/0x94 curly quotes, 0x96/0x97 dashes). The dump
// declares SET NAMES latin1 but its text columns hold cp1252 bytes; storing them raw
// into a utf8mb4 column fails, so we decode to proper Unicode first.
var cp1252High = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
}

// decodeCP1252 converts Windows-1252 bytes to a UTF-8 string. Structural bytes used by
// the parser (quotes, parens, backslashes) are ASCII and unaffected; only high bytes
// are remapped, so decoding the whole dump up front is safe.
func decodeCP1252(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b))
	for _, c := range b {
		switch {
		case c < 0x80 || c >= 0xA0:
			sb.WriteRune(rune(c)) // ASCII and 0xA0-0xFF match Unicode 1:1 (Latin-1)
		default:
			sb.WriteRune(cp1252High[c-0x80])
		}
	}
	return sb.String()
}

// parsePosts reads the `post` INSERT statement out of a mysqldump file and returns the
// columns this tool needs. The statement is a single very large line of
// "(v,v,...),(v,v,...)" tuples with single-quoted, backslash-escaped strings.
func parsePosts(path string) ([]post, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data := decodeCP1252(raw)

	marker := "INSERT INTO `post` VALUES "
	start := strings.Index(data, marker)
	if start < 0 {
		return nil, fmt.Errorf("no `post` INSERT found")
	}
	start += len(marker)
	end := strings.Index(data[start:], "/*!40000 ALTER TABLE `post` ENABLE KEYS")
	if end < 0 {
		return nil, fmt.Errorf("no end marker for `post` INSERT found")
	}
	blob := strings.TrimSpace(data[start : start+end])
	blob = strings.TrimSuffix(blob, ";")

	var posts []post
	i, n := 0, len(blob)
	for i < n {
		// Skip separators between tuples.
		for i < n && (blob[i] == ' ' || blob[i] == ',' || blob[i] == '\n' || blob[i] == '\r') {
			i++
		}
		if i >= n {
			break
		}
		if blob[i] != '(' {
			i++
			continue
		}
		vals, next, err := parseTuple(blob, i)
		if err != nil {
			return nil, err
		}
		i = next
		if len(vals) != colCount {
			continue // shape we do not recognize; skip defensively
		}
		posts = append(posts, post{
			title:            vals[colTitle],
			content:          vals[colContent],
			status:           vals[colStatus],
			publishDateStart: vals[colPublishDateStart],
			createdAt:        vals[colCreatedAt],
			deletedAt:        vals[colDeletedAt],
		})
	}
	return posts, nil
}

// parseTuple reads one "(v,v,...)" tuple starting at s[i]=='(' and returns the decoded
// values, the index just past the closing ')', or an error. NULL becomes "".
func parseTuple(s string, i int) ([]string, int, error) {
	if s[i] != '(' {
		return nil, i, fmt.Errorf("expected '(' at %d", i)
	}
	i++
	var vals []string
	n := len(s)
	for {
		if i >= n {
			return nil, i, fmt.Errorf("unterminated tuple")
		}
		if s[i] == '\'' {
			str, next := parseString(s, i)
			vals = append(vals, str)
			i = next
		} else {
			// Number or NULL, up to the next ',' or ')'.
			j := i
			for j < n && s[j] != ',' && s[j] != ')' {
				j++
			}
			tok := strings.TrimSpace(s[i:j])
			if tok == "NULL" {
				tok = ""
			}
			vals = append(vals, tok)
			i = j
		}
		if i >= n {
			return nil, i, fmt.Errorf("unterminated tuple")
		}
		if s[i] == ',' {
			i++
			continue
		}
		if s[i] == ')' {
			return vals, i + 1, nil
		}
		return nil, i, fmt.Errorf("unexpected byte %q at %d", s[i], i)
	}
}

// parseString reads a single-quoted, backslash-escaped MySQL string starting at
// s[i]=='\” and returns the decoded value and the index just past the closing quote.
func parseString(s string, i int) (string, int) {
	i++ // opening quote
	var b strings.Builder
	n := len(s)
	for i < n {
		c := s[i]
		if c == '\\' && i+1 < n {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '0':
				b.WriteByte(0)
			default:
				b.WriteByte(s[i+1]) // \' \" \\ and any other -> literal next byte
			}
			i += 2
			continue
		}
		if c == '\'' {
			return b.String(), i + 1
		}
		b.WriteByte(c)
		i++
	}
	return b.String(), i
}
