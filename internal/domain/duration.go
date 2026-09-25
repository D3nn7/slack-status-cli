package domain

import "time"

// ClearMode mirrors Slack's "Clear after" selector for a status.
type ClearMode int

const (
	ClearNever ClearMode = iota
	Clear30Minutes
	Clear1Hour
	Clear4Hours
	ClearToday
	ClearThisWeek
	ClearCustom
)

// ClearModeLabel returns the human readable, Slack-like label for a mode.
func ClearModeLabel(mode ClearMode) string {
	switch mode {
	case Clear30Minutes:
		return "30 minutes"
	case Clear1Hour:
		return "1 hour"
	case Clear4Hours:
		return "4 hours"
	case ClearToday:
		return "Today"
	case ClearThisWeek:
		return "This week"
	case ClearCustom:
		return "Custom"
	default:
		return "Don't clear"
	}
}

// AllClearModes lists every selectable mode in display order.
func AllClearModes() []ClearMode {
	return []ClearMode{
		ClearNever,
		Clear30Minutes,
		Clear1Hour,
		Clear4Hours,
		ClearToday,
		ClearThisWeek,
		ClearCustom,
	}
}

// Expiration computes the absolute expiry for a mode relative to now. A zero
// time is returned for ClearNever and for custom durations that are not set.
func (m ClearMode) Expiration(now time.Time, custom time.Duration) time.Time {
	switch m {
	case Clear30Minutes:
		return now.Add(30 * time.Minute)
	case Clear1Hour:
		return now.Add(time.Hour)
	case Clear4Hours:
		return now.Add(4 * time.Hour)
	case ClearToday:
		return endOfDay(now)
	case ClearThisWeek:
		return endOfWeek(now)
	case ClearCustom:
		if custom <= 0 {
			return time.Time{}
		}
		return now.Add(custom)
	default:
		return time.Time{}
	}
}

// endOfDay returns the last second of the day containing t.
func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
}

// endOfWeek returns the end of the calendar week (Sunday, 23:59:59) for t.
func endOfWeek(t time.Time) time.Time {
	daysUntilSunday := (int(time.Sunday) - int(t.Weekday()) + 7) % 7
	return endOfDay(t.AddDate(0, 0, daysUntilSunday))
}
