package domain

import "time"

// Recurrence describes a weekly, time-of-day based repetition of a status.
type Recurrence struct {
	// Weekdays uses time.Weekday values (Sunday = 0). An empty slice matches
	// every day.
	Weekdays  []time.Weekday `json:"weekdays,omitempty"`
	StartTime string         `json:"startTime"` // HH:MM
	EndTime   string         `json:"endTime"`   // HH:MM
}

// Contains reports whether the recurrence is active at the given instant.
// End times before start times are treated as spanning midnight.
func (r Recurrence) Contains(now time.Time) bool {
	start, ok := parseClock(r.StartTime)
	if !ok {
		return false
	}
	end, ok := parseClock(r.EndTime)
	if !ok {
		return false
	}
	if !r.matchesWeekday(now) {
		return false
	}
	minutesNow := now.Hour()*60 + now.Minute()
	startMin := start.Hour()*60 + start.Minute()
	endMin := end.Hour()*60 + end.Minute()

	if startMin == endMin {
		return minutesNow == startMin
	}
	if endMin > startMin {
		return minutesNow >= startMin && minutesNow < endMin
	}
	// Overnight span, e.g. 22:00 -> 06:00.
	return minutesNow >= startMin || minutesNow < endMin
}

func (r Recurrence) matchesWeekday(now time.Time) bool {
	if len(r.Weekdays) == 0 {
		return true
	}
	for _, wd := range r.Weekdays {
		if wd == now.Weekday() {
			return true
		}
	}
	return false
}

// Schedule is a status that is applied automatically, either once in a given
// window or on a weekly recurrence.
type Schedule struct {
	ID         string      `json:"id,omitempty"`
	Label      string      `json:"label"`
	Text       string      `json:"text"`
	Emoji      string      `json:"emoji"`
	Enabled    bool        `json:"enabled"`
	Start      time.Time   `json:"start,omitzero"`
	End        time.Time   `json:"end,omitzero"`
	Recurrence *Recurrence `json:"recurrence,omitempty"`
}

// Active reports whether the schedule should currently be applied.
func (s Schedule) Active(now time.Time) bool {
	if !s.Enabled {
		return false
	}
	if s.Recurrence != nil {
		return s.Recurrence.Contains(now)
	}
	if s.Start.IsZero() || s.End.IsZero() {
		return false
	}
	return !now.Before(s.Start) && now.Before(s.End)
}

// NextStart returns the next activation time at or after from. The second
// return value is false when it cannot be determined (e.g. disabled).
func (s Schedule) NextStart(from time.Time) (time.Time, bool) {
	if !s.Enabled {
		return time.Time{}, false
	}
	if s.Recurrence == nil {
		if s.Start.IsZero() {
			return time.Time{}, false
		}
		return s.Start, true
	}

	start, ok := parseClock(s.Recurrence.StartTime)
	if !ok {
		return time.Time{}, false
	}
	// Search the next 14 days for a matching weekday.
	for day := 0; day < 14; day++ {
		candidateDay := from.AddDate(0, 0, day)
		candidate := time.Date(
			candidateDay.Year(), candidateDay.Month(), candidateDay.Day(),
			start.Hour(), start.Minute(), 0, 0, from.Location(),
		)
		if candidate.Before(from) || !s.Recurrence.matchesWeekday(candidate) {
			continue
		}
		return candidate, true
	}
	return time.Time{}, false
}
