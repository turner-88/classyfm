// Package models holds domain/view-model helpers shared across handlers and templates.
package models

import (
	"database/sql"
	"strconv"
)

// Weekday returns the label for a program_schedules.day_of_week value
// (0=Sunday .. 6=Saturday).
func Weekday(day int) string {
	if day < 0 || day > 6 {
		return ""
	}
	return weekdayNames[day]
}

var weekdayNames = [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// Weekdays returns all seven weekday labels, indexed 0=Sunday..6=Saturday.
func Weekdays() [7]string { return weekdayNames }

// ClockLabel formats a MySQL TIME value ("HH:MM:SS", as returned by the driver) as
// "HH:MM" for display.
func ClockLabel(s string) string {
	if len(s) >= 5 {
		return s[:5]
	}
	return s
}

// IsAiringToday reports whether a schedule slot whose day_of_week is *today* is
// airing right now. now/start/end are "HH:MM:SS" strings (as returned by the MySQL
// driver / time.Now().Format("15:04:05")), so plain string comparison works.
// Overnight-spanning slots (start > end, e.g. 23:00-01:00) air from start until
// midnight.
func IsAiringToday(now, start, end string) bool {
	switch {
	case start < end:
		return now >= start && now < end
	case start > end:
		return now >= start
	default: // start == end: invalid/zero-length, admin validation prevents this
		return false
	}
}

// IsAiringFromYesterday reports whether an overnight-spanning slot (start > end)
// whose day_of_week is *yesterday* is still airing now - the second half of a show
// like 23:00 Mon-01:00 Tue, checked at 00:30 Tue where the row's day is Monday.
func IsAiringFromYesterday(now, start, end string) bool {
	return start > end && now < end
}

// Progress returns how far (0-100) "now" is through the [start,end) slot, using
// minutes-of-day mod 1440 arithmetic so overnight-spanning slots (e.g.
// 23:00-01:00) work the same as same-day ones without a separate branch.
// now/start/end are "HH:MM..." strings; only the first 5 characters are read.
func Progress(now, start, end string) int {
	toMinutes := func(s string) int {
		h, _ := strconv.Atoi(s[0:2])
		m, _ := strconv.Atoi(s[3:5])
		return h*60 + m
	}
	const day = 24 * 60
	total := (toMinutes(end) - toMinutes(start) + day) % day
	if total == 0 {
		return 0
	}
	elapsed := (toMinutes(now) - toMinutes(start) + day) % day
	if elapsed > total {
		elapsed = total
	}
	return elapsed * 100 / total
}

// ResolveHost returns a schedule slot's own host if set, else the program's
// default host. Empty string if neither is set.
func ResolveHost(slotHost, programHost sql.NullString) string {
	if slotHost.Valid && slotHost.String != "" {
		return slotHost.String
	}
	return programHost.String
}

// ScheduleGroup is a compact display range merging consecutive weekdays that share
// an identical start/end/host, e.g. "Monday-Friday 07:00-10:00 - Budi".
type ScheduleGroup struct {
	FromDay, ToDay     int8
	StartTime, EndTime string
	Host               string
}

// GroupSlots merges consecutive-day slots (by day_of_week) that share identical
// start/end/host into single display ranges. slots must already be ordered by
// day_of_week ascending. Does not wrap Saturday(6)->Sunday(0); a schedule spanning
// every day of the week may render as two adjacent groups depending on which day
// it starts on - a display-only limitation, not a correctness issue.
func GroupSlots(slots []ProgramSlot) []ScheduleGroup {
	var groups []ScheduleGroup
	for _, s := range slots {
		if n := len(groups); n > 0 {
			last := &groups[n-1]
			if last.ToDay+1 == s.Day && last.StartTime == s.StartTime &&
				last.EndTime == s.EndTime && last.Host == s.Host {
				last.ToDay = s.Day
				continue
			}
		}
		groups = append(groups, ScheduleGroup{
			FromDay: s.Day, ToDay: s.Day,
			StartTime: s.StartTime, EndTime: s.EndTime, Host: s.Host,
		})
	}
	return groups
}

// ProgramSlot is one weekly airing slot on a program card, resolved for display
// (clock-formatted times, resolved host).
type ProgramSlot struct {
	Day                int8
	StartTime, EndTime string
	Host               string
}

// SourceLabel maps a news_items.source enum value to its display label.
func SourceLabel(source string) string {
	switch source {
	case "youtube":
		return "YouTube"
	case "klikpositif":
		return "KlikPositif"
	case "katasumbar":
		return "KataSumbar"
	case "hot_release":
		return "Hot Release"
	default:
		return source
	}
}
