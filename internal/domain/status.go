// Package domain contains the core, UI- and transport-agnostic types of the
// application: statuses, templates, schedules, recurrences and events.
package domain

import "time"

// Status is a Slack custom status with an optional expiry. A zero Expiration
// means the status never expires on its own.
type Status struct {
	Text       string    `json:"text"`
	Emoji      string    `json:"emoji"`
	Expiration time.Time `json:"expiration,omitzero"`
}

// Expired reports whether the status has an expiry that has already passed.
func (s Status) Expired(now time.Time) bool {
	return !s.Expiration.IsZero() && !now.Before(s.Expiration)
}

// IsZero reports whether the status is empty (no text and no emoji).
func (s Status) IsZero() bool {
	return s.Text == "" && s.Emoji == ""
}
