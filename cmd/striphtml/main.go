// Command striphtml is a one-off maintenance tool that re-runs sanitize.PlainText
// over existing Hot Release rows, cleaning up HTML tags saved before the admin
// content field dropped HTML support. Defaults to a dry run; pass -dry-run=false
// to actually write the cleaned content back to the database.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/classyfm/classyfm/internal/config"
	"github.com/classyfm/classyfm/internal/db"
	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/sanitize"
)

func main() {
	dryRun := flag.Bool("dry-run", true, "print what would change without writing to the database")
	flag.Parse()

	cfg := config.Load()
	if cfg.DatabaseDSN == "" {
		log.Fatal("DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("connect to mysql: %v", err)
	}
	defer pool.Close()

	q := sqlc.New(pool)
	items, err := q.ListAllHotRelease(ctx)
	if err != nil {
		log.Fatalf("list hot release: %v", err)
	}

	changed := 0
	for _, item := range items {
		cleaned := sanitize.PlainText(item.Content.String)
		if cleaned == item.Content.String {
			continue
		}
		changed++
		fmt.Printf("#%d %q: %d chars -> %d chars\n", item.ID, item.Title, len(item.Content.String), len(cleaned))
		fmt.Printf("  before: %q\n", preview(item.Content.String))
		fmt.Printf("  after:  %q\n", preview(cleaned))

		if *dryRun {
			continue
		}
		err := q.UpdateHotRelease(ctx, sqlc.UpdateHotReleaseParams{
			Title:       item.Title,
			Slug:        item.Slug,
			Excerpt:     item.Excerpt,
			Content:     sql.NullString{String: cleaned, Valid: cleaned != ""},
			ImageUrl:    item.ImageUrl,
			PublishedAt: item.PublishedAt,
			IsPublished: item.IsPublished,
			IsFeatured:  item.IsFeatured,
			ID:          item.ID,
		})
		if err != nil {
			log.Fatalf("update #%d: %v", item.ID, err)
		}
	}

	if *dryRun {
		fmt.Printf("\ndry run: %d of %d rows would change (re-run with -dry-run=false to apply)\n", changed, len(items))
		return
	}
	fmt.Printf("\ndone: %d of %d rows updated\n", changed, len(items))
}

func preview(s string) string {
	const max = 120
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
