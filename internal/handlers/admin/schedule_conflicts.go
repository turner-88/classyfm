package admin

import (
	"fmt"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/models"
)

// conflictSlot is one weekly airing slot reduced to what overlap detection needs: which
// program owns it and its day/time. It is a neutral view so the same engine serves the
// dashboard, the programs list, and the edit form, whose source rows differ.
type conflictSlot struct {
	ProgramID uint64
	Program   string // display title
	Day       int8
	Start     string // "HH:MM:SS" or "HH:MM"
	End       string
}

// conflictPair is two slots that share airtime. A and B may belong to the same program
// (two overlapping slots on one show) or to two different programs.
type conflictPair struct {
	A, B conflictSlot
}

// scheduleConflicts returns every unordered pair of slots that overlap, using the
// overnight-aware models.SlotsOverlap. O(n^2), which is fine: the whole station has on
// the order of tens of slots.
func scheduleConflicts(slots []conflictSlot) []conflictPair {
	var pairs []conflictPair
	for i := 0; i < len(slots); i++ {
		for j := i + 1; j < len(slots); j++ {
			a, b := slots[i], slots[j]
			if models.SlotsOverlap(a.Day, a.Start, a.End, b.Day, b.Start, b.End) {
				pairs = append(pairs, conflictPair{A: a, B: b})
			}
		}
	}
	return pairs
}

// conflictSlotsFromRows adapts the station-wide schedule query (active programs) into the
// neutral slot view.
func conflictSlotsFromRows(rows []sqlc.ListAllSchedulesWithProgramRow) []conflictSlot {
	slots := make([]conflictSlot, len(rows))
	for i, r := range rows {
		slots[i] = conflictSlot{
			ProgramID: r.ProgramID,
			Program:   r.ProgramTitle,
			Day:       r.DayOfWeek,
			Start:     r.StartTime,
			End:       r.EndTime,
		}
	}
	return slots
}

// conflictLine renders one overlapping pair as a human sentence. A same-program pair reads
// as one show doubling up; a cross-program pair names both shows.
func conflictLine(p conflictPair) string {
	day := models.Weekday(int(p.A.Day))
	aTime := models.ClockLabel(p.A.Start) + "–" + models.ClockLabel(p.A.End)
	bTime := models.ClockLabel(p.B.Start) + "–" + models.ClockLabel(p.B.End)
	if p.A.ProgramID == p.B.ProgramID {
		return fmt.Sprintf("%s: “%s” has two overlapping slots (%s and %s).", day, p.A.Program, aTime, bTime)
	}
	return fmt.Sprintf("%s: “%s” (%s) overlaps “%s” (%s).", day, p.A.Program, aTime, p.B.Program, bTime)
}

// conflictLines renders up to max pair sentences, summarising any remainder as one extra
// "+N more" line so a badly tangled schedule can't produce a wall of text. max <= 0 means
// no cap.
func conflictLines(pairs []conflictPair, max int) []string {
	lines := make([]string, 0, len(pairs))
	for _, p := range pairs {
		lines = append(lines, conflictLine(p))
	}
	if max > 0 && len(lines) > max {
		extra := len(lines) - max
		lines = lines[:max]
		lines = append(lines, fmt.Sprintf("+%d more overlap%s.", extra, plural(int64(extra), "", "s")))
	}
	return lines
}

// conflictNames returns the distinct program titles involved in any conflict, in first-seen
// order, for a one-line summary.
func conflictNames(pairs []conflictPair) []string {
	seen := map[string]bool{}
	var names []string
	for _, p := range pairs {
		for _, name := range [2]string{p.A.Program, p.B.Program} {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}
