package domain

import "time"

// Template is a reusable status preset. The optional ID provides a stable
// identity independent of the (editable) label; legacy templates without an
// ID are upgraded on load.
type Template struct {
	ID                  string `json:"id,omitempty"`
	Label               string `json:"label"`
	Text                string `json:"text"`
	Emoji               string `json:"emoji"`
	DurationInMinutes   *int   `json:"durationInMinutes,omitempty"`
	UntilTime           string `json:"untilTime,omitempty"`
	UseDurationSelector bool   `json:"useDurationSelector,omitempty"`
}

// Status materialises the template into a concrete status. now anchors
// relative durations; customDuration overrides DurationInMinutes when the
// template opts into the duration selector.
func (t Template) Status(now time.Time, customDuration *time.Duration) Status {
	s := Status{Text: t.Text, Emoji: t.Emoji}

	if customDuration != nil && *customDuration > 0 {
		s.Expiration = now.Add(*customDuration)
		return s
	}
	if t.DurationInMinutes != nil && *t.DurationInMinutes > 0 {
		s.Expiration = now.Add(time.Duration(*t.DurationInMinutes) * time.Minute)
		return s
	}
	if until, ok := parseClock(t.UntilTime); ok {
		target := time.Date(now.Year(), now.Month(), now.Day(), until.Hour(), until.Minute(), 0, 0, now.Location())
		if !target.After(now) {
			target = target.AddDate(0, 0, 1)
		}
		s.Expiration = target
	}
	return s
}
