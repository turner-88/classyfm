// Command importhotrelease backfills Hot Release news items from the legacy
// classyfm.co.id site's "Hot Release" category into news_items (source =
// hot_release). internal/feeds deliberately never touches hot_release rows -
// this is a one-off migration tool, not a recurring worker.
//
// Dedupe is checked against whatever is already in the database (by
// normalized title and by the old site's article URL, which this tool also
// records going forward), so the command is safe to interrupt and re-run: a
// re-run just skips everything already imported and continues.
//
// Defaults to a dry run (-dry-run=true); pass -dry-run=false to actually
// write rows and download images.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/config"
	"github.com/classyfm/classyfm/internal/db"
	"github.com/classyfm/classyfm/internal/db/sqlc"
)

const oldSiteBase = "https://classyfm.co.id"

func main() {
	dryRun := flag.Bool("dry-run", true, "print what would be imported without writing to the database or downloading images")
	limit := flag.Int("limit", 0, "stop after importing this many new articles (0 = unlimited)")
	delay := flag.Duration("delay", 400*time.Millisecond, "delay between requests to the old site")
	upgradeImages := flag.Bool("upgrade-images", false, "instead of importing new articles, re-fetch existing hot_release rows and replace their thumbnail with the full-resolution original from classyfm.co.id (requires the old site to be reachable)")
	flag.Parse()

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

	// This tool needs every existing row for dedup, not one page of them.
	existing, err := q.ListAllHotRelease(ctx, sqlc.ListAllHotReleaseParams{Limit: math.MaxInt32})
	if err != nil {
		log.Fatalf("list existing hot release: %v", err)
	}

	imp := &importer{
		q:          q,
		uploadDir:  cfg.UploadDir,
		client:     &http.Client{Timeout: 20 * time.Second},
		delay:      *delay,
		dryRun:     *dryRun,
		limit:      *limit,
		seenTitles: make(map[string]bool, len(existing)),
		seenURLs:   make(map[string]bool, len(existing)),
		seenSlugs:  make(map[string]bool, len(existing)),
	}
	for _, item := range existing {
		imp.seenTitles[normalizeTitle(item.Title)] = true
		if item.Url.Valid && item.Url.String != "" {
			imp.seenURLs[item.Url.String] = true
		}
		if item.Slug.Valid && item.Slug.String != "" {
			imp.seenSlugs[item.Slug.String] = true
		}
	}
	log.Printf("loaded %d existing hot_release rows for dedupe", len(existing))
	if *dryRun {
		log.Println("DRY RUN: no database writes or image downloads will happen")
	}

	if *upgradeImages {
		if err := imp.upgradeImages(ctx); err != nil {
			log.Fatalf("upgrade-images failed: %v", err)
		}
		return
	}

	if err := imp.run(ctx); err != nil {
		log.Fatalf("import failed: %v", err)
	}
}

// importer holds the shared state for one import run: the dedupe sets seeded
// from the database, plus running counters for the final summary.
type importer struct {
	q         *sqlc.Queries
	uploadDir string
	client    *http.Client
	delay     time.Duration
	dryRun    bool
	limit     int

	seenTitles map[string]bool
	seenURLs   map[string]bool
	seenSlugs  map[string]bool

	imported int
	skipped  int
	failed   int
}

// run paginates the old site's Hot Release listing until a page comes back
// with no articles (the site doesn't bound-check page numbers - past the
// last real page it silently renders unrelated homepage widgets instead).
func (imp *importer) run(ctx context.Context) error {
	for page := 1; ; page++ {
		items, err := imp.fetchListingPage(ctx, page)
		if err != nil {
			return fmt.Errorf("fetch listing page %d: %w", page, err)
		}
		if len(items) == 0 {
			log.Printf("page %d: no items, stopping", page)
			break
		}

		pageImported, pageSkipped := 0, 0
		for _, li := range items {
			if imp.limit > 0 && imp.imported >= imp.limit {
				log.Printf("reached -limit=%d, stopping", imp.limit)
				imp.summary()
				return nil
			}

			ok, err := imp.processListing(ctx, li)
			time.Sleep(imp.delay)
			if err != nil {
				log.Printf("  error processing %q: %v", li.Title, err)
				imp.failed++
				continue
			}
			if ok {
				pageImported++
			} else {
				pageSkipped++
			}
		}
		log.Printf("page %d: %d items, %d imported, %d skipped", page, len(items), pageImported, pageSkipped)
	}

	imp.summary()
	return nil
}

func (imp *importer) summary() {
	log.Printf("done: %d imported, %d skipped (dupes), %d failed", imp.imported, imp.skipped, imp.failed)
}

// processListing dedupes, fetches the detail page, downloads the image, and
// (unless dry-run) inserts the row. imported reports whether a new article
// was created (or would be, in dry-run).
func (imp *importer) processListing(ctx context.Context, li listingItem) (imported bool, err error) {
	norm := normalizeTitle(li.Title)
	if imp.seenTitles[norm] || imp.seenURLs[li.URL] {
		imp.skipped++
		return false, nil
	}

	detail, err := imp.fetchDetail(ctx, li.URL)
	if err != nil {
		return false, err
	}

	content := detail.Body
	slug := imp.uniqueSlug(li.Title)

	var imageURL sql.NullString
	if detail.ImageSrc != "" {
		stored, err := imp.downloadImage(ctx, detail.ImageSrc)
		if err != nil {
			log.Printf("  warn: image download failed for %q: %v", li.Title, err)
		} else {
			imageURL = sql.NullString{String: stored, Valid: true}
		}
	}

	if imp.dryRun {
		log.Printf("  [dry-run] would create: title=%q slug=%q published_at=%s excerpt=%dch content=%dch image=%v",
			li.Title, slug, li.PublishedAt.Format("2006-01-02"), len(li.Excerpt), len(content), detail.ImageSrc != "")
	} else {
		_, err = imp.q.CreateHotReleaseImported(ctx, sqlc.CreateHotReleaseImportedParams{
			Title:    li.Title,
			Slug:     sql.NullString{String: slug, Valid: true},
			Excerpt:  toNullString(li.Excerpt),
			Content:  toNullString(content),
			Url:      sql.NullString{String: li.URL, Valid: true},
			ImageUrl: imageURL,
			// Only one file is downloaded at import time - it's a low-res
			// GD thumbnail from classyfm.co.id's /thumbs/ path (see
			// upgrade.go), so it doubles as ThumbUrl until -upgrade-images
			// later fetches a genuine hi-res replacement.
			ThumbUrl:    imageURL,
			PublishedAt: li.PublishedAt,
			IsPublished: true,
			IsFeatured:  false,
		})
		if err != nil {
			return false, fmt.Errorf("insert: %w", err)
		}
		log.Printf("  created %q (slug=%s, published=%s)", li.Title, slug, li.PublishedAt.Format("2006-01-02"))
	}

	imp.seenTitles[norm] = true
	imp.seenURLs[li.URL] = true
	imp.seenSlugs[slug] = true
	imp.imported++
	return true, nil
}

// dashVariants replaces Unicode dash characters (en dash, em dash, minus
// sign) with a plain hyphen - the old site's listing and detail pages don't
// consistently use the same dash character for the same article title.
var dashVariants = strings.NewReplacer("–", "-", "—", "-", "−", "-")

// normalizeTitle collapses whitespace, case, and dash-character differences
// so the dedupe check isn't thrown off by minor formatting drift between how
// a title was hand-entered originally and how it's scraped now.
func normalizeTitle(title string) string {
	title = dashVariants.Replace(title)
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

func toNullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
