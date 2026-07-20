package main

import (
	"context"
	"log"
	"math"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// nonThumbURL strips the "/thumbs" path segment from a classyfm.co.id image
// URL, e.g. "https://classyfm.co.id/file/post/thumbs/<hash>.jpeg" ->
// "https://classyfm.co.id/file/post/<hash>.jpeg". WordPress-era classyfm.co.id
// serves resized copies under /thumbs/ alongside the original at the same path
// without it; ok is false when src has no such segment to strip.
func nonThumbURL(src string) (full string, ok bool) {
	const marker = "/thumbs/"
	idx := strings.Index(src, marker)
	if idx == -1 {
		return "", false
	}
	return src[:idx] + "/" + src[idx+len(marker):], true
}

// upgradeImages re-visits every existing hot_release row and tries to replace
// its already-downloaded thumbnail with the full-resolution original from
// classyfm.co.id. This depends on the old site being reachable and serving
// real images (as of this writing its PHP backend is down, so every attempt
// will fail and be skipped - safe to re-run once it recovers).
func (imp *importer) upgradeImages(ctx context.Context) error {
	rows, err := imp.q.ListAllHotRelease(ctx, sqlc.ListAllHotReleaseParams{Search: "%", Limit: math.MaxInt32})
	if err != nil {
		return err
	}
	log.Printf("upgrade-images: checking %d hot_release rows", len(rows))

	upgraded, skipped, failed := 0, 0, 0
	for _, row := range rows {
		if !row.Url.Valid || row.Url.String == "" {
			skipped++
			continue
		}

		detail, err := imp.fetchDetail(ctx, row.Url.String)
		if err != nil {
			log.Printf("  warn: fetch %q: %v", row.Url.String, err)
			failed++
			continue
		}
		full, ok := nonThumbURL(detail.ImageSrc)
		if !ok {
			skipped++
			continue
		}

		if imp.dryRun {
			log.Printf("  [dry-run] would upgrade %q: %s -> %s", row.Title, detail.ImageSrc, full)
			upgraded++
			continue
		}

		stored, err := imp.downloadImage(ctx, full)
		if err != nil {
			log.Printf("  warn: download full-res failed for %q: %v", row.Title, err)
			failed++
			continue
		}

		// Preserve the already-downloaded low-res copy as ThumbUrl (for
		// newsfeed/list views) rather than losing it - ImageUrl becomes the
		// new hi-res download, used only by the hero section.
		err = imp.q.UpdateHotRelease(ctx, sqlc.UpdateHotReleaseParams{
			Title:       row.Title,
			Slug:        row.Slug,
			Excerpt:     row.Excerpt,
			Content:     row.Content,
			ImageUrl:    toNullString(stored),
			ThumbUrl:    row.ImageUrl,
			PublishedAt: row.PublishedAt,
			IsPublished: row.IsPublished,
			IsFeatured:  row.IsFeatured,
			ID:          row.ID,
		})
		if err != nil {
			log.Printf("  warn: update row %d failed: %v", row.ID, err)
			failed++
			continue
		}
		log.Printf("  upgraded %q", row.Title)
		upgraded++

		time.Sleep(imp.delay)
	}

	log.Printf("upgrade-images done: %d upgraded, %d skipped (no /thumbs/ variant), %d failed", upgraded, skipped, failed)
	return nil
}
