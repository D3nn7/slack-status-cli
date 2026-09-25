package domain

import (
	"testing"
	"time"
)

func TestRecurrenceContains(t *testing.T) {
	rec := Recurrence{
		Weekdays:  []time.Weekday{time.Monday, time.Friday},
		StartTime: "09:00",
		EndTime:   "17:00",
	}
	// Monday 2026-01-12, 10:00 -> inside.
	monday := time.Date(2026, time.January, 12, 10, 0, 0, 0, time.UTC)
	if !rec.Contains(monday) {
		t.Error("expected Monday 10:00 to be inside")
	}
	// Tuesday not selected.
	tuesday := time.Date(2026, time.January, 13, 10, 0, 0, 0, time.UTC)
	if rec.Contains(tuesday) {
		t.Error("expected Tuesday to be outside")
	}
	// Monday before start.
	early := time.Date(2026, time.January, 12, 8, 0, 0, 0, time.UTC)
	if rec.Contains(early) {
		t.Error("expected Monday 08:00 to be outside")
	}
}

func TestRecurrenceOvernight(t *testing.T) {
	rec := Recurrence{StartTime: "22:00", EndTime: "06:00"}
	late := time.Date(2026, time.January, 12, 23, 30, 0, 0, time.UTC)
	early := time.Date(2026, time.January, 12, 5, 0, 0, 0, time.UTC)
	noon := time.Date(2026, time.January, 12, 12, 0, 0, 0, time.UTC)
	if !rec.Contains(late) {
		t.Error("expected 23:30 inside overnight window")
	}
	if !rec.Contains(early) {
		t.Error("expected 05:00 inside overnight window")
	}
	if rec.Contains(noon) {
		t.Error("expected noon outside overnight window")
	}
}

func TestScheduleActiveAndNextStart(t *testing.T) {
	start := time.Date(2026, time.February, 1, 9, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	s := Schedule{ID: "1", Enabled: true, Start: start, End: end}

	if s.Active(start.Add(-time.Minute)) {
		t.Error("schedule should not be active before start")
	}
	if !s.Active(start.Add(time.Minute)) {
		t.Error("schedule should be active inside window")
	}
	if s.Active(end) {
		t.Error("schedule should not be active at end (exclusive)")
	}

	disabled := s
	disabled.Enabled = false
	if disabled.Active(start.Add(time.Minute)) {
		t.Error("disabled schedule must not be active")
	}
}

func TestScheduleNextStartRecurring(t *testing.T) {
	// Friday 2026-01-16, looking for next Monday 09:00.
	friday := time.Date(2026, time.January, 16, 12, 0, 0, 0, time.UTC)
	s := Schedule{
		Enabled: true,
		Recurrence: &Recurrence{
			Weekdays:  []time.Weekday{time.Monday},
			StartTime: "09:00",
			EndTime:   "17:00",
		},
	}
	next, ok := s.NextStart(friday)
	if !ok {
		t.Fatal("expected a next start")
	}
	want := time.Date(2026, time.January, 19, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}
