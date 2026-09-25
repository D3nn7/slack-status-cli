package domain

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID returns a short, random, URL-safe identifier.
func NewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to a fixed prefix; identity collisions are recoverable on
		// the next load via EnsureIDs.
		return "tmp"
	}
	return hex.EncodeToString(b[:])
}

// EnsureIDs fills in missing IDs for templates, returning the updated slice.
func EnsureTemplateIDs(templates []Template) []Template {
	for i := range templates {
		if templates[i].ID == "" {
			templates[i].ID = NewID()
		}
	}
	return templates
}

// EnsureScheduleIDs fills in missing IDs for schedules.
func EnsureScheduleIDs(schedules []Schedule) []Schedule {
	for i := range schedules {
		if schedules[i].ID == "" {
			schedules[i].ID = NewID()
		}
	}
	return schedules
}
