// Command backfillpodcastdescriptions is a one-off backfill: it fills the
// description of existing podcasts from their Spotify link, for rows imported
// before the admin's Spotify auto-fetch existed (cmd/importpodcasts left them
// blank). It uses the same spotify.FetchDescription + sanitize.PlainText path the
// admin handler runs on create, so backfilled and freshly-created podcasts read
// identically.
//
// Safe to re-run: by default only rows with an empty description are touched, so a
// rerun just retries the ones a transient Spotify fetch missed. Pass -all to refetch
// and overwrite every podcast.
//
// Defaults to a dry run (-dry-run=true); pass -dry-run=false to actually write.
package main

import (
	"context"
	"flag"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/classyfm/classyfm/internal/config"
	"github.com/classyfm/classyfm/internal/db"
	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/sanitize"
	"github.com/classyfm/classyfm/internal/spotify"
)

func main() {
	dryRun := flag.Bool("dry-run", true, "print what would change without writing to the database")
	delay := flag.Duration("delay", 300*time.Millisecond, "delay between requests to Spotify")
	all := flag.Bool("all", false, "refetch and overwrite every podcast, not just those with an empty description")
	envFile := flag.String("env", ".env", "path to a .env file to load DATABASE_DSN etc. from (skipped if absent)")
	flag.Parse()

	// Run by hand rather than under systemd/Makefile, so load .env ourselves; an
	// already-set DATABASE_DSN still wins. See config.LoadEnvFile.
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
	client := &http.Client{Timeout: 20 * time.Second}

	rows, err := q.ListPodcasts(ctx, sqlc.ListPodcastsParams{
		Search: "%",
		Sort:   "created_at",
		Dir:    "desc",
		Limit:  math.MaxInt32,
		Offset: 0,
	})
	if err != nil {
		log.Fatalf("list podcasts: %v", err)
	}
	log.Printf("checking %d podcasts", len(rows))
	if *dryRun {
		log.Println("DRY RUN: no database writes will happen")
	}

	filled, skipped, failed := 0, 0, 0
	for _, row := range rows {
		if row.Description != "" && !*all {
			skipped++
			continue
		}

		desc, err := spotify.FetchDescription(ctx, client, row.SpotifyUrl)
		if err != nil {
			log.Printf("  warn: %q: %v", row.Title, err)
			failed++
			continue
		}
		desc = sanitize.PlainText(desc)
		if desc == "" {
			log.Printf("  warn: %q: Spotify returned no description", row.Title)
			failed++
			time.Sleep(*delay)
			continue
		}

		if *dryRun {
			log.Printf("  [dry-run] %q -> %.80q", row.Title, desc)
		} else {
			if err := q.SetPodcastDescription(ctx, sqlc.SetPodcastDescriptionParams{
				Description: desc,
				ID:          row.ID,
			}); err != nil {
				log.Printf("  warn: update podcast %d failed: %v", row.ID, err)
				failed++
				time.Sleep(*delay)
				continue
			}
			log.Printf("  filled %q", row.Title)
		}
		filled++
		time.Sleep(*delay)
	}

	log.Printf("done: %d filled, %d skipped, %d failed", filled, skipped, failed)
}
