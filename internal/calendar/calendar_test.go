package calendar

import (
	"strings"
	"testing"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
)

const sampleICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//EN
BEGIN:VEVENT
UID:event-1
DTSTART:20260115T090000Z
DTEND:20260115T100000Z
SUMMARY:Team Meeting
END:VEVENT
BEGIN:VEVENT
UID:event-2
DTSTART;VALUE=DATE:20260116
DTEND;VALUE=DATE:20260117
SUMMARY:Vacation
END:VEVENT
END:VCALENDAR
`

func TestParseSampleICS(t *testing.T) {
	events, err := Parse(strings.NewReader(sampleICS))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	ev := events[0]
	if ev.ID != "event-1" || ev.Subject != "Team Meeting" {
		t.Errorf("unexpected event: %+v", ev)
	}
	if !ev.Start.Equal(time.Date(2026, time.January, 15, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected start: %v", ev.Start)
	}
	if !events[1].AllDay {
		t.Error("expected second event to be all-day")
	}
}

func TestActiveAndMarkMeetings(t *testing.T) {
	events, err := Parse(strings.NewReader(sampleICS))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	events = MarkMeetings(events, nil)
	if !events[0].Meeting {
		t.Error("expected 'Team Meeting' to be flagged as meeting")
	}
	if events[1].Meeting {
		t.Error("expected 'Vacation' not to be flagged as meeting")
	}

	// 09:30 on event day -> event 1 active.
	now := time.Date(2026, time.January, 15, 9, 30, 0, 0, time.UTC)
	active := Active(events, now)
	if len(active) != 1 || active[0].ID != "event-1" {
		t.Errorf("unexpected active events: %+v", active)
	}
}

func TestIsMeetingSubject(t *testing.T) {
	if !domain.IsMeetingSubject("Daily Standup", nil) {
		t.Error("expected standup to be a meeting")
	}
	if !domain.IsMeetingSubject("Projekt Kickoff", []string{"kickoff"}) {
		t.Error("expected custom keyword to match")
	}
	if domain.IsMeetingSubject("Einkaufsliste", nil) {
		t.Error("expected unrelated subject not to match")
	}
}
