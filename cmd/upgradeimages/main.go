// Command upgradeimages is a one-off backfill: it revisits already-aggregated
// news_items rows (klikpositif, katasumbar, youtube - hot_release is handled
// separately by cmd/importhotrelease -upgrade-images) and replaces any
// low-resolution image with a higher-resolution one, using the exact same
// resolution logic internal/feeds applies to newly-fetched items.
//
// Safe to re-run: rows that already have their best available image are just
// left unchanged.
//
// Defaults to a dry run (-dry-run=true); pass -dry-run=false to actually write.
package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/classyfm/classyfm/internal/config"
	"github.com/classyfm/classyfm/internal/db"
	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/feeds"
)

func main() {
	dryRun := flag.Bool("dry-run", true, "print what would change without writing to the database")
	delay := flag.Duration("delay", 300*time.Millisecond, "delay between requests to external sites")
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
	client := &http.Client{Timeout: 20 * time.Second}

	rows, err := q.ListAggregatedNews(ctx, sqlc.ListAggregatedNewsParams{
		Search: "%",
		Limit:  math.MaxInt32,
	})
	if err != nil {
		log.Fatalf("list aggregated news: %v", err)
	}
	log.Printf("checking %d rows (klikpositif/katasumbar/youtube)", len(rows))
	if *dryRun {
		log.Println("DRY RUN: no database writes will happen")
	}

	upgraded, unchanged, failed := 0, 0, 0
	for _, row := range rows {
		image, thumb, err := bestImages(ctx, client, row)
		if err != nil {
			log.Printf("  warn: %s %q: %v", row.Source, row.Title, err)
			failed++
			continue
		}
		if image == "" || (image == row.ImageUrl.String && thumb == row.ThumbUrl.String) {
			unchanged++
			continue
		}

		if *dryRun {
			log.Printf("  [dry-run] %s %q: image %s -> %s, thumb %s -> %s",
				row.Source, row.Title, row.ImageUrl.String, image, row.ThumbUrl.String, thumb)
		} else {
			err := q.UpdateNewsItemImages(ctx, sqlc.UpdateNewsItemImagesParams{
				ImageUrl: sql.NullString{String: image, Valid: true},
				ThumbUrl: sql.NullString{String: thumb, Valid: thumb != ""},
				ID:       row.ID,
			})
			if err != nil {
				log.Printf("  warn: update row %d failed: %v", row.ID, err)
				failed++
				continue
			}
			log.Printf("  upgraded %s %q", row.Source, row.Title)
		}
		upgraded++
		time.Sleep(*delay)
	}

	log.Printf("done: %d upgraded, %d unchanged, %d failed", upgraded, unchanged, failed)
}

// bestImages resolves the hi-res image (for the hero section) and the
// list-appropriate thumbnail (for everywhere else) available for row, using
// the same logic as the live feed fetchers. Returns "" for image if no image
// is available at all; thumb falls back to whatever's already on record
// rather than being duplicated from image once it's already resolved, so a
// rerun doesn't clobber a thumb populated by some other pass.
func bestImages(ctx context.Context, client *http.Client, row sqlc.NewsItem) (image, thumb string, err error) {
	switch row.Source {
	case sqlc.NewsItemsSourceKlikpositif, sqlc.NewsItemsSourceKatasumbar:
		if row.ImageUrl.Valid && row.ImageUrl.String != "" {
			original, resized := feeds.WPOriginalURL(row.ImageUrl.String)
			if !resized {
				// No WordPress size suffix to strip: row.ImageUrl is already
				// the full-size original (or was never a WP-generated size).
				return row.ImageUrl.String, row.ThumbUrl.String, nil
			}
			if feeds.ProbeImageExists(ctx, client, original) {
				// row.ImageUrl was still the raw RSS thumbnail - it becomes thumb.
				return original, row.ImageUrl.String, nil
			}
			// Same second chance WordPressSource.resolveImages takes: the
			// original is gone but og:image still points at the full-size
			// featured image. PreferImage guards against it being a resize of
			// what's already stored, so this can only upgrade.
			if row.Url.Valid && row.Url.String != "" {
				og, err := feeds.FetchOGImage(ctx, client, row.Url.String)
				if err == nil && og != "" && feeds.PreferImage(row.ImageUrl.String, og) == og {
					return og, row.ImageUrl.String, nil
				}
			}
			return row.ImageUrl.String, row.ThumbUrl.String, nil
		}
		if row.Url.Valid && row.Url.String != "" {
			// No RSS-embedded thumbnail ever existed (katasumbar) - og:image
			// is the only image available, used for both.
			og, err := feeds.FetchOGImage(ctx, client, row.Url.String)
			return og, og, err
		}
		return "", "", nil

	case sqlc.NewsItemsSourceYoutube:
		if !row.ExternalID.Valid || row.ExternalID.String == "" {
			return row.ImageUrl.String, row.ThumbUrl.String, nil
		}
		hires := feeds.BestYouTubeThumbnail(ctx, client, row.ExternalID.String, row.ImageUrl.String)
		thumbURL := "https://i.ytimg.com/vi/" + row.ExternalID.String + "/hqdefault.jpg"
		return hires, thumbURL, nil

	default:
		return row.ImageUrl.String, row.ThumbUrl.String, nil
	}
}
