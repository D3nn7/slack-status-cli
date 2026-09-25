package scheduler

import (
	"testing"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
)

func meetingConfig() Config {
	return Config{
		UseEventTitle: true,
		MeetingEmoji:  ":calendar:",
		MeetingText:   "In einem Meeting",
		MeetingOnly:   true,
	}
}

func TestEvaluateMeetingWinsOverSchedule(t *testing.T) {
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)
	ev := domain.Event{
		ID:      "evt",
		Subject: "Team Meeting",
		Start:   now.Add(-5 * time.Minute),
		End:     now.Add(30 * time.Minute),
		Meeting: true,
	}
	sched := domain.Schedule{
		ID:      "sch",
		Label:   "Focus",
		Text:    "Focus time",
		Emoji:   ":dart:",
		Enabled: true,
		Recurrence: &domain.Recurrence{
			StartTime: "09:00",
			EndTime:   "17:00",
		},
	}

	got := Evaluate(Inputs{Now: now, Events: []domain.Event{ev}, Schedules: []domain.Schedule{sched}, Config: meetingConfig()})
	if !got.Active || got.Source != SourceMeeting {
		t.Fatalf("expected meeting override, got %+v", got)
	}
	if got.Status.Text != "Team Meeting" || got.Status.Emoji != ":calendar:" {
		t.Errorf("unexpected status: %+v", got.Status)
	}
	if want := ev.End; !got.Status.Expiration.Equal(want) {
		t.Errorf("expiration got %v, want %v", got.Status.Expiration, want)
	}
}

func TestEvaluateNonMeetingIgnoredWhenMeetingOnly(t *testing.T) {
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)
	ev := domain.Event{
		ID:      "evt",
		Subject: "Mittagspause",
		Start:   now.Add(-time.Minute),
		End:     now.Add(time.Hour),
		Meeting: false,
	}
	got := Evaluate(Inputs{Now: now, Events: []domain.Event{ev}, Config: meetingConfig()})
	if got.Active {
		t.Fatalf("expected no override, got %+v", got)
	}
}

func TestEvaluateScheduleWhenNoMeeting(t *testing.T) {
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)
	sched := domain.Schedule{
		ID:      "sch",
		Label:   "Focus",
		Text:    "Focus time",
		Emoji:   ":dart:",
		Enabled: true,
		Recurrence: &domain.Recurrence{
			StartTime: "09:00",
			EndTime:   "17:00",
		},
	}
	got := Evaluate(Inputs{Now: now, Schedules: []domain.Schedule{sched}, Config: meetingConfig()})
	if !got.Active || got.Source != SourceSchedule {
		t.Fatalf("expected schedule override, got %+v", got)
	}
	if !got.Expires.Equal(time.Date(2026, time.January, 15, 17, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected expiry: %v", got.Expires)
	}
}
