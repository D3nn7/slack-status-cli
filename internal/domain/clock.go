package domain

import "time"

// clockLayouts are the accepted "HH:MM" clock formats for templates.
var clockLayouts = []string{"15:04", "15:04:05"}

// ParseClock parses an "HH:MM" (or "HH:MM:SS") clock string.
func ParseClock(v string) (time.Time, bool) {
	return parseClock(v)
}

func parseClock(v string) (time.Time, bool) {
	for _, layout := range clockLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
