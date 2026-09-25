package domain

import (
	"strings"
	"time"
)

// Event is a calendar event relevant to temporary status overriding.
type Event struct {
	ID      string
	Subject string
	Start   time.Time
	End     time.Time
	AllDay  bool
	Meeting bool
}

// defaultMeetingKeywords are matched case-insensitively against the subject to
// decide whether an event counts as a meeting worth overriding the status for.
var defaultMeetingKeywords = []string{
	"meeting", "call", "huddle", "standup", "stand-up", "sync",
	"review", "1:1", "one-on-one", "interview", "kickoff", "retro",
	"daily", "weekly", "teams", "zoom", "webex", "google meet",
}

// IsMeetingSubject reports whether a subject looks like a meeting. Additional
// keywords extend the built-in list.
func IsMeetingSubject(subject string, extra []string) bool {
	lower := strings.ToLower(subject)
	if strings.TrimSpace(lower) == "" {
		return false
	}
	for _, kw := range defaultMeetingKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	for _, kw := range extra {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw != "" && strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
