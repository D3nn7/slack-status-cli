// Package scheduler is a pure decision engine. Given the current schedules,
// calendar events and configuration it computes which status should be active
// right now. It performs no I/O, which keeps it easy to test.
package scheduler

import (
	"time"

	"github.com/D3nn7/slack-status-cli/internal/calendar"
	"github.com/D3nn7/slack-status-cli/internal/domain"
)

// Source identifies what drives an override.
type Source string

const (
	SourceMeeting  Source = "meeting"
	SourceSchedule Source = "schedule"
)

// Override is the status that should currently be applied by an automation.
type Override struct {
	Active   bool
	Source   Source
	RefID    string
	Label    string
	Status   domain.Status
	Expires  time.Time
	Priority int
}

// Config holds the automation-related configuration.
type Config struct {
	UseEventTitle bool
	MeetingEmoji  string
	MeetingText   string
	MeetingOnly   bool
	Keywords      []string
}

// Inputs are the dynamic inputs for one evaluation.
type Inputs struct {
	Now       time.Time
	Schedules []domain.Schedule
	Events    []domain.Event
	Config    Config
}

// Evaluate returns the override that should be active at in.Now. Meetings take
// precedence over schedules; within a category the earliest start wins.
func Evaluate(in Inputs) Override {
	if ov, ok := meetingOverride(in); ok {
		return ov
	}
	if ov, ok := scheduleOverride(in); ok {
		return ov
	}
	return Override{}
}

func meetingOverride(in Inputs) (Override, bool) {
	active := calendar.Active(in.Events, in.Now)
	candidates := make([]domain.Event, 0, len(active))
	for _, ev := range active {
		if in.Config.MeetingOnly && !ev.Meeting {
			continue
		}
		candidates = append(candidates, ev)
	}
	if len(candidates) == 0 {
		return Override{}, false
	}

	ev := calendar.Earliest(candidates)
	text := in.Config.MeetingText
	if in.Config.UseEventTitle && ev.Subject != "" {
		text = ev.Subject
	}
	return Override{
		Active: true,
		Source: SourceMeeting,
		RefID:  ev.ID,
		Label:  ev.Subject,
		Status: domain.Status{
			Text:       text,
			Emoji:      in.Config.MeetingEmoji,
			Expiration: ev.End,
		},
		Expires:  ev.End,
		Priority: 100,
	}, true
}

func scheduleOverride(in Inputs) (Override, bool) {
	var chosen *domain.Schedule
	var chosenStart time.Time
	for i := range in.Schedules {
		s := in.Schedules[i]
		if !s.Active(in.Now) {
			continue
		}
		start, ok := s.NextStart(in.Now)
		if !ok {
			continue
		}
		if chosen == nil || start.Before(chosenStart) {
			c := s
			chosen = &c
			chosenStart = start
		}
	}
	if chosen == nil {
		return Override{}, false
	}
	expiry := scheduleExpiration(*chosen, in.Now)
	return Override{
		Active: true,
		Source: SourceSchedule,
		RefID:  chosen.ID,
		Label:  chosen.Label,
		Status: domain.Status{
			Text:       chosen.Text,
			Emoji:      chosen.Emoji,
			Expiration: expiry,
		},
		Expires:  expiry,
		Priority: 50,
	}, true
}

// scheduleExpiration computes when the currently running schedule window ends.
func scheduleExpiration(s domain.Schedule, now time.Time) time.Time {
	if s.Recurrence == nil {
		return s.End
	}
	endClock, ok := domain.ParseClock(s.Recurrence.EndTime)
	if !ok {
		return time.Time{}
	}
	startClock, startOK := domain.ParseClock(s.Recurrence.StartTime)
	end := time.Date(now.Year(), now.Month(), now.Day(),
		endClock.Hour(), endClock.Minute(), 0, 0, now.Location())
	// Overnight spans end on the following day.
	if startOK {
		startMin := startClock.Hour()*60 + startClock.Minute()
		endMin := endClock.Hour()*60 + endClock.Minute()
		if endMin <= startMin && now.Hour()*60+now.Minute() >= startMin {
			end = end.AddDate(0, 0, 1)
		}
	}
	return end
}
