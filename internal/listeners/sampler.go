// Package listeners records the station's audience over time. The Shoutcast
// server reports the number live but remembers nothing across a restart, so the
// only way to have a listener history is to poll for one and store it.
package listeners

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/radio"
	"github.com/classyfm/classyfm/internal/schedule"
)

// Sampler polls the Shoutcast server's current listener count on an interval,
// appending each reading to listener_samples and folding it into that day's
// listener_stats row. Same shape as feeds.Worker: it owns its own ticker, logs
// every failure, and never returns an error to its caller.
type Sampler struct {
	q         *sqlc.Queries
	radio     *radio.Service
	interval  time.Duration
	retention time.Duration

	// prunedDay is the station-local date the raw trail was last pruned for, so
	// the prune runs once a day rather than on all ~288 of a day's ticks.
	prunedDay string
}

// NewSampler builds a listener sampler. q and r must be non-nil.
func NewSampler(q *sqlc.Queries, r *radio.Service, interval, retention time.Duration) *Sampler {
	return &Sampler{q: q, radio: r, interval: interval, retention: retention}
}

// Run samples immediately, then again on every tick, until ctx is canceled.
func (s *Sampler) Run(ctx context.Context) {
	s.RunOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RunOnce(ctx)
		}
	}
}

// RunOnce takes a single reading and stores it.
func (s *Sampler) RunOnce(ctx context.Context) {
	count, live, ok := s.radio.StreamStats(ctx)
	if !ok {
		// Record nothing at all rather than a zero. An unreachable server and an
		// empty audience are different facts, and writing the first as the second
		// would silently drag the day's peak down through every outage.
		slog.Warn("listener sample skipped: stats unavailable")
		return
	}

	now := time.Now().In(schedule.Loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, schedule.Loc)

	if err := s.q.InsertListenerSample(ctx, sqlc.InsertListenerSampleParams{
		SampledAt: now,
		Listeners: uint32(count),
		IsLive:    live,
	}); err != nil {
		slog.Error("listener sample insert failed", "err", err)
	}

	if err := s.q.UpsertListenerDay(ctx, sqlc.UpsertListenerDayParams{
		StatDate:  day,
		Listeners: uint32(count),
		SampledAt: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		slog.Error("listener day upsert failed", "err", err)
	}

	s.prune(ctx, now)
}

// prune drops raw samples past the retention window, at most once per day. The
// daily rollup is never pruned - it's one small row per day and it's the whole
// point of the feature.
func (s *Sampler) prune(ctx context.Context, now time.Time) {
	today := now.Format("2006-01-02")
	if s.prunedDay == today || s.retention <= 0 {
		return
	}
	s.prunedDay = today
	if err := s.q.PruneListenerSamples(ctx, now.Add(-s.retention)); err != nil {
		slog.Warn("listener sample prune failed", "err", err)
	}
}
