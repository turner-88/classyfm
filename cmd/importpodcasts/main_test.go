package main

import (
	"os"
	"path/filepath"
	"testing"
)

// dumpPath locates the legacy dump at the repo root relative to this package.
func dumpPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join("..", "..", "backup-old-db.sql")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("dump not available: %v", err)
	}
	return p
}

// TestParseAndClassify checks that parsing the real dump and running the same filter
// and classification the importer uses reproduces the analyzed shape. Of 251 posts
// with a Spotify embed, one is soft-deleted and two more collapse as duplicate embeds
// (a second post pointing at an already-seen episode), leaving 248 unique episodes
// across the expected 8 series.
func TestParseAndClassify(t *testing.T) {
	posts, err := parsePosts(dumpPath(t))
	if err != nil {
		t.Fatalf("parsePosts: %v", err)
	}

	perSeries := map[string]int{}
	seenURL := map[string]bool{}
	importable := 0
	for _, p := range posts {
		if p.deletedAt != "" {
			continue
		}
		ep := episodeID(p.content)
		if ep == "" {
			continue
		}
		url := "https://open.spotify.com/episode/" + ep
		if seenURL[url] {
			continue // duplicate embed
		}
		seenURL[url] = true
		rule := matchSeries(p.title)
		if rule == nil {
			t.Errorf("no series for title %q", p.title)
			continue
		}
		if got := rule.cleanTitle(p.title); got == "" {
			t.Errorf("empty clean title for %q", p.title)
		}
		perSeries[rule.name]++
		importable++
	}

	want := map[string]int{
		"Semen Padang Journal":            124, // 125 posts, one a duplicate embed
		"Kanal BRI":                       53,
		"Special Talkshow":                22, // 24 posts, one soft-deleted + one duplicate embed
		"The Art of Leadership":           23,
		"Talkshow Bebas Pusing":           12,
		"Parliament Talk":                 9,
		"Communitalk with Yeni Maiasnita": 4,
		"Podcast Bank Indonesia":          1,
	}
	for name, n := range want {
		if perSeries[name] != n {
			t.Errorf("series %q: got %d, want %d", name, perSeries[name], n)
		}
	}
	for name := range perSeries {
		if _, ok := want[name]; !ok {
			t.Errorf("unexpected series %q (%d)", name, perSeries[name])
		}
	}
	// 251 unique spotify posts; one duplicate embed collapses, so importable is a bit
	// under. Assert the total against the sum of the expected per-series counts.
	total := 0
	for _, n := range want {
		total += n
	}
	if importable != total {
		t.Errorf("importable = %d, want %d", importable, total)
	}
}
