// Package schedule resolves "what is on air right now" from the weekly program
// schedule. It lives outside the handler packages because both the public site
// (Home's on-air card, /live's timeline, /api/schedule/current) and the admin
// dashboard need the same answer, and neither should re-derive the overnight and
// progress rules that models.IsAiring*/Progress encode.
package schedule

import (
	"context"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/models"
)

// Loc is the station's local timezone (WIB, Padang/West Sumatra), used for all
// schedule/on-air comparisons so correctness doesn't depend on the host OS's
// configured timezone.
var Loc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}()

// Row is the view-model for one weekly schedule slot (Home's "On Air" card,
// Live's full schedule list, the admin dashboard's on-air strip).
type Row struct {
	StartTime    string
	EndTime      string
	ProgramTitle string
	ProgramSlug  string
	ProgramHost  string
	ProgramImage string
	OnAir        bool
	Progress     int  // 0-100, only meaningful when OnAir
	Ended        bool // slot already finished earlier today (dimmed in the timeline)
}

// TodayRows returns today's full schedule (each row flagged OnAir), including any
// overnight-spanning slot that started yesterday and is still airing (spillover -
// otherwise "on air now" could point at nothing right after midnight). Returns nil
// if no database is configured.
func TodayRows(ctx context.Context, q *sqlc.Queries) []Row {
	if q == nil {
		return nil
	}
	var today []Row
	now := time.Now().In(Loc)
	nowClock := now.Format("15:04:05")
	todayDOW := int8(now.Weekday())
	yesterdayDOW := int8((int(todayDOW) + 6) % 7)

	if rows, err := q.ListSchedulesByDay(ctx, yesterdayDOW); err == nil {
		for _, row := range rows {
			if models.IsAiringFromYesterday(nowClock, row.StartTime, row.EndTime) {
				today = append(today, Row{
					StartTime:    models.ClockLabel(row.StartTime),
					EndTime:      models.ClockLabel(row.EndTime),
					ProgramTitle: row.ProgramTitle,
					ProgramSlug:  row.ProgramSlug,
					ProgramHost:  row.BroadcasterName.String,
					ProgramImage: row.ProgramImageUrl.String,
					OnAir:        true,
					Progress:     models.Progress(nowClock, row.StartTime, row.EndTime),
				})
			}
		}
	}
	if rows, err := q.ListSchedulesByDay(ctx, todayDOW); err == nil {
		for _, row := range rows {
			onAir := models.IsAiringToday(nowClock, row.StartTime, row.EndTime)
			progress := 0
			if onAir {
				progress = models.Progress(nowClock, row.StartTime, row.EndTime)
			}
			today = append(today, Row{
				StartTime:    models.ClockLabel(row.StartTime),
				EndTime:      models.ClockLabel(row.EndTime),
				ProgramTitle: row.ProgramTitle,
				ProgramSlug:  row.ProgramSlug,
				ProgramHost:  row.BroadcasterName.String,
				ProgramImage: row.ProgramImageUrl.String,
				OnAir:        onAir,
				Progress:     progress,
				Ended:        models.HasEnded(nowClock, row.StartTime, row.EndTime),
			})
		}
	}
	return today
}

// Current returns the currently on-air row from rows (if any) and its index within
// rows. Used wherever a single "now" card is shown instead of the whole day.
func Current(rows []Row) (*Row, int) {
	for i := range rows {
		if rows[i].OnAir {
			return &rows[i], i
		}
	}
	return nil, -1
}

// Next returns the first slot of today that hasn't started yet, or nil once the
// broadcast day's listed slots are done. rows arrive in broadcast order (yesterday's
// spillover first, then today ascending), so the first row that is neither on air
// nor ended is the one coming up.
func Next(rows []Row) *Row {
	for i := range rows {
		if !rows[i].OnAir && !rows[i].Ended {
			return &rows[i]
		}
	}
	return nil
}
