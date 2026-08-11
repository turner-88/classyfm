// Command backfillbroadcasters is a one-shot migration helper: it reads the
// legacy free-text host columns from programs and program_schedules, matches
// them against broadcaster names, and populates the new broadcaster_id FK on
// program_schedules.
//
// Run AFTER migration 0022 (which adds broadcaster_id) and BEFORE migration
// 0023 (which drops the host columns).
//
// Defaults to a dry run (-dry-run=true); pass -dry-run=false to actually write.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/classyfm/classyfm/internal/config"
	"github.com/classyfm/classyfm/internal/db"
	_ "github.com/go-sql-driver/mysql"
)

type broadcaster struct {
	ID   uint64
	Name string
}

func main() {
	dryRun := flag.Bool("dry-run", true, "print what would change without writing to the database")
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
		log.Fatalf("db.Open: %v", err)
	}
	defer pool.Close()

	broadcasters := loadBroadcasters(ctx, pool)
	if len(broadcasters) == 0 {
		log.Println("no broadcasters found, nothing to do")
		return
	}
	log.Printf("loaded %d broadcasters", len(broadcasters))

	programHosts := loadProgramHosts(ctx, pool)
	log.Printf("loaded %d programs with host text", len(programHosts))

	type slot struct {
		ID        uint64
		ProgramID uint64
		Host      sql.NullString
	}
	rows, err := pool.QueryContext(ctx, "SELECT id, program_id, host FROM program_schedules")
	if err != nil {
		log.Fatalf("query schedules: %v", err)
	}
	var slots []slot
	for rows.Next() {
		var s slot
		if err := rows.Scan(&s.ID, &s.ProgramID, &s.Host); err != nil {
			log.Fatalf("scan schedule: %v", err)
		}
		slots = append(slots, s)
	}
	rows.Close()
	log.Printf("loaded %d schedule slots", len(slots))

	var matched, unmatched int
	for _, s := range slots {
		hostText := s.Host.String
		if !s.Host.Valid || hostText == "" {
			hostText = programHosts[s.ProgramID]
		}
		if hostText == "" {
			continue
		}

		bid := matchBroadcaster(broadcasters, hostText)
		if bid == 0 {
			unmatched++
			log.Printf("  NO MATCH  slot=%d program=%d host=%q", s.ID, s.ProgramID, hostText)
			continue
		}

		matched++
		if *dryRun {
			log.Printf("  DRY RUN   slot=%d -> broadcaster=%d host=%q", s.ID, bid, hostText)
		} else {
			if _, err := pool.ExecContext(ctx, "UPDATE program_schedules SET broadcaster_id = ? WHERE id = ?", bid, s.ID); err != nil {
				log.Printf("  ERROR     slot=%d: %v", s.ID, err)
			} else {
				log.Printf("  UPDATED   slot=%d -> broadcaster=%d", s.ID, bid)
			}
		}
	}

	fmt.Println()
	log.Printf("done: %d matched, %d unmatched", matched, unmatched)
	if *dryRun {
		log.Println("(dry run — pass -dry-run=false to write)")
	}
}

func loadBroadcasters(ctx context.Context, pool *sql.DB) []broadcaster {
	rows, err := pool.QueryContext(ctx, "SELECT id, name FROM broadcasters")
	if err != nil {
		log.Fatalf("query broadcasters: %v", err)
	}
	defer rows.Close()
	var list []broadcaster
	for rows.Next() {
		var b broadcaster
		if err := rows.Scan(&b.ID, &b.Name); err != nil {
			log.Fatalf("scan broadcaster: %v", err)
		}
		list = append(list, b)
	}
	return list
}

func loadProgramHosts(ctx context.Context, pool *sql.DB) map[uint64]string {
	rows, err := pool.QueryContext(ctx, "SELECT id, host FROM programs WHERE host IS NOT NULL AND host != ''")
	if err != nil {
		log.Fatalf("query programs: %v", err)
	}
	defer rows.Close()
	m := map[uint64]string{}
	for rows.Next() {
		var id uint64
		var host string
		if err := rows.Scan(&id, &host); err != nil {
			log.Fatalf("scan program: %v", err)
		}
		m[id] = host
	}
	return m
}

// matchBroadcaster tries to find the best broadcaster match for a host text
// string. The host field may contain multiple names separated by commas or "&".
// We try to match the first name fragment that matches a broadcaster.
func matchBroadcaster(broadcasters []broadcaster, hostText string) uint64 {
	hostText = strings.ReplaceAll(hostText, "&", ",")
	parts := strings.Split(hostText, ",")
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		nameLower := strings.ToLower(name)
		for _, b := range broadcasters {
			bLower := strings.ToLower(b.Name)
			if bLower == nameLower || strings.Contains(bLower, nameLower) || strings.Contains(nameLower, bLower) {
				return b.ID
			}
		}
	}
	return 0
}
